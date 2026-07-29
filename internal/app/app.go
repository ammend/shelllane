package app

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ammend/shelllane/internal/audit"
	"github.com/ammend/shelllane/internal/config"
	"github.com/ammend/shelllane/internal/executor"
	"github.com/ammend/shelllane/internal/identity"
	"github.com/ammend/shelllane/internal/invocation"
	"github.com/ammend/shelllane/internal/routing"
	"golang.org/x/term"
)

var Version = "dev"

type App struct {
	Stdout    io.Writer
	Stderr    io.Writer
	Getenv    func(string) string
	Environ   func() []string
	Getwd     func() (string, error)
	IsTTY     func() bool
	NewRunner func(*config.Config) *executor.Runner
}

func New() *App {
	return &App{
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
		Getenv:    os.Getenv,
		Environ:   os.Environ,
		Getwd:     os.Getwd,
		IsTTY:     func() bool { return term.IsTerminal(int(os.Stdin.Fd())) },
		NewRunner: executor.New,
	}
}

func (a *App) Run(argv []string) error {
	if len(argv) == 0 {
		return errors.New("missing argv[0]")
	}
	switch name := filepath.Base(argv[0]); name {
	case "bash", "zsh", "sh":
		return a.runShell(name, argv)
	}
	if len(argv) < 2 {
		a.usage()
		return flag.ErrHelp
	}
	switch argv[1] {
	case "version", "--version", "-version":
		_, err := fmt.Fprintln(a.Stdout, Version)
		return err
	case "config":
		return a.runConfig(argv[2:])
	case "route":
		return a.runRoute(argv[2:])
	case "doctor":
		return a.runDoctor(argv[2:])
	case "run-remote":
		return a.runRemote(argv[2:])
	case "help", "-h", "--help":
		a.usage()
		return nil
	default:
		return fmt.Errorf("unknown command %q", argv[1])
	}
}

func (a *App) runShell(kind string, argv []string) error {
	cfg, _, err := config.Load("")
	if err != nil {
		return err
	}
	id, err := identity.Resolve(cfg.Identity, argv, a.Getenv)
	if err != nil {
		return err
	}
	resolution, err := routing.Resolve(cfg, id)
	if err != nil {
		return err
	}
	cwd, err := a.Getwd()
	if err != nil {
		return fmt.Errorf("resolve cwd: %w", err)
	}
	requestID := newRequestID()
	event := audit.Event{
		RequestID: requestID, Event: "dispatch", IdentityHash: id.Hash,
		IdentitySource: id.Source, RouteIndex: resolution.RouteIndex,
		Target: resolution.TargetName, Node: resolution.NodeName,
		Connector: resolution.Node.Connector, Runtime: resolution.RuntimeName,
		Shell: kind, TTY: a.IsTTY(),
	}
	if err := audit.Write(cfg.Audit.Path, event); err != nil {
		return err
	}
	forwarded := stripIdentityArg(argv, cfg.Identity)
	err = a.NewRunner(cfg).Execute(resolution, id, kind, forwarded, a.Environ(), cwd, requestID, event.TTY)
	if err != nil {
		event.Event, event.Error = "dispatch_error", err.Error()
		_ = audit.Write(cfg.Audit.Path, event)
	}
	return err
}

func (a *App) runConfig(args []string) error {
	if len(args) == 0 || args[0] != "check" {
		return errors.New("usage: shelllane config check [--config path]")
	}
	set := flag.NewFlagSet("config check", flag.ContinueOnError)
	set.SetOutput(a.Stderr)
	path := set.String("config", "", "configuration file")
	if err := set.Parse(args[1:]); err != nil {
		return err
	}
	_, resolved, err := config.Load(*path)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(a.Stdout, "configuration valid: %s\n", resolved)
	return err
}

func (a *App) runRoute(args []string) error {
	if len(args) == 0 || args[0] != "test" {
		return errors.New("usage: shelllane route test --identity value [--config path]")
	}
	set := flag.NewFlagSet("route test", flag.ContinueOnError)
	set.SetOutput(a.Stderr)
	path := set.String("config", "", "configuration file")
	value := set.String("identity", "", "identity value")
	if err := set.Parse(args[1:]); err != nil {
		return err
	}
	if *value == "" {
		return errors.New("--identity is required")
	}
	cfg, _, err := config.Load(*path)
	if err != nil {
		return err
	}
	idCfg := cfg.Identity
	idCfg.Env = "SHELLLANE_ROUTE_TEST_IDENTITY"
	id, err := identity.Resolve(idCfg, []string{"shelllane"}, func(string) string { return *value })
	if err != nil {
		return err
	}
	resolution, err := routing.Resolve(cfg, id)
	if err != nil {
		return err
	}
	output := struct {
		IdentityHash string `json:"identityHash"`
		RouteIndex   int    `json:"routeIndex"`
		Target       string `json:"target"`
		Node         string `json:"node"`
		Connector    string `json:"connector"`
		Runtime      string `json:"runtime"`
	}{
		id.Hash, resolution.RouteIndex, resolution.TargetName, resolution.NodeName,
		resolution.Node.Connector, resolution.RuntimeName,
	}
	encoder := json.NewEncoder(a.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func (a *App) runDoctor(args []string) error {
	set := flag.NewFlagSet("doctor", flag.ContinueOnError)
	set.SetOutput(a.Stderr)
	path := set.String("config", "", "configuration file")
	if err := set.Parse(args); err != nil {
		return err
	}
	cfg, resolved, err := config.Load(*path)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(a.Stdout, "ok config=%s routes=%d nodes=%d runtimes=%d targets=%d\n",
		resolved, len(cfg.Routes), len(cfg.Nodes), len(cfg.Runtimes), len(cfg.Targets))
	return err
}

func (a *App) runRemote(args []string) error {
	if len(args) != 0 {
		return errors.New("run-remote accepts no arguments")
	}
	encoded := a.Getenv("SHELLLANE_INTERNAL_REQUEST")
	if encoded == "" {
		return errors.New("missing SHELLLANE_INTERNAL_REQUEST")
	}
	request, err := invocation.Decode(encoded)
	if err != nil {
		return err
	}
	cfg, _, err := config.Load("")
	if err != nil {
		return err
	}
	return a.NewRunner(cfg).ExecuteRemote(request)
}

func stripIdentityArg(argv []string, cfg config.IdentityConfig) []string {
	if !cfg.ArgvEnvFallback {
		return append([]string(nil), argv...)
	}
	prefix := cfg.Env + "="
	result := make([]string, 0, len(argv))
	for index, arg := range argv {
		if index != 0 && strings.HasPrefix(arg, prefix) {
			continue
		}
		result = append(result, arg)
	}
	return result
}

func newRequestID() string {
	var data [12]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(data[:])
}

func (a *App) usage() {
	fmt.Fprintln(a.Stdout, `ShellLane - identity-aware shell execution

Usage:
  shelllane config check [--config path]
  shelllane route test --identity value [--config path]
  shelllane doctor [--config path]
  shelllane version

Install symlinks named bash, zsh, or sh to use ShellLane as a shell wrapper.`)
}
