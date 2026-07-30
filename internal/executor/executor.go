package executor

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/ammend/shelllane/internal/config"
	"github.com/ammend/shelllane/internal/identity"
	"github.com/ammend/shelllane/internal/invocation"
	"github.com/ammend/shelllane/internal/routing"
	"github.com/ammend/shelllane/internal/state"
)

type SyscallExec func(path string, argv []string, envv []string) error

type Runner struct {
	Config *config.Config
	Exec   SyscallExec
}

func New(cfg *config.Config) *Runner {
	return &Runner{Config: cfg, Exec: syscall.Exec}
}

func (r *Runner) Execute(res routing.Resolution, id identity.Identity, shellKind string, args, environment []string, cwd, requestID string, tty bool) error {
	stateKey, err := state.IdentityKey(r.Config.State, id)
	if err != nil {
		return err
	}
	id.StateKey = stateKey
	// Identity is routing metadata and user context. Always preserve both the
	// canonical and mapped names across container and SSH allowlist boundaries.
	res.Runtime.PassEnv = appendIdentityEnv(res.Runtime.PassEnv, identity.EnvNames(r.Config.Identity))
	if res.Node.Connector == "ssh" {
		return r.executeSSH(res, id, shellKind, args, environment, cwd, requestID, tty)
	}
	if res.Node.Connector != "local" {
		return fmt.Errorf("backend_unavailable: unsupported connector %q", res.Node.Connector)
	}
	return r.executeLocal(res.Runtime, res.Target, id, shellKind, args, environment, cwd, requestID, tty)
}

func appendIdentityEnv(allow, names []string) []string {
	result := append([]string(nil), allow...)
	seen := make(map[string]struct{}, len(result))
	for _, name := range result {
		seen[name] = struct{}{}
	}
	for _, name := range names {
		if _, found := seen[name]; !found {
			result = append(result, name)
			seen[name] = struct{}{}
		}
	}
	return result
}

func (r *Runner) ExecuteRemote(request invocation.Request) error {
	runtime, ok := r.Config.Runtimes[request.Runtime]
	if !ok {
		return fmt.Errorf("runtime_unavailable: remote runtime %q is not configured", request.Runtime)
	}
	id := identity.Identity{Hash: request.IdentityHash, StateKey: request.IdentityKey, Source: "remote-request"}
	if id.StateKey == "" {
		// Compatibility with requests emitted before identity_key was added.
		id.StateKey = request.IdentityHash
	}
	target := config.Target{Runtime: request.Runtime, StateMode: request.StateMode}
	env := environmentMapToSlice(request.Env)
	return r.executeLocal(runtime, target, id, request.ShellKind, request.Args, env, request.Cwd, request.RequestID, request.TTY)
}

func (r *Runner) executeLocal(runtime config.Runtime, target config.Target, id identity.Identity, shellKind string, args, environment []string, cwd, requestID string, tty bool) error {
	paths, err := state.Prepare(r.Config.State, runtime, target, id)
	if err != nil {
		return err
	}
	switch runtime.Type {
	case "host":
		return r.executeHost(runtime, paths, shellKind, args, environment, cwd)
	case "container":
		return r.executeContainer(runtime, paths, shellKind, args, environment, requestID, tty)
	default:
		return fmt.Errorf("runtime_unavailable: unsupported runtime %q", runtime.Type)
	}
}

func (r *Runner) executeHost(runtime config.Runtime, paths state.Paths, shellKind string, args, environment []string, cwd string) error {
	realShell, err := r.realShell(runtime, shellKind)
	if err != nil {
		return err
	}
	targetCwd, err := resolveHostCwd(runtime.Workdir, cwd, paths)
	if err != nil {
		return err
	}
	if err := os.Chdir(targetCwd); err != nil {
		return fmt.Errorf("cwd_unmappable: chdir %s: %w", targetCwd, err)
	}
	execArgs := append([]string(nil), args...)
	return r.Exec(realShell, execArgs, state.Environment(environment, paths))
}

