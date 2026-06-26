package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/microsoft/kuafu/internal/api"
	"github.com/microsoft/kuafu/internal/auth"
	"github.com/microsoft/kuafu/internal/config"
	"github.com/microsoft/kuafu/internal/domain"
	"github.com/microsoft/kuafu/internal/repository"
	"github.com/microsoft/kuafu/internal/scheduler"
	"github.com/microsoft/kuafu/pkg/fixtures"
)

func main() {
	cfg := loadConfig()

	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid Kuafu config: %v", err)
	}

	if err := cfg.ValidateProductionConnectivity(context.Background()); err != nil {
		log.Fatalf("Kuafu production dependency validation failed: %v", err)
	}

	if cfg.Runtime.Mode == config.ModeProduction {
		log.Fatalf("production configuration is valid, but production runtime is not wired yet: Kubernetes client, GPU device plugin, scheduler adapter, and persistent repository are required by steps 3-10. Use --mode lab only for the in-memory prototype.")
	}
	log.Println("Starting Kuafu in LAB MODE: in-memory state and simulated scheduling are active")

	// Create repository
	repo := repository.NewMemoryRepository()

	// Seed with testbed data if requested
	if cfg.Lab.SeedData {
		log.Println("Seeding repository with testbed A00/A01 data...")
		if err := fixtures.SeedTestbedA00A01(repo.AddNode, repo.AddGPU, repo.AddQueue, repo.AddJob); err != nil {
			log.Fatalf("Failed to seed data: %v", err)
		}
		if err := fixtures.SeedLabTenants(repo.AddProject, repo.AddUser); err != nil {
			log.Fatalf("Failed to seed lab tenants: %v", err)
		}
		log.Println("Seeding complete")
	}

	// Initialize default queues (only if not seeded)
	if !cfg.Lab.SeedData {
		log.Println("Initializing queues...")
		queues := []*domain.Queue{
			{Name: "default", MaxGPUs: 8, Priority: 100},
			{Name: "high", MaxGPUs: 16, Priority: 200},
			{Name: "batch", MaxGPUs: 4, Priority: 50},
		}
		for _, q := range queues {
			if err := repo.AddQueue(q); err != nil {
				log.Fatalf("Failed to add queue %s: %v", q.Name, err)
			}
		}
		log.Println("Queues initialized")
	}

	// Create and start scheduler
	sched := scheduler.NewScheduler(repo)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sched.Start(ctx)
	log.Println("Scheduler started")

	// Create and start API server
	server := api.NewServer(repo, sched, cfg.Server.Addr)
	server.SetAuthenticator(auth.HeaderAuthenticator{Users: repo})

	// Handle graceful shutdown
	done := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		log.Println("Shutting down server...")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}

		// Stop scheduler
		cancel()
		sched.Stop()
		log.Println("Scheduler stopped")

		close(done)
	}()

	// Start server
	if err := server.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}

	<-done
	log.Println("Server stopped")
}

func loadConfig() config.Config {
	configPath := flag.String("config", "", "Path to Kuafu JSON config file")
	addr := flag.String("addr", "", "HTTP server address")
	mode := flag.String("mode", "", "Runtime mode: production or lab")
	seedData := flag.Bool("seed", true, "Seed with testbed A00/A01 data in lab mode")
	kubeConfigPath := flag.String("kubeconfig", "", "Path to Kubernetes kubeconfig for production mode")
	kubernetesAPIServer := flag.String("kube-api-server", "", "Kubernetes API server URL for production mode")
	mongoDBURI := flag.String("mongodb-uri", "", "MongoDB URI for production mode")
	schedulerBackend := flag.String("scheduler-backend", "", "Scheduler backend: frameworkcontroller, hived, or volcano")
	authProvider := flag.String("auth-provider", "", "Authentication provider for production mode")
	checkConnectivity := flag.Bool("check-connectivity", false, "Validate TCP connectivity to production dependencies before startup")
	flag.Parse()

	overrides := config.Overrides{}
	flag.Visit(func(flagValue *flag.Flag) {
		switch flagValue.Name {
		case "addr":
			overrides.Addr = addr
		case "mode":
			overrides.Mode = mode
		case "seed":
			overrides.SeedData = seedData
		case "kubeconfig":
			overrides.KubeConfigPath = kubeConfigPath
		case "kube-api-server":
			overrides.KubernetesAPIServer = kubernetesAPIServer
		case "mongodb-uri":
			overrides.MongoDBURI = mongoDBURI
		case "scheduler-backend":
			overrides.SchedulerBackend = schedulerBackend
		case "auth-provider":
			overrides.AuthProvider = authProvider
		case "check-connectivity":
			overrides.CheckConnectivity = checkConnectivity
		}
	})

	cfg, err := config.Load(*configPath, overrides, os.LookupEnv)
	if err != nil {
		log.Fatalf("failed to load Kuafu config: %v", err)
	}
	return cfg
}
