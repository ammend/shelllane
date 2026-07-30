package identity

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/ammend/shelllane/internal/config"
)

var (
	ErrMissing  = errors.New("identity_missing")
	ErrConflict = errors.New("identity_conflict")
	ErrInvalid  = errors.New("identity_invalid")
)

type Identity struct {
	Value    string
	Hash     string
	Source   string
	StateKey string
}

func Resolve(cfg config.IdentityConfig, args []string, getenv func(string) string) (Identity, error) {
	type candidate struct {
		value  string
		source string
	}
	var candidates []candidate
	envNames := EnvNames(cfg)
	for _, name := range envNames {
		if value := strings.TrimSpace(getenv(name)); value != "" {
			candidates = append(candidates, candidate{value: value, source: "env:" + name})
		}
	}
	if cfg.ArgvEnvFallback {
		for _, arg := range args[1:] {
			for _, name := range envNames {
				prefix := name + "="
				if strings.HasPrefix(arg, prefix) {
					value := strings.TrimSpace(strings.TrimPrefix(arg, prefix))
					if value != "" {
						candidates = append(candidates, candidate{value: value, source: "argv:" + name})
					}
				}
			}
		}
	}
	value, source := "", ""
	for _, candidate := range candidates {
		if value != "" && value != candidate.value {
			return Identity{}, fmt.Errorf("%w: identity sources disagree", ErrConflict)
		}
		if value == "" {
			value, source = candidate.value, candidate.source
		}
	}
	if value == "" {
		if cfg.Required == nil || *cfg.Required {
			return Identity{}, ErrMissing
		}
		value, source = "anonymous", "fallback:anonymous"
	}
	if !utf8.ValidString(value) || len(value) > 512 || strings.ContainsRune(value, 0) {
		return Identity{}, ErrInvalid
	}
	sum := sha256.Sum256([]byte(value))
	return Identity{
		Value:  value,
		Hash:   hex.EncodeToString(sum[:]),
		Source: source,
	}, nil
}

// EnvNames returns the canonical identity variable followed by any mapped
// upstream variable. Callers use the same list for resolution and argv
// metadata removal.
func EnvNames(cfg config.IdentityConfig) []string {
	names := []string{cfg.Env}
	if expression, ok := cfg.EnvMap[cfg.Env]; ok {
		if mapped := envName(expression); mapped != "" && mapped != cfg.Env {
			names = append(names, mapped)
		}
	}
	return names
}

// ArgvEnvironment returns identity assignments carried after argv[0]. They are
// removed from shell argv but restored as process environment, preserving the
// launcher's intended KEY=value context.
func ArgvEnvironment(cfg config.IdentityConfig, args []string) map[string]string {
	result := map[string]string{}
	if !cfg.ArgvEnvFallback {
		return result
	}
	for _, arg := range args[1:] {
		for _, name := range EnvNames(cfg) {
			if value, ok := strings.CutPrefix(arg, name+"="); ok {
				result[name] = value
			}
		}
	}
	return result
}

func envName(expression string) string {
	if strings.HasPrefix(expression, "${") && strings.HasSuffix(expression, "}") {
		return strings.TrimSuffix(strings.TrimPrefix(expression, "${"), "}")
	}
	return strings.TrimPrefix(expression, "$")
}
