package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCloneCreatesDeepCopy(t *testing.T) {
	t.Parallel()

	original := &Config{
		Server: ServerConfig{AllowedOrigins: []string{"https://a.example.com"}},
		Channels: []ChannelConfig{
			{
				Type:    "webhook",
				Enabled: true,
				To:      []string{"a@example.com"},
				Headers: map[string]string{"X-Test": "old"},
			},
		},
	}

	clone := original.Clone()
	clone.Server.AllowedOrigins[0] = "https://b.example.com"
	clone.Channels[0].To[0] = "b@example.com"
	clone.Channels[0].Headers["X-Test"] = "new"

	if got := original.Server.AllowedOrigins[0]; got != "https://a.example.com" {
		t.Fatalf("AllowedOrigins was mutated: %q", got)
	}
	if got := original.Channels[0].To[0]; got != "a@example.com" {
		t.Fatalf("channel recipients were mutated: %q", got)
	}
	if got := original.Channels[0].Headers["X-Test"]; got != "old" {
		t.Fatalf("channel headers were mutated: %q", got)
	}
}

func TestLoadAppliesDefaultsWithoutRewritingFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.toml")
	original := "# keep this comment\n[server]\nenabled = true\n"
	if err := os.WriteFile(path, []byte(original), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Server.Listen != ":8080" {
		t.Fatalf("default listen not applied: %q", cfg.Server.Listen)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(data) != original {
		t.Fatalf("Load should not rewrite config file; got:\n%s", data)
	}
}

func TestSaveUsesPrivatePermissionsForSensitiveConfig(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("[server]\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	cfg := &Config{Server: ServerConfig{Enabled: true, Listen: ":8080"}}
	if err := cfg.Save(path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("Save should use 0600 for world/group-readable configs, got %o", got)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !strings.Contains(string(data), "listen = \":8080\"") {
		t.Fatalf("saved config missing listen value:\n%s", data)
	}
}
