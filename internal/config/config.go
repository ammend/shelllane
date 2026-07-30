package config

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	DefaultIdentityEnv = "SHELLLANE_IDENTITY"
	DefaultStateRoot   = "/var/lib/shelllane/users"
)

type Config struct {
	Version   int                 `yaml:"version"`
	Shells    map[string]string   `yaml:"shells"`
	Identity  IdentityConfig      `yaml:"identity"`
	State     StateConfig         `yaml:"state"`
	Routes    []Route             `yaml:"routes"`
	Nodes     map[string]Node     `yaml:"nodes"`
	NodePools map[string]NodePool `yaml:"nodePools"`
	Runtimes  map[string]Runtime  `yaml:"runtimes"`
	Targets   map[string]Target   `yaml:"targets"`
	Audit     AuditConfig         `yaml:"audit"`
}

type IdentityConfig struct {
	Env             string `yaml:"env"`
	ArgvEnvFallback bool   `yaml:"argvEnvFallback"`
	Required        *bool  `yaml:"required"`
	Hash            string `yaml:"hash"`
}

type StateConfig struct {
	Root string `yaml:"root"`
}

type Route struct {
	Match  Match  `yaml:"match"`
	Target string `yaml:"target"`
}

type Match struct {
	Identity   string   `yaml:"identity"`
	IdentityIn []string `yaml:"identityIn"`
}

type Node struct {
	Connector string   `yaml:"connector"`
	Address   string   `yaml:"address"`
	User      string   `yaml:"user"`
	Port      int      `yaml:"port"`
	Helper    string   `yaml:"helper"`
	SSHArgs   []string `yaml:"sshArgs"`
}

type NodePool struct {
	Nodes    []string `yaml:"nodes"`
	Strategy string   `yaml:"strategy"`
	Storage  string   `yaml:"storage"`
	Failover string   `yaml:"failover"`
	HashSeed string   `yaml:"hashSeed"`
}

type Runtime struct {
	Type       string            `yaml:"type"`
	Profile    string            `yaml:"profile"`
	StateMode  string            `yaml:"stateMode"`
	Engine     string            `yaml:"engine"`
	Image      string            `yaml:"image"`
	Platform   string            `yaml:"platform"`
	User       string            `yaml:"user"`
	PullPolicy string            `yaml:"pullPolicy"`
	Shell      map[string]string `yaml:"shell"`
	Workdir    string            `yaml:"workdir"`
	Network    string            `yaml:"network"`
	ReadOnly   bool              `yaml:"readOnly"`
	Mounts     []Mount           `yaml:"mounts"`
	Resources  ResourceConfig    `yaml:"resources"`
	Env        map[string]string `yaml:"env"`
	PassEnv    []string          `yaml:"passEnv"`
}

type Mount struct {
	Source   string `yaml:"source"`
	Target   string `yaml:"target"`
	ReadOnly bool   `yaml:"readOnly"`
}

func (m *Mount) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		parts := strings.SplitN(value.Value, ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return fmt.Errorf("mount must be source:target, got %q", value.Value)
		}
		m.Source, m.Target = parts[0], parts[1]
		return nil
	}
	type plain Mount
	var decoded plain
	if err := value.Decode(&decoded); err != nil {
		return err
	}
	*m = Mount(decoded)
	return nil
}

type ResourceConfig struct {
	CPUs   string `yaml:"cpus"`
	Memory string `yaml:"memory"`
	PIDs   int    `yaml:"pids"`
}

type Target struct {
	Node      string `yaml:"node"`
	NodePool  string `yaml:"nodePool"`
	Runtime   string `yaml:"runtime"`
	StateMode string `yaml:"stateMode"`
}

type AuditConfig struct {
	Path string `yaml:"path"`
}

func Load(path string) (*Config, string, error) {
	resolved, err := resolvePath(path)
	if err != nil {
		return nil, "", err
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return nil, "", fmt.Errorf("read config %s: %w", resolved, err)
	}
	var cfg Config
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, "", fmt.Errorf("decode config %s: %w", resolved, err)
	}
	cfg.defaults()
	if err := cfg.Validate(); err != nil {
		return nil, "", fmt.Errorf("validate config %s: %w", resolved, err)
	}
	return &cfg, resolved, nil
}

