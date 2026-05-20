package configs

import (
	"testing"

	"github.com/spf13/viper"
)

func TestValidateConfigRequiresNodeID(t *testing.T) {
	err := validateConfig(Config{})
	if err == nil {
		t.Fatal("validateConfig() error is nil, want NODE_ID required error")
	}
}

func TestValidateConfigAcceptsNodeID(t *testing.T) {
	err := validateConfig(Config{NodeID: "node-a"})
	if err != nil {
		t.Fatalf("validateConfig() error = %v", err)
	}
}

func TestBindEnvLoadsNodeID(t *testing.T) {
	t.Setenv("NODE_ID", "node-a")

	v := viper.New()
	if err := bindEnv(v); err != nil {
		t.Fatalf("bindEnv() error = %v", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if cfg.NodeID != "node-a" {
		t.Fatalf("NodeID = %q, want node-a", cfg.NodeID)
	}
}
