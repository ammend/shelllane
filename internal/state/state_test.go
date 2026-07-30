package state

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/ammend/shelllane/internal/config"
	"github.com/ammend/shelllane/internal/identity"
)

func TestPrepareIsolatedState(t *testing.T) {
	root := t.TempDir()
	paths, err := Prepare(
		config.StateConfig{Root: root},
		config.Runtime{Type: "container", Profile: "default"},
		config.Target{},
		identity.Identity{Value: "alice", Hash: "abc"},
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		paths.Home, paths.Config, paths.Data, paths.State, paths.Workspace, paths.Cache, paths.Tmp,
		filepath.Join(paths.Home, ".config"),
		filepath.Join(paths.Home, ".local", "share"),
		filepath.Join(paths.Home, ".cache"),
	} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o700 {
			t.Fatalf("%s mode is %o", path, info.Mode().Perm())
		}
	}
}

func TestPrepareRejectsSymlinkComponent(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "alice"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "outside"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "outside"), filepath.Join(root, "alice", "profiles")); err != nil {
		t.Fatal(err)
	}
	_, err := Prepare(
		config.StateConfig{Root: root},
		config.Runtime{Type: "host", Profile: "host"},
		config.Target{},
		identity.Identity{Value: "alice", Hash: "abc"},
	)
	if err == nil {
		t.Fatal("expected symlink rejection")
	}
}

func TestIdentityKeyModes(t *testing.T) {
	id := identity.Identity{Value: "alice@example.com", Hash: "abc123"}
	tests := []struct {
		mode string
		want string
	}{
		{"", "alice@example.com"},
		{"plain", "alice@example.com"},
		{"base64url", base64.RawURLEncoding.EncodeToString([]byte("alice@example.com"))},
		{"hash", "abc123"},
	}
	for _, test := range tests {
		got, err := IdentityKey(config.StateConfig{IdentityNaming: test.mode}, id)
		if err != nil {
			t.Fatalf("%s: %v", test.mode, err)
		}
		if got != test.want {
			t.Fatalf("%s: got %q, want %q", test.mode, got, test.want)
		}
	}
}

func TestPlainIdentityRejectsUnsafePath(t *testing.T) {
	_, err := IdentityKey(
		config.StateConfig{IdentityNaming: "plain"},
		identity.Identity{Value: "../alice", Hash: "abc"},
	)
	if err == nil {
		t.Fatal("expected unsafe plain identity rejection")
	}
}

func TestResolveRejectsEscape(t *testing.T) {
	if _, err := (Paths{Root: "/tmp/user"}).Resolve("../other"); err == nil {
		t.Fatal("expected escape rejection")
	}
}