func (r *Runner) executeContainer(runtime config.Runtime, paths state.Paths, shellKind string, args, environment []string, requestID string, tty bool) error {
	engine, err := exec.LookPath(runtime.Engine)
	if err != nil {
		return fmt.Errorf("runtime_unavailable: %s not found: %w", runtime.Engine, err)
	}
	command, err := ContainerArgs(runtime, paths, shellKind, args, environment, requestID, tty)
	if err != nil {
		return err
	}
	return r.Exec(engine, append([]string{runtime.Engine}, command...), environment)
}

func ContainerArgs(runtime config.Runtime, paths state.Paths, shellKind string, args, environment []string, requestID string, tty bool) ([]string, error) {
	realShell := runtime.Shell[shellKind]
	if realShell == "" {
		switch shellKind {
		case "bash":
			realShell = "/bin/bash"
		case "zsh":
			realShell = "/bin/zsh"
		default:
			realShell = "/bin/sh"
		}
	}
	result := []string{"run", "--rm", "--interactive"}
	if tty {
		result = append(result, "--tty")
	}
	if runtime.PullPolicy != "" {
		result = append(result, "--pull", runtime.PullPolicy)
	}
	if runtime.Platform != "" {
		result = append(result, "--platform", runtime.Platform)
	}
	if runtime.User != "" {
		result = append(result, "--user", runtime.User)
	}
	if runtime.ReadOnly {
		result = append(result, "--read-only")
	}
	if runtime.Network == "restricted" {
		result = append(result, "--network", "none")
	} else if runtime.Network != "" {
		result = append(result, "--network", runtime.Network)
	}
	if runtime.Resources.CPUs != "" {
		result = append(result, "--cpus", runtime.Resources.CPUs)
	}
	if runtime.Resources.Memory != "" {
		result = append(result, "--memory", runtime.Resources.Memory)
	}
	if runtime.Resources.PIDs > 0 {
		result = append(result, "--pids-limit", strconv.Itoa(runtime.Resources.PIDs))
	}
	result = append(result, "--cap-drop", "ALL", "--security-opt", "no-new-privileges", "--tmpfs", "/tmp")
	if requestID != "" {
		name := "shelllane-" + sanitizeName(requestID)
		if len(name) > 63 {
			name = name[:63]
		}
		result = append(result, "--name", name)
	}

	mounts := map[string]config.Mount{
		"/home/agent":              {Source: profileRelative(runtime.Profile, "home"), Target: "/home/agent"},
		"/home/agent/.config":      {Source: "shared/config", Target: "/home/agent/.config"},
		"/home/agent/.local/share": {Source: "shared/data", Target: "/home/agent/.local/share"},
		"/home/agent/.cache":       {Source: profileRelative(runtime.Profile, "cache"), Target: "/home/agent/.cache"},
		"/workspace":               {Source: "shared/workspace", Target: "/workspace"},
	}
	for _, mount := range runtime.Mounts {
		mounts[mount.Target] = mount
	}
	targets := make([]string, 0, len(mounts))
	for target := range mounts {
		targets = append(targets, target)
	}
	sort.Slice(targets, func(i, j int) bool {
		depthI, depthJ := strings.Count(targets[i], "/"), strings.Count(targets[j], "/")
		if depthI == depthJ {
			return targets[i] < targets[j]
		}
		return depthI < depthJ
	})
	for _, target := range targets {
		mount := mounts[target]
		source, err := paths.Resolve(mount.Source)
		if err != nil {
			return nil, err
		}
		if err := os.MkdirAll(source, 0o700); err != nil {
			return nil, fmt.Errorf("state_prepare_failed: create mount source: %w", err)
		}
		spec := "type=bind,src=" + source + ",dst=" + mount.Target
		if mount.ReadOnly {
			spec += ",readonly"
		}
		result = append(result, "--mount", spec)
	}

	envMap := filterEnvironment(environment, runtime.PassEnv)
	envMap["HOME"] = "/home/agent"
	envMap["XDG_CONFIG_HOME"] = "/home/agent/.config"
	envMap["XDG_DATA_HOME"] = "/home/agent/.local/share"
	envMap["XDG_CACHE_HOME"] = "/home/agent/.cache"
	envMap["TMPDIR"] = "/tmp"
	for key, value := range runtime.Env {
		envMap[key] = value
	}
	envKeys := make([]string, 0, len(envMap))
	for key := range envMap {
		envKeys = append(envKeys, key)
	}
	sort.Strings(envKeys)
	for _, key := range envKeys {
		result = append(result, "--env", key+"="+envMap[key])
	}
	if runtime.Workdir != "" {
		result = append(result, "--workdir", runtime.Workdir)
	}
	result = append(result, runtime.Image, realShell)
	result = append(result, args[1:]...)
	return result, nil
}

