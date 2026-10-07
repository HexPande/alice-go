package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HexPande/alice-go/internal/config"
)

func TestLoad(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		yaml string
		port int
	}{
		{"custom", "port: 9000\n", 9000},
		{"default", "{}\n", 8090},
		{"comments", "# My settings\nport: 8091 # HTTP\n", 8091},
		{"zero", "port: 0\n", 0},
		{"negative", "port: -1\n", 0},
		{"too large", "port: 65536\n", 0},
		{"text", "port: hello\n", 0},
		{"fraction", "port: 8090.5\n", 0},
		{"malformed", "port: [\n", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(path, []byte(tt.yaml), 0600); err != nil {
				t.Fatal(err)
			}
			got, err := config.Load(path)
			if tt.port == 0 {
				if err == nil {
					t.Fatal("expected invalid configuration to be rejected")
				}
				return
			}
			if err != nil || got.Port != tt.port {
				t.Fatalf("Load() = %+v, %v; want port %d", got, err, tt.port)
			}
		})
	}
}

func TestLoadWithoutFile(t *testing.T) {
	t.Parallel()
	cfg, err := config.Load("")
	if err != nil || cfg.Port != 8090 {
		t.Fatalf("unexpected defaults: %+v, %v", cfg, err)
	}
	if _, err := config.Load(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Fatal("explicitly selected missing file must fail")
	}
}
