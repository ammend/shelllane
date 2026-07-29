package state

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ammend/shelllane/internal/config"
	"github.com/ammend/shelllane/internal/identity"
)

type Paths struct {
	Root        string
	Home        string
	Config      string
	Data        string
	State       string
	Credentials string
	Workspace   string
	Cache       string
	Tmp         string
}

func Prepare(cfg config.StateConfig, runtime config.Runtime, target config.Target, id identity.Identity) (Paths, error) {
	mode := target.StateMode
	if mode == "" {
		mode = runtime.StateMode
	}
	if mode == "existing-home" {
		home, err := os.UserHomeDir()
		if err != nil {
			return Paths{}, fmt.Errorf("state_prepare_failed: resolve existing home: %w", err)
		}
		return Paths{
			Root:        home,
			Home:        home,
			Config:      filepath.Join(home, ".config"),
			Data:        filepath.Join(home, ".local", "share"),
			State:       filepath.Join(home, ".local", "state"),
			Credentials: filepath.Join(home, ".config"),
			Workspace:   "",
			Cache:       filepath.Join(home, ".cache"),
			Tmp:         os.TempDir(),
		}, nil
	}
	if mode != "" && mode != "isolated" {
		return Paths{}, fmt.Errorf("state_prepare_failed: unsupported state mode %q", mode)
	}
	profile := runtime.Profile
	if profile == "" {
		profile = runtime.Type
	}
	root := filepath.Join(cfg.Root, id.Hash)
	paths := Paths{
		Root:        root,
		Home:        filepath.Join(root, "profiles", profile, "home"),
		Config:      filepath.Join(root, "shared", "config"),
		Data:        filepath.Join(root, "shared", "data"),
		State:       filepath.Join(root, "shared", "state"),
		Credentials: filepath.Join(root, "shared", "credentials"),
		Workspace:   filepath.Join(root, "shared", "workspace"),
		Cache:       filepath.Join(root, "profiles", profile, "cache"),
		Tmp:         filepath.Join(root, "tmp"),
	}
	for _, path := range []string{
		paths.Home, paths.Config, paths.Data, paths.State, paths.Credentials,
		paths.Workspace, paths.Cache, paths.Tmp,
		// Docker otherwise creates these nested bind-mount targets as root.
		// Preparing them here preserves the ShellLane service account's
		// ownership when the daemon runs with greater privileges.
		filepath.Join(paths.Home, ".config"),
		filepath.Join(paths.Home, ".local", "share"),
		filepath.Join(paths.Home, ".cache"),
	} {
		if err := secureMkdirAll(cfg.Root, path); err != nil {
			return Paths{}, fmt.Errorf("state_prepare_failed: %w", err)
		}
	}
	return paths, nil
}

func secureMkdirAll(root, path string) error {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path %q escapes state root %q", path, root)
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return err
	}
	if info, err := os.Lstat(root); err != nil {
		return err
	} else if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("state root %q is not a real directory", root)
	}
	current := root
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		if component == "." || component == "" {
			continue
		}
		current = filepath.Join(current, component)
		if err := os.Mkdir(current, 0o700); err != nil && !os.IsExist(err) {
			return err
		}
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("state component %q is not a real directory", current)
		}
		if err := os.Chmod(current, 0o700); err != nil {
			return err
		}
	}
	return nil
}

func (p Paths) Resolve(relative string) (string, error) {
	clean := filepath.Clean(relative)
	if clean == "." || clean == ".." || filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid user-state mount source %q", relative)
	}
	path := filepath.Join(p.Root, clean)
	rel, err := filepath.Rel(p.Root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("mount source %q escapes user state", relative)
	}
	return path, nil
}

func Environment(base []string, paths Paths) []string {
	overrides := map[string]string{
		"HOME":              paths.Home,
		"XDG_CONFIG_HOME":   paths.Config,
		"XDG_DATA_HOME":     paths.Data,
		"XDG_CACHE_HOME":    paths.Cache,
		"XDG_STATE_HOME":    paths.State,
		"TMPDIR":            paths.Tmp,
		"SHELLLANE_MANAGED": "1",
	}
	return mergeEnvironment(base, overrides)
}

func mergeEnvironment(base []string, overrides map[string]string) []string {
	result := make([]string, 0, len(base)+len(overrides))
	for _, item := range base {
		key, _, found := strings.Cut(item, "=")
		if _, overridden := overrides[key]; overridden {
			continue
		}
		if strings.HasPrefix(key, "SHELLLANE_INTERNAL_") {
			continue
		}
		if found {
			result = append(result, item)
		}
	}
	for key, value := range overrides {
		result = append(result, key+"="+value)
	}
	return result
}
