package executor

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/ammend/shelllane/internal/config"
	"github.com/ammend/shelllane/internal/identity"
	"github.com/ammend/shelllane/internal/routing"
	"github.com/ammend/shelllane/internal/state"
)

func TestContainerArgsAreIsolatedAndAllowlisted(t *testing.T) {
	root := t.TempDir()
	paths := state.Paths{Root: root}
	runtime := config.Runtime{
		Type: "container", Profile: "default", Image: "example/shelllane@sha256:abc",
		PullPolicy: "never", Workdir: "/workspace", ReadOnly: true,
		PassEnv:   []string{"TERM", "SHELLLANE_CONTEXT_*"},
		Resources: config.ResourceConfig{CPUs: "1", Memory: "1g", PIDs: 128},
	}
	args, err := ContainerArgs(runtime, paths, "sh", []string{"sh", "-lc", "true"},
		[]string{"TERM=xterm", "SHELLLANE_CONTEXT_REQUEST=user", "SECRET=do-not-pass"}, "request", true)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, "\x00")
	for _, required := range []string{
		"--read-only", "--cap-drop\x00ALL", "no-new-privileges",
		"TERM=xterm", "SHELLLANE_CONTEXT_REQUEST=user", "example/shelllane@sha256:abc",
	} {
		if !strings.Contains(joined, required) {
			t.Fatalf("missing %q in %#v", required, args)
		}
	}
	if strings.Contains(joined, "SECRET") {
		t.Fatalf("secret leaked into container environment: %#v", args)
	}
}

func TestHostPreservesArgvZeroAndIsolatesHome(t *testing.T) {
	stateRoot := t.TempDir()
	cfg := &config.Config{
		Shells: map[string]string{"sh": "/bin/sh"},
		State:  config.StateConfig{Root: stateRoot},
	}
	runner := New(cfg)
	var gotPath string
	var gotArgs, gotEnv []string
	runner.Exec = func(path string, argv, env []string) error {
		gotPath, gotArgs, gotEnv = path, argv, env
		return nil
	}
	runtime := config.Runtime{Type: "host", Profile: "host", Workdir: "{workspace}"}
	err := runner.executeLocal(runtime, config.Target{}, identity.Identity{Hash: "abc"}, "sh",
		[]string{"sh", "-c", "true"}, []string{"HOME=/unsafe", "SHELLLANE_INTERNAL_REQUEST=x"}, "/", "request", false)
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/bin/sh" || gotArgs[0] != "sh" {
		t.Fatalf("unexpected exec: %q %#v", gotPath, gotArgs)
	}
	joined := strings.Join(gotEnv, "\n")
	if strings.Contains(joined, "HOME=/unsafe") || strings.Contains(joined, "SHELLLANE_INTERNAL_REQUEST") {
		t.Fatalf("unsafe environment retained: %s", joined)
	}
	if !strings.Contains(joined, "HOME="+filepath.Join(stateRoot, "abc", "profiles", "host", "home")) {
		t.Fatalf("isolated HOME missing: %s", joined)
	}
}

func TestSSHBuildsStrictCommand(t *testing.T) {
	cfg := &config.Config{}
	runner := New(cfg)
	var gotArgs []string
	runner.Exec = func(_ string, argv, _ []string) error {
		gotArgs = argv
		return nil
	}
	resolution := routing.Resolution{
		Node:        config.Node{Connector: "ssh", Address: "dev.example", User: "runner", Port: 2222, Helper: "shelllane"},
		RuntimeName: "host", Runtime: config.Runtime{PassEnv: []string{"TERM"}},
	}
	err := runner.executeSSH(resolution, identity.Identity{Hash: "abc"}, "sh",
		[]string{"sh", "-lc", "true"}, []string{"TERM=xterm", "SECRET=no"}, "/tmp", "req", false)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(gotArgs, " ")
	for _, required := range []string{"-T", "BatchMode=yes", "StrictHostKeyChecking=yes", "-p 2222", "runner@dev.example", "shelllane run-remote"} {
		if !strings.Contains(joined, required) {
			t.Fatalf("missing %q in %s", required, joined)
		}
	}
	if strings.Contains(joined, "SECRET") {
		t.Fatalf("secret leaked into request: %s", joined)
	}
}