func (r *Runner) executeSSH(res routing.Resolution, id identity.Identity, shellKind string, args, environment []string, cwd, requestID string, tty bool) error {
	ssh, err := exec.LookPath("ssh")
	if err != nil {
		return fmt.Errorf("ssh_failed: ssh executable not found: %w", err)
	}
	request := invocation.Request{
		Version:      1,
		RequestID:    requestID,
		IdentityHash: id.Hash,
		IdentityKey:  id.StateKey,
		Runtime:      res.RuntimeName,
		StateMode:    res.Target.StateMode,
		ShellKind:    shellKind,
		Args:         append([]string(nil), args...),
		Cwd:          cwd,
		TTY:          tty,
		Env:          filterEnvironment(environment, res.Runtime.PassEnv),
	}
	encoded, err := request.Encode()
	if err != nil {
		return fmt.Errorf("ssh_failed: encode request: %w", err)
	}
	sshArgs := []string{"ssh"}
	if tty {
		sshArgs = append(sshArgs, "-tt")
	} else {
		sshArgs = append(sshArgs, "-T")
	}
	sshArgs = append(sshArgs,
		"-o", "BatchMode=yes",
		"-o", "StrictHostKeyChecking=yes",
		"-p", strconv.Itoa(res.Node.Port),
	)
	sshArgs = append(sshArgs, res.Node.SSHArgs...)
	host := res.Node.User + "@" + res.Node.Address
	remoteCommand := "SHELLLANE_INTERNAL_REQUEST=" + encoded + " " + res.Node.Helper + " run-remote"
	sshArgs = append(sshArgs, host, remoteCommand)
	return r.Exec(ssh, sshArgs, environment)
}

func (r *Runner) realShell(runtime config.Runtime, kind string) (string, error) {
	path := runtime.Shell[kind]
	if path == "" {
		path = r.Config.Shells[kind]
	}
	if path == "" {
		return "", fmt.Errorf("real_shell_missing: no %s shell configured", kind)
	}
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("real_shell_missing: shell path %q is not absolute", path)
	}
	resolvedShell, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("real_shell_missing: %w", err)
	}
	self, err := os.Executable()
	if err == nil {
		resolvedSelf, resolveErr := filepath.EvalSymlinks(self)
		if resolveErr == nil && resolvedSelf == resolvedShell {
			return "", errors.New("recursive_shell: configured real shell points to ShellLane")
		}
	}
	return path, nil
}

func resolveHostCwd(template, requested string, paths state.Paths) (string, error) {
	target := template
	if target == "" {
		target = requested
	}
	target = strings.ReplaceAll(target, "{workspace}", paths.Workspace)
	target = strings.ReplaceAll(target, "{home}", paths.Home)
	if target == "" {
		target = paths.Home
	}
	info, err := os.Stat(target)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("cwd_unmappable: %s is not a directory", target)
	}
	return target, nil
}

func filterEnvironment(environment, allow []string) map[string]string {
	result := map[string]string{}
	for _, item := range environment {
		key, value, found := strings.Cut(item, "=")
		if !found || !allowed(key, allow) {
			continue
		}
		result[key] = value
	}
	return result
}

func allowed(key string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.HasSuffix(pattern, "*") {
			if strings.HasPrefix(key, strings.TrimSuffix(pattern, "*")) {
				return true
			}
		} else if key == pattern {
			return true
		}
	}
	return false
}

func environmentMapToSlice(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		result = append(result, key+"="+values[key])
	}
	return result
}

func profileRelative(profile, leaf string) string {
	if profile == "" {
		profile = "container"
	}
	return filepath.Join("profiles", profile, leaf)
}

func sanitizeName(value string) string {
	var builder strings.Builder
	for _, char := range strings.ToLower(value) {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' || char == '_' {
			builder.WriteRune(char)
		} else {
			builder.WriteByte('-')
		}
	}
	result := strings.Trim(builder.String(), "-_")
	if result == "" {
		return "run"
	}
	return result
}
