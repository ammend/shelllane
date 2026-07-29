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
	Value  string
	Hash   string
	Source string
}

func Resolve(cfg config.IdentityConfig, args []string, getenv func(string) string) (Identity, error) {
	fromEnv := strings.TrimSpace(getenv(cfg.Env))
	fromArgv := ""
	if cfg.ArgvEnvFallback {
		prefix := cfg.Env + "="
		for _, arg := range args[1:] {
			if !strings.HasPrefix(arg, prefix) {
				continue
			}
			candidate := strings.TrimPrefix(arg, prefix)
			if fromArgv != "" && fromArgv != candidate {
				return Identity{}, fmt.Errorf("%w: multiple %s values in argv", ErrConflict, cfg.Env)
			}
			fromArgv = candidate
		}
	}
	if fromEnv != "" && fromArgv != "" && fromEnv != fromArgv {
		return Identity{}, fmt.Errorf("%w: env and argv disagree", ErrConflict)
	}
	value, source := fromEnv, "env:"+cfg.Env
	if value == "" {
		value, source = fromArgv, "argv:"+cfg.Env
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
