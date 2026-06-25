package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/microsoft/kuafu/internal/api"
	"github.com/microsoft/kuafu/internal/domain"
	"github.com/microsoft/kuafu/internal/repository"
	"github.com/microsoft/kuafu/internal/scheduler"
	"github.com/microsoft/kuafu/pkg/fixtures"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP server address")
	seedData := flag.Bool("seed", true, "Seed with testbed A00/A01 data")
	flag.Parse()

	// Create repository
	repo := repository.NewMemoryRepository()

	// Seed with testbed data if requested
	if *seedData {
		log.Println("Seeding repository with testbed A00/A01 data...")
		if err := fixtures.SeedTestbedA00A01(repo.AddNode, repo.AddGPU, repo.AddQueue, repo.AddJob); err != nil {
			log.Fatalf("Failed to seed data: %v", err)
		}
		log.Println("Seeding complete")
	}

	// Initialize default queues (only if not seeded)
	if !*seedData {
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
	server := api.NewServer(repo, sched, *addr)

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
	if err := server.Start(); err != nil && err != context.Canceled {
		log.Fatalf("Server error: %v", err)
	}

	<-done
	log.Println("Server stopped")
}
