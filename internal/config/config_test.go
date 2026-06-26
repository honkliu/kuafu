package config

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultProductionConfigRequiresDependencies(t *testing.T) {
	cfg := Default()
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected production config validation errors")
	}
	message := err.Error()
	for _, expected := range []string{"kubernetes", "mongoDB", "scheduler", "auth"} {
		if !strings.Contains(message, expected) {
			t.Fatalf("expected validation error to mention %s, got %q", expected, message)
		}
	}
}

func TestLabConfigValidatesWithoutProductionDependencies(t *testing.T) {
	cfg := Default()
	cfg.Runtime.Mode = ModeLab
	if err := cfg.Validate(); err != nil {
		t.Fatalf("lab config should validate without production dependencies: %v", err)
	}
}

func TestLoadAppliesDefaultsFileEnvAndOverrides(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "kuafu.json")
	content := `{
  "server": {"addr": ":9000"},
  "runtime": {"mode": "lab"},
  "lab": {"seedData": false},
  "production": {
    "kubernetes": {"apiServerURL": "https://file-k8s.example"},
    "mongoDB": {"uri": "mongodb://file-mongo:27017/kuafu"},
    "scheduler": {"backend": "hived"},
    "auth": {"provider": "oidc"}
  }
}`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	mode := ModeProduction
	mongoURI := "mongodb://override-mongo:27017/kuafu"
	cfg, err := Load(configPath, Overrides{Mode: &mode, MongoDBURI: &mongoURI}, mapLookup(map[string]string{
		"KUAFU_ADDR":              ":9100",
		"KUAFU_SCHEDULER_BACKEND": SchedulerVolcano,
	}))
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Server.Addr != ":9100" {
		t.Fatalf("expected env addr to win over file, got %s", cfg.Server.Addr)
	}
	if cfg.Runtime.Mode != ModeProduction {
		t.Fatalf("expected override mode to win, got %s", cfg.Runtime.Mode)
	}
	if cfg.Production.MongoDB.URI != mongoURI {
		t.Fatalf("expected override mongo uri to win, got %s", cfg.Production.MongoDB.URI)
	}
	if cfg.Production.Scheduler.Backend != SchedulerVolcano {
		t.Fatalf("expected env scheduler backend, got %s", cfg.Production.Scheduler.Backend)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected loaded production config to validate: %v", err)
	}
}

func TestLoadRejectsInvalidBooleanEnv(t *testing.T) {
	_, err := Load("", Overrides{}, mapLookup(map[string]string{"KUAFU_LAB_SEED_DATA": "sometimes"}))
	if err == nil {
		t.Fatal("expected invalid boolean env to fail")
	}
}

func TestProductionValidationRejectsUnsupportedScheduler(t *testing.T) {
	cfg := Default()
	cfg.Production.Kubernetes.APIServerURL = "https://k8s.example"
	cfg.Production.MongoDB.URI = "mongodb://mongo:27017/kuafu"
	cfg.Production.Scheduler.Backend = "fifo"
	cfg.Production.Auth.Provider = "oidc"

	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "frameworkcontroller") {
		t.Fatalf("expected scheduler validation error, got %v", err)
	}
}

func TestProductionValidationRejectsNonMongoScheme(t *testing.T) {
	cfg := Default()
	cfg.Production.Kubernetes.APIServerURL = "https://k8s.example"
	cfg.Production.MongoDB.URI = "https://mongo.example/kuafu"
	cfg.Production.Scheduler.Backend = SchedulerFrameworkController
	cfg.Production.Auth.Provider = "oidc"

	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "MongoDB URI") {
		t.Fatalf("expected MongoDB URI validation error, got %v", err)
	}
}

func TestProductionConnectivityRejectsMongoSRV(t *testing.T) {
	cfg := Default()
	cfg.Production.Kubernetes.APIServerURL = "https://k8s.example"
	cfg.Production.MongoDB.URI = "mongodb+srv://mongo.example/kuafu"
	cfg.Production.Scheduler.Backend = SchedulerFrameworkController
	cfg.Production.Auth.Provider = "oidc"
	cfg.Production.Validation.CheckConnectivity = true

	err := cfg.ValidateProductionConnectivity(context.Background())
	if err == nil || !strings.Contains(err.Error(), "mongodb+srv connectivity validation is not supported") {
		t.Fatalf("expected mongodb+srv connectivity validation error, got %v", err)
	}
}

func mapLookup(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}
