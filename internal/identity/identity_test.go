package identity

import (
	"errors"
	"testing"

	"github.com/ammend/shelllane/internal/config"
)

func TestResolveFromEnvironment(t *testing.T) {
	required := true
	cfg := config.IdentityConfig{Env: "OWNER", Required: &required, ArgvEnvFallback: true}
	id, err := Resolve(cfg, []string{"bash"}, func(key string) string { return "alice" })
	if err != nil {
		t.Fatal(err)
	}
	if id.Value != "alice" || len(id.Hash) != 64 || id.Source != "env:OWNER" {
		t.Fatalf("unexpected identity: %#v", id)
	}
}

func TestResolveArgvAndConflict(t *testing.T) {
	required := true
	cfg := config.IdentityConfig{Env: "OWNER", Required: &required, ArgvEnvFallback: true}
	id, err := Resolve(cfg, []string{"bash", "OWNER=bob", "-lc", "true"}, func(string) string { return "" })
	if err != nil || id.Value != "bob" {
		t.Fatalf("unexpected result: %#v %v", id, err)
	}
	_, err = Resolve(cfg, []string{"bash", "OWNER=bob"}, func(string) string { return "alice" })
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestResolveMissing(t *testing.T) {
	required := true
	cfg := config.IdentityConfig{Env: "OWNER", Required: &required}
	_, err := Resolve(cfg, []string{"bash"}, func(string) string { return "" })
	if !errors.Is(err, ErrMissing) {
		t.Fatalf("expected missing identity, got %v", err)
	}
}

func TestResolveFromMappedEnvironment(t *testing.T) {
	required := true
	cfg := config.IdentityConfig{
		Env:      "SHELLLANE_IDENTITY",
		EnvMap:   map[string]string{"SHELLLANE_IDENTITY": "$BOTMUX_USER_ID"},
		Required: &required,
	}
	values := map[string]string{"BOTMUX_USER_ID": "ou_alice"}
	id, err := Resolve(cfg, []string{"bash"}, func(key string) string { return values[key] })
	if err != nil {
		t.Fatal(err)
	}
	if id.Value != "ou_alice" || id.Source != "env:BOTMUX_USER_ID" {
		t.Fatalf("unexpected identity: %#v", id)
	}
}

func TestResolveMappedEnvironmentConflict(t *testing.T) {
	required := true
	cfg := config.IdentityConfig{
		Env:      "SHELLLANE_IDENTITY",
		EnvMap:   map[string]string{"SHELLLANE_IDENTITY": "${BOTMUX_USER_ID}"},
		Required: &required,
	}
	values := map[string]string{
		"SHELLLANE_IDENTITY": "alice",
		"BOTMUX_USER_ID":     "bob",
	}
	_, err := Resolve(cfg, []string{"bash"}, func(key string) string { return values[key] })
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestResolveMappedArgv(t *testing.T) {
	required := true
	cfg := config.IdentityConfig{
		Env:             "SHELLLANE_IDENTITY",
		EnvMap:          map[string]string{"SHELLLANE_IDENTITY": "$BOTMUX_USER_ID"},
		ArgvEnvFallback: true,
		Required:        &required,
	}
	id, err := Resolve(cfg, []string{"bash", "BOTMUX_USER_ID=alice", "-lc", "pwd"}, func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	if id.Value != "alice" || id.Source != "argv:BOTMUX_USER_ID" {
		t.Fatalf("unexpected identity: %#v", id)
	}
}
