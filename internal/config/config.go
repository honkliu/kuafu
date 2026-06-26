package config

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	ModeLab        = "lab"
	ModeProduction = "production"

	SchedulerFrameworkController = "frameworkcontroller"
	SchedulerHiveD               = "hived"
	SchedulerVolcano             = "volcano"
)

type Config struct {
	Server     ServerConfig     `json:"server"`
	Runtime    RuntimeConfig    `json:"runtime"`
	Lab        LabConfig        `json:"lab"`
	Production ProductionConfig `json:"production"`
}

type ServerConfig struct {
	Addr string `json:"addr"`
}

type RuntimeConfig struct {
	Mode string `json:"mode"`
}

type LabConfig struct {
	SeedData bool `json:"seedData"`
}

type ProductionConfig struct {
	Kubernetes KubernetesConfig `json:"kubernetes"`
	MongoDB    MongoDBConfig    `json:"mongoDB"`
	Scheduler  SchedulerConfig  `json:"scheduler"`
	Auth       AuthConfig       `json:"auth"`
	Validation ValidationConfig `json:"validation"`
}

type KubernetesConfig struct {
	KubeConfigPath string `json:"kubeConfigPath"`
	APIServerURL   string `json:"apiServerURL"`
}

type MongoDBConfig struct {
	URI string `json:"uri"`
}

type SchedulerConfig struct {
	Backend string `json:"backend"`
}

type AuthConfig struct {
	Provider string `json:"provider"`
}

type ValidationConfig struct {
	CheckConnectivity bool `json:"checkConnectivity"`
}

type Overrides struct {
	Addr                *string
	Mode                *string
	SeedData            *bool
	KubeConfigPath      *string
	KubernetesAPIServer *string
	MongoDBURI          *string
	SchedulerBackend    *string
	AuthProvider        *string
	CheckConnectivity   *bool
}

type ValidationError struct {
	Problems []string
}

func (e *ValidationError) Error() string {
	return strings.Join(e.Problems, "; ")
}

func Default() Config {
	return Config{
		Server: ServerConfig{
			Addr: ":8080",
		},
		Runtime: RuntimeConfig{
			Mode: ModeProduction,
		},
		Lab: LabConfig{
			SeedData: true,
		},
	}
}

func Load(path string, overrides Overrides, lookupEnv func(string) (string, bool)) (Config, error) {
	cfg := Default()

	if path != "" {
		if err := loadFile(path, &cfg); err != nil {
			return Config{}, err
		}
	}

	if lookupEnv == nil {
		lookupEnv = os.LookupEnv
	}
	if err := applyEnv(&cfg, lookupEnv); err != nil {
		return Config{}, err
	}

	applyOverrides(&cfg, overrides)
	return cfg, nil
}

func (c Config) Validate() error {
	problems := make([]string, 0)

	if c.Server.Addr == "" {
		problems = append(problems, "server.addr is required")
	}

	switch c.Runtime.Mode {
	case ModeLab:
		return validationResult(problems)
	case ModeProduction:
		problems = append(problems, validateProduction(c.Production)...)
	default:
		problems = append(problems, fmt.Sprintf("runtime.mode must be %q or %q", ModeLab, ModeProduction))
	}

	return validationResult(problems)
}

func (c Config) ValidateProductionConnectivity(ctx context.Context) error {
	if c.Runtime.Mode != ModeProduction || !c.Production.Validation.CheckConnectivity {
		return nil
	}

	problems := make([]string, 0)
	if c.Production.Kubernetes.KubeConfigPath != "" {
		if _, err := os.Stat(c.Production.Kubernetes.KubeConfigPath); err != nil {
			problems = append(problems, fmt.Sprintf("kubernetes.kubeConfigPath is not readable: %v", err))
		}
	}
	if c.Production.Kubernetes.APIServerURL != "" {
		if err := dialURL(ctx, c.Production.Kubernetes.APIServerURL, "443"); err != nil {
			problems = append(problems, fmt.Sprintf("kubernetes.apiServerURL is unreachable: %v", err))
		}
	}
	if c.Production.MongoDB.URI != "" {
		if err := dialMongoURI(ctx, c.Production.MongoDB.URI); err != nil {
			problems = append(problems, fmt.Sprintf("mongoDB.uri is unreachable: %v", err))
		}
	}

	return validationResult(problems)
}

func loadFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file %s: %w", path, err)
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("parse config file %s: %w", path, err)
	}
	return nil
}

