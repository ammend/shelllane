package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Event struct {
	SchemaVersion  int       `json:"schema_version"`
	Time           time.Time `json:"time"`
	RequestID      string    `json:"request_id"`
	Event          string    `json:"event"`
	IdentityHash   string    `json:"identity_hash,omitempty"`
	IdentitySource string    `json:"identity_source,omitempty"`
	RouteIndex     int       `json:"route_index,omitempty"`
	Target         string    `json:"target,omitempty"`
	Node           string    `json:"node,omitempty"`
	Connector      string    `json:"connector,omitempty"`
	Runtime        string    `json:"runtime,omitempty"`
	Shell          string    `json:"shell,omitempty"`
	TTY            bool      `json:"tty"`
	Error          string    `json:"error,omitempty"`
}

func Write(path string, event Event) error {
	if path == "" {
		return nil
	}
	if event.SchemaVersion == 0 {
		event.SchemaVersion = 1
	}
	if event.Time.IsZero() {
		event.Time = time.Now().UTC()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create audit directory: %w", err)
	}
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode audit event: %w", err)
	}
	data = append(data, '\n')
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open audit log: %w", err)
	}
	defer file.Close()
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("write audit event: %w", err)
	}
	return nil
}