func resolvePath(explicit string) (string, error) {
	if explicit != "" {
		return expandHome(explicit), nil
	}
	if fromEnv := os.Getenv("SHELLLANE_CONFIG"); fromEnv != "" {
		return expandHome(fromEnv), nil
	}
	var candidates []string
	if configHome, err := os.UserConfigDir(); err == nil && configHome != "" {
		candidates = append(candidates, filepath.Join(configHome, "shelllane", "config.yaml"))
	}
	candidates = append(candidates, "/etc/shelllane/config.yaml")
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("configuration not found; set SHELLLANE_CONFIG or create %s", strings.Join(candidates, " or "))
}

func expandHome(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		if current, err := user.Current(); err == nil {
			return filepath.Join(current.HomeDir, strings.TrimPrefix(path, "~/"))
		}
	}
	return path
}

func (c *Config) defaults() {
	if c.Version == 0 {
		c.Version = 1
	}
	if c.Identity.Env == "" {
		c.Identity.Env = DefaultIdentityEnv
	}
	if c.Identity.Required == nil {
		value := true
		c.Identity.Required = &value
	}
	if c.Identity.Hash == "" {
		c.Identity.Hash = "sha256"
	}
	if c.State.Root == "" {
		c.State.Root = DefaultStateRoot
	}
	for name, node := range c.Nodes {
		if node.Port == 0 {
			node.Port = 22
		}
		if node.Connector == "ssh" && node.Helper == "" {
			node.Helper = "shelllane"
		}
		c.Nodes[name] = node
	}
	for name, runtime := range c.Runtimes {
		if len(runtime.PassEnv) == 0 {
			runtime.PassEnv = []string{
				"TERM", "COLORTERM", "LANG", "LC_*", "SHELLLANE_CONTEXT_*",
				"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY",
				"http_proxy", "https_proxy", "no_proxy",
			}
		}
		if runtime.Type == "container" {
			if runtime.Engine == "" {
				runtime.Engine = "docker"
			}
			if runtime.PullPolicy == "" {
				runtime.PullPolicy = "missing"
			}
			if runtime.Profile == "" {
				runtime.Profile = name
			}
			if runtime.Workdir == "" {
				runtime.Workdir = "/workspace"
			}
		}
		if runtime.Type == "host" && runtime.Profile == "" {
			runtime.Profile = "host"
		}
		if runtime.Type == "host" && runtime.Workdir == "" && runtime.StateMode != "existing-home" {
			runtime.Workdir = "{workspace}"
		}
		c.Runtimes[name] = runtime
	}
}

