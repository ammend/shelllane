package routing

import (
	"testing"

	"github.com/ammend/shelllane/internal/config"
	"github.com/ammend/shelllane/internal/identity"
)

func fixture() *config.Config {
	return &config.Config{
		Routes: []config.Route{
			{Match: config.Match{IdentityIn: []string{"admin"}}, Target: "local"},
			{Target: "pool"},
		},
		Nodes: map[string]config.Node{
			"local": {Connector: "local"},
			"a":     {Connector: "ssh"},
			"b":     {Connector: "ssh"},
		},
		NodePools: map[string]config.NodePool{
			"workers": {Nodes: []string{"a", "b"}, Strategy: "rendezvous-hash"},
		},
		Runtimes: map[string]config.Runtime{"host": {Type: "host"}},
		Targets: map[string]config.Target{
			"local": {Node: "local", Runtime: "host"},
			"pool":  {NodePool: "workers", Runtime: "host"},
		},
	}
}

func TestFirstMatchWinsAndDefaultCoversUsers(t *testing.T) {
	cfg := fixture()
	admin, err := Resolve(cfg, identity.Identity{Value: "admin", Hash: "1"})
	if err != nil || admin.NodeName != "local" || admin.RouteIndex != 0 {
		t.Fatalf("unexpected admin resolution: %#v %v", admin, err)
	}
	user, err := Resolve(cfg, identity.Identity{Value: "unlisted-user", Hash: "2"})
	if err != nil || (user.NodeName != "a" && user.NodeName != "b") || user.RouteIndex != 1 {
		t.Fatalf("unexpected user resolution: %#v %v", user, err)
	}
}

func TestRendezvousIsStable(t *testing.T) {
	cfg := fixture()
	var previous string
	for i := 0; i < 20; i++ {
		resolution, err := Resolve(cfg, identity.Identity{Value: "same", Hash: "stable-hash"})
		if err != nil {
			t.Fatal(err)
		}
		if previous != "" && previous != resolution.NodeName {
			t.Fatalf("node changed from %s to %s", previous, resolution.NodeName)
		}
		previous = resolution.NodeName
	}
}
