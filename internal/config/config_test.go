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
	if resolved != path || cfg.Identity.Hash != "sha256" || cfg.State.IdentityNaming != "plain" || cfg.Nodes["local"].Port != 22 {
		t.Fatalf("defaults not applied: %#v", cfg)
	}
}

func TestLoadUsesXDGConfigHome(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("SHELLLANE_CONFIG", "")
	path := filepath.Join(configHome, "shelllane", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	content := strings.Replace(validConfig, "STATE_ROOT", filepath.Join(t.TempDir(), "state"), 1)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	_, resolved, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if resolved != path {
		t.Fatalf("resolved %q, want XDG path %q", resolved, path)
	}
}

func TestValidateRejectsIdentityEnvMapExpression(t *testing.T) {
	required := true
	cfg := Config{
		Version: 1,
		Identity: IdentityConfig{
			Env:      "SHELLLANE_IDENTITY",
			EnvMap:   map[string]StringList{"SHELLLANE_IDENTITY": {"$(id)"}},
			Required: &required,
			Hash:     "sha256",
		},
		State:  StateConfig{Root: "/tmp/shelllane"},
		Routes: []Route{{Target: "local"}},
		Nodes:  map[string]Node{"local": {Connector: "local"}},
		Runtimes: map[string]Runtime{
			"host": {Type: "host"},
		},
		Targets: map[string]Target{
			"local": {Node: "local", Runtime: "host"},
		},
	}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "must be $NAME or ${NAME}") {
		t.Fatalf("expected invalid envMap error, got %v", err)
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
