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