func applyEnv(cfg *Config, lookupEnv func(string) (string, bool)) error {
	if value, ok := lookupEnv("KUAFU_ADDR"); ok {
		cfg.Server.Addr = value
	}
	if value, ok := lookupEnv("KUAFU_MODE"); ok {
		cfg.Runtime.Mode = value
	}
	if value, ok := lookupEnv("KUAFU_LAB_SEED_DATA"); ok {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("parse KUAFU_LAB_SEED_DATA: %w", err)
		}
		cfg.Lab.SeedData = parsed
	}
	if value, ok := lookupEnv("KUAFU_KUBECONFIG"); ok {
		cfg.Production.Kubernetes.KubeConfigPath = value
	}
	if value, ok := lookupEnv("KUAFU_KUBERNETES_API_SERVER"); ok {
		cfg.Production.Kubernetes.APIServerURL = value
	}
	if value, ok := lookupEnv("KUAFU_MONGODB_URI"); ok {
		cfg.Production.MongoDB.URI = value
	}
	if value, ok := lookupEnv("KUAFU_SCHEDULER_BACKEND"); ok {
		cfg.Production.Scheduler.Backend = value
	}
	if value, ok := lookupEnv("KUAFU_AUTH_PROVIDER"); ok {
		cfg.Production.Auth.Provider = value
	}
	if value, ok := lookupEnv("KUAFU_VALIDATE_CONNECTIVITY"); ok {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("parse KUAFU_VALIDATE_CONNECTIVITY: %w", err)
		}
		cfg.Production.Validation.CheckConnectivity = parsed
	}
	return nil
}

func applyOverrides(cfg *Config, overrides Overrides) {
	if overrides.Addr != nil {
		cfg.Server.Addr = *overrides.Addr
	}
	if overrides.Mode != nil {
		cfg.Runtime.Mode = *overrides.Mode
	}
	if overrides.SeedData != nil {
		cfg.Lab.SeedData = *overrides.SeedData
	}
	if overrides.KubeConfigPath != nil {
		cfg.Production.Kubernetes.KubeConfigPath = *overrides.KubeConfigPath
	}
	if overrides.KubernetesAPIServer != nil {
		cfg.Production.Kubernetes.APIServerURL = *overrides.KubernetesAPIServer
	}
	if overrides.MongoDBURI != nil {
		cfg.Production.MongoDB.URI = *overrides.MongoDBURI
	}
	if overrides.SchedulerBackend != nil {
		cfg.Production.Scheduler.Backend = *overrides.SchedulerBackend
	}
	if overrides.AuthProvider != nil {
		cfg.Production.Auth.Provider = *overrides.AuthProvider
	}
	if overrides.CheckConnectivity != nil {
		cfg.Production.Validation.CheckConnectivity = *overrides.CheckConnectivity
	}
}

func validateProduction(cfg ProductionConfig) []string {
	problems := make([]string, 0)
	if cfg.Kubernetes.KubeConfigPath == "" && cfg.Kubernetes.APIServerURL == "" {
		problems = append(problems, "production.kubernetes.kubeConfigPath or production.kubernetes.apiServerURL is required")
	}
	if cfg.Kubernetes.APIServerURL != "" {
		if _, err := url.ParseRequestURI(cfg.Kubernetes.APIServerURL); err != nil {
			problems = append(problems, fmt.Sprintf("production.kubernetes.apiServerURL is invalid: %v", err))
		}
	}
	if cfg.MongoDB.URI == "" {
		problems = append(problems, "production.mongoDB.uri is required")
	} else if parsed, err := url.Parse(cfg.MongoDB.URI); err != nil || parsed.Host == "" || !isSupportedMongoScheme(parsed.Scheme) {
		problems = append(problems, "production.mongoDB.uri must be a valid MongoDB URI")
	}
	if cfg.Scheduler.Backend == "" {
		problems = append(problems, "production.scheduler.backend is required")
	} else if !isSupportedScheduler(cfg.Scheduler.Backend) {
		problems = append(problems, fmt.Sprintf("production.scheduler.backend must be one of %s, %s, %s", SchedulerFrameworkController, SchedulerHiveD, SchedulerVolcano))
	}
	if cfg.Auth.Provider == "" {
		problems = append(problems, "production.auth.provider is required")
	}
	return problems
}

func isSupportedScheduler(value string) bool {
	switch value {
	case SchedulerFrameworkController, SchedulerHiveD, SchedulerVolcano:
		return true
	default:
		return false
	}
}

func isSupportedMongoScheme(value string) bool {
	switch value {
	case "mongodb", "mongodb+srv":
		return true
	default:
		return false
	}
}

func validationResult(problems []string) error {
	if len(problems) == 0 {
		return nil
	}
	return &ValidationError{Problems: problems}
}

func dialURL(ctx context.Context, rawURL string, defaultPort string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	host := parsed.Hostname()
	port := parsed.Port()
	if port == "" {
		port = defaultPort
	}
	return dialHostPort(ctx, net.JoinHostPort(host, port))
}

func dialMongoURI(ctx context.Context, rawURI string) error {
	parsed, err := url.Parse(rawURI)
	if err != nil {
		return err
	}
	if parsed.Scheme == "mongodb+srv" {
		return fmt.Errorf("mongodb+srv connectivity validation is not supported without MongoDB driver SRV resolution")
	}
	for _, host := range strings.Split(parsed.Host, ",") {
		if host == "" {
			continue
		}
		if _, _, err := net.SplitHostPort(host); err != nil {
			host = net.JoinHostPort(host, "27017")
		}
		if err := dialHostPort(ctx, host); err != nil {
			return err
		}
	}
	return nil
}

func dialHostPort(ctx context.Context, address string) error {
	dialer := net.Dialer{Timeout: 2 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return err
	}
	return conn.Close()
}
