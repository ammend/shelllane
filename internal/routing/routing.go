package routing

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/ammend/shelllane/internal/config"
	"github.com/ammend/shelllane/internal/identity"
)

var ErrNoRoute = errors.New("route_not_found")

type Resolution struct {
	RouteIndex  int
	TargetName  string
	Target      config.Target
	NodeName    string
	Node        config.Node
	RuntimeName string
	Runtime     config.Runtime
}

func Resolve(cfg *config.Config, id identity.Identity) (Resolution, error) {
	for index, route := range cfg.Routes {
		if !matches(route.Match, id.Value) {
			continue
		}
		target := cfg.Targets[route.Target]
		nodeName := target.Node
		if target.NodePool != "" {
			pool := cfg.NodePools[target.NodePool]
			nodeName = rendezvous(id.Hash, target.NodePool, pool)
			if nodeName == "" {
				return Resolution{}, fmt.Errorf("node_unavailable: pool %s is empty", target.NodePool)
			}
		}
		return Resolution{
			RouteIndex:  index,
			TargetName:  route.Target,
			Target:      target,
			NodeName:    nodeName,
			Node:        cfg.Nodes[nodeName],
			RuntimeName: target.Runtime,
			Runtime:     cfg.Runtimes[target.Runtime],
		}, nil
	}
	return Resolution{}, ErrNoRoute
}

func matches(match config.Match, value string) bool {
	if match.Identity != "" {
		return match.Identity == value
	}
	if len(match.IdentityIn) != 0 {
		for _, candidate := range match.IdentityIn {
			if candidate == value {
				return true
			}
		}
		return false
	}
	return true
}

func rendezvous(identityHash, poolName string, pool config.NodePool) string {
	var selected string
	var highest uint64
	for _, node := range pool.Nodes {
		sum := sha256.Sum256([]byte(pool.HashSeed + "\x00" + poolName + "\x00" + identityHash + "\x00" + node))
		score := binary.BigEndian.Uint64(sum[:8])
		if selected == "" || score > highest {
			selected, highest = node, score
		}
	}
	return selected
}
