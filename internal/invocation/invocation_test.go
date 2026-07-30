package invocation

import (
	"reflect"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	want := Request{
		Version: 1, RequestID: "req", IdentityHash: "abc", IdentityKey: "alice", Runtime: "host",
		StateMode: "isolated", ShellKind: "bash", Args: []string{"bash", "-lc", "echo ok"}, Cwd: "/tmp",
		TTY: true, Env: map[string]string{"LANG": "C.UTF-8"},
	}
	encoded, err := want.Encode()
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestRejectsIncompleteRequest(t *testing.T) {
	encoded, err := (Request{Version: 1}).Encode()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decode(encoded); err == nil {
		t.Fatal("expected invalid request")
	}
}