func (c *Config) Validate() error {
	var problems []string
	if c.Version != 1 {
		problems = append(problems, fmt.Sprintf("unsupported version %d", c.Version))
	}
	if c.Identity.Hash != "sha256" {
		problems = append(problems, "identity.hash must be sha256")
	}
	if c.Identity.Env == "" || strings.ContainsRune(c.Identity.Env, '=') {
		problems = append(problems, "identity.env must be a valid environment variable name")
	}
	for _, kind := range []string{"bash", "zsh", "sh"} {
		if value := c.Shells[kind]; value != "" && !filepath.IsAbs(value) {
			problems = append(problems, fmt.Sprintf("shells.%s must be absolute", kind))
		}
	}
	if !filepath.IsAbs(c.State.Root) {
		problems = append(problems, "state.root must be absolute")
	}
	if len(c.Routes) == 0 {
		problems = append(problems, "at least one route is required")
	}
	hasDefaultRoute := false
	for i, route := range c.Routes {
		if route.Target == "" {
			problems = append(problems, fmt.Sprintf("routes[%d].target is required", i))
		} else if _, ok := c.Targets[route.Target]; !ok {
			problems = append(problems, fmt.Sprintf("routes[%d] references unknown target %q", i, route.Target))
		}
		if route.Match.Identity != "" && len(route.Match.IdentityIn) != 0 {
			problems = append(problems, fmt.Sprintf("routes[%d] cannot set both identity and identityIn", i))
		}
		if route.Match.Identity == "" && len(route.Match.IdentityIn) == 0 {
			hasDefaultRoute = true
			if i != len(c.Routes)-1 {
				problems = append(problems, fmt.Sprintf("routes[%d] is a default route and must be last", i))
			}
		}
	}
	if len(c.Routes) != 0 && !hasDefaultRoute {
		problems = append(problems, "a final default route is required")
	}
	for name, node := range c.Nodes {
		switch node.Connector {
		case "local":
			if node.Address != "" || node.User != "" {
				problems = append(problems, fmt.Sprintf("local node %q cannot set address or user", name))
			}
		case "ssh":
			if node.Address == "" || node.User == "" {
				problems = append(problems, fmt.Sprintf("ssh node %q requires address and user", name))
			}
			if node.Helper == "" || strings.ContainsAny(node.Helper, " \t\r\n;&|`$'\"\\(){}[]<>!?*") {
				problems = append(problems, fmt.Sprintf("ssh node %q helper must be a simple executable path", name))
			}
		default:
			problems = append(problems, fmt.Sprintf("node %q has unsupported connector %q", name, node.Connector))
		}
	}
	for name, pool := range c.NodePools {
		if len(pool.Nodes) == 0 {
			problems = append(problems, fmt.Sprintf("nodePool %q must contain nodes", name))
		}
		if pool.Strategy != "" && pool.Strategy != "rendezvous-hash" {
			problems = append(problems, fmt.Sprintf("nodePool %q only supports rendezvous-hash", name))
		}
		for _, node := range pool.Nodes {
			if _, ok := c.Nodes[node]; !ok {
				problems = append(problems, fmt.Sprintf("nodePool %q references unknown node %q", name, node))
			}
		}
	}
	for name, runtime := range c.Runtimes {
		switch runtime.Type {
		case "host":
		case "container":
			if runtime.Image == "" {
				problems = append(problems, fmt.Sprintf("container runtime %q requires image", name))
			}
			if runtime.Engine != "docker" && runtime.Engine != "podman" {
				problems = append(problems, fmt.Sprintf("runtime %q engine must be docker or podman", name))
			}
			switch runtime.PullPolicy {
			case "always", "missing", "never":
			default:
				problems = append(problems, fmt.Sprintf("runtime %q pullPolicy must be always, missing, or never", name))
			}
			for i, mount := range runtime.Mounts {
				if mount.Source == "" || filepath.IsAbs(mount.Source) || filepath.Clean(mount.Source) == ".." || strings.HasPrefix(filepath.Clean(mount.Source), "../") {
					problems = append(problems, fmt.Sprintf("runtime %q mount[%d] source must be a relative user-state path", name, i))
				}
				if !filepath.IsAbs(mount.Target) {
					problems = append(problems, fmt.Sprintf("runtime %q mount[%d] target must be absolute", name, i))
				}
			}
		default:
			problems = append(problems, fmt.Sprintf("runtime %q has unsupported type %q", name, runtime.Type))
		}
	}
	for name, target := range c.Targets {
		if (target.Node == "") == (target.NodePool == "") {
			problems = append(problems, fmt.Sprintf("target %q must set exactly one of node or nodePool", name))
		}
		if target.Node != "" {
			if _, ok := c.Nodes[target.Node]; !ok {
				problems = append(problems, fmt.Sprintf("target %q references unknown node %q", name, target.Node))
			}
		}
		if target.NodePool != "" {
			if _, ok := c.NodePools[target.NodePool]; !ok {
				problems = append(problems, fmt.Sprintf("target %q references unknown nodePool %q", name, target.NodePool))
			}
		}
		if _, ok := c.Runtimes[target.Runtime]; !ok {
			problems = append(problems, fmt.Sprintf("target %q references unknown runtime %q", name, target.Runtime))
		}
	}
	if len(problems) != 0 {
		sort.Strings(problems)
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}
