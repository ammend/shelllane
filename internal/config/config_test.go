package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validConfig = `version: 1
shells:
  sh: /bin/sh
identity:
  env: OWNER
state:
  root: STATE_ROOT
routes:
  - target: default
nodes:
  local:
    connector: local
runtimes:
  host:
    type: host
targets:
  default:
    node: local
    runtime: host
`

func TestLoadValidAndDefaults(t *testing.T) {
	path := writeConfig(t, strings.Replace(validConfig, "STATE_ROOT", filepath.Join(t.TempDir(), "state"), 1))
	cfg, resolved, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if resolved != path || cfg.Identity.Hash != "sha256" || cfg.Nodes["local"].Port != 22 {
		t.Fatalf("defaults not applied: %#v", cfg)
	}
}

func TestLoadRejectsUnknownField(t *testing.T) {
	content := strings.Replace(validConfig, "STATE_ROOT", filepath.Join(t.TempDir(), "state"), 1)
	content += "surprise: true\n"
	path := writeConfig(t, content)
	if _, _, err := Load(path); err == nil || !strings.Contains(err.Error(), "field surprise not found") {
		t.Fatalf("expected strict YAML error, got %v", err)
	}
}

func TestValidationRejectsDanglingTarget(t *testing.T) {
	content := strings.Replace(validConfig, "STATE_ROOT", filepath.Join(t.TempDir(), "state"), 1)
	content = strings.Replace(content, "target: default", "target: missing", 1)
	path := writeConfig(t, content)
	if _, _, err := Load(path); err == nil || !strings.Contains(err.Error(), "unknown target") {
		t.Fatalf("expected target error, got %v", err)
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
