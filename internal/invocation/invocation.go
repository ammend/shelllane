package invocation

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

type Request struct {
	Version      int               `json:"version"`
	RequestID    string            `json:"request_id"`
	IdentityHash string            `json:"identity_hash"`
	Runtime      string            `json:"runtime"`
	StateMode    string            `json:"state_mode,omitempty"`
	ShellKind    string            `json:"shell_kind"`
	Args         []string          `json:"args"`
	Cwd          string            `json:"cwd"`
	TTY          bool              `json:"tty"`
	Env          map[string]string `json:"env,omitempty"`
}

func (r Request) Encode() (string, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func Decode(encoded string) (Request, error) {
	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return Request{}, fmt.Errorf("decode remote request: %w", err)
	}
	var request Request
	if err := json.Unmarshal(data, &request); err != nil {
		return Request{}, fmt.Errorf("parse remote request: %w", err)
	}
	if request.Version != 1 || request.IdentityHash == "" || request.Runtime == "" || request.ShellKind == "" || len(request.Args) == 0 {
		return Request{}, fmt.Errorf("invalid remote request")
	}
	return request, nil
}
