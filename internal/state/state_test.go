package state

import (
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
		identity.Identity{Hash: "abc"},
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
	if err := os.Mkdir(filepath.Join(root, "abc"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "outside"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "outside"), filepath.Join(root, "abc", "profiles")); err != nil {
		t.Fatal(err)
	}
	_, err := Prepare(
		config.StateConfig{Root: root},
		config.Runtime{Type: "host", Profile: "host"},
		config.Target{},
		identity.Identity{Hash: "abc"},
	)
	if err == nil {
		t.Fatal("expected symlink rejection")
	}
}

func TestResolveRejectsEscape(t *testing.T) {
	if _, err := (Paths{Root: "/tmp/user"}).Resolve("../other"); err == nil {
		t.Fatal("expected escape rejection")
	}
}
