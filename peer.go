package woodwatch

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/cpu/woodwatch/internal/states"
	"github.com/cpu/woodwatch/internal/webhook"
)

// peer is a struct describing a peer to be monitored.
type peer struct {
	// Name is the friendly display name for the peer . E.g. "Comcast", "Cocego :fire:".
	Name string
	// Network is the IP network that the peer is expected to send ICMP echo
	// request messages from.
	Network *net.IPNet
	// Webhook is an optional webhook to dispatch events to.
	Webhook *webhook.Hook
	// UpThreshold is how many cycles the peer needs to be sending ICMP echo
	// requests without timeout before it is considered up.
	upThreshold uint
	// DownThreshold is how many cycles the peer needs to miss sending ICMP echo
	// requests before it is considered down.
	downThreshold uint
	// lastSeenMu is a r/w mutex for controlling access to the lastSeen timestamp
	// for multiple goroutines.
	lastSeenMu *sync.RWMutex
	// lastSeen is the time the server last received an ICMP echo request from the
	// peer. Reading or writing this field must be done only after acquiring the
	// lastSeenMu.
	lastSeen time.Time
	// state is the peer's current PeerState
	state states.PeerState
}

// String returns a string representation of the peer.
func (p peer) String() string {
	return fmt.Sprintf("Peer %s - Network %s - State %s",
		p.Name, p.Network, p.state)
}

// NewPeer constructs a peer for the given arguments or returns an error.
func newPeer(
	name string,
	network string,
	upThreshold, downThreshold uint,
	hook *webhook.Hook) (*peer, error) {
	// parse the string representation of the CIDR network to ensure it is
	// valid.
	_, parsedNetwork, err := net.ParseCIDR(network)
	if err != nil {
		return nil, err
	}
	return &peer{
		Name:          name,
		Network:       parsedNetwork,
		Webhook:       hook,
		upThreshold:   upThreshold,
		downThreshold: downThreshold,
		// Build a state representation for the peer given the peer's thresholds
		state: states.NewPeer(upThreshold, downThreshold),
		// Construct a RW Mutex for this peer
		lastSeenMu: new(sync.RWMutex),
	}, nil
}

// loadPeers constructs a list of *woodwatch.Peer instances from the Config's
// individual PeerConfigs. The constructed Peer instances will use either their
// own override config values from each PeerConfig or the global values from the
// Config if no override was specified. Before constructing Peers Config.Valid()
// is called and any errors are returned, ensuring the config is sensible before
// trying to construct Peers.
func loadPeers(c Config) ([]*peer, error) {
	// Check the config is valid
	if err := c.Valid(); err != nil {
		return nil, err
	}

	// Build the Peers with the PeerConfigs
	var peers []*peer
	for _, pc := range c.Peers {
		// If there is an override UpThreshold use it, otherwise use the global
		upThreshold := pc.UpThreshold
		if upThreshold == 0 {
			upThreshold = c.UpThreshold
		}

		// If there is an override DownThreshold use it, otherwise use the global
		downThreshold := pc.DownThreshold
		if downThreshold == 0 {
			downThreshold = c.DownThreshold
		}

		// If there is an override WebHook use it, otherwise use the global
		hookURL := pc.Webhook
		if hookURL == "" {
			hookURL = c.Webhook
		}
		// Build a webhook pointer out of the URL if set
		var hook *webhook.Hook
		if hookURL != "" {
			h := webhook.Hook(hookURL)
			hook = &h
		}

		// Construct the peer and append it to the peers list
		peer, err := newPeer(pc.Name, pc.Network, upThreshold, downThreshold, hook)
		if err != nil {
			return nil, err
		}
		peers = append(peers, peer)
	}

	return peers, nil
}
