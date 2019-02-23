package woodwatch

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"time"
)

var (
	// ErrNoPeerName is returned from PeerConfig.Valid() when the PeerConfig doesn't
	// have a Name.
	ErrNoPeerName = errors.New("All PeerConfigs must have a Name")
	// ErrNoPeerNetwork is returned from PeerConfig.Valid() when the PeerConfig
	// doesn't have a Network.
	ErrNoPeerNetwork = errors.New("All PeerConfigs must have a Network")
)

// PeerConfig is a struct holding configuration related to monitoring a Peer.
type PeerConfig struct {
	// Name is the name of the peer. Supports :slack: emoji!
	Name string
	// Network is the string representation of a CIDR network. To be considered up
	// the peer must periodically send ICMP echo requests from a host within this
	// CIDR network. E.g. "192.168.1.0/24".
	Network string
	// UpThreshold is how many cycles the peer needs to be sending ICMP echo
	// requests without timeout before it is considered up. If zero the global
	// UpThreshold is used.
	UpThreshold uint
	// DownThreshold is how many cycles the peer needs to miss sending ICMP echo
	// requests before it is considered down. If zero the global DownThreshold is
	// used.
	DownThreshold uint
	// Webhook is an optional webhook to be POSTed for events. If not provided the
	// global Webhook is used.
	Webhook string
}

// Valid checks that a PeerConfig has a Name and Network or returns
// ErrNoPeerName/ErrNoPeerNetwork if the PeerConfig is not valid.
func (pc PeerConfig) Valid() error {
	if pc.Name == "" {
		return ErrNoPeerName
	}
	if pc.Network == "" {
		return ErrNoPeerNetwork
	}
	return nil
}

// Config describes the global woodwatch configuration and the peers to be
// monitored.
type Config struct {
	// UpThreshold is how many cycles a peer needs to be sending ICMP echo
	// requests without timeout before it is considered up. Individual PeerConfigs
	// may set their own UpThreshold.
	UpThreshold uint
	// DownThreshold is how many cycles a peer needs to miss sending ICMP echo
	// requests before it is considered down. Individual PeerConfigs may set their
	// own DownThreshold.
	DownThreshold uint
	// MonitorCycle is a mandatory string describing the duration between checking
	// if a Peer has sent ICMP echo requests within the PeerTimeout. E.g. "4s",
	// "1m".
	MonitorCycle string
	// PeerTimeout is a mandatory string describing the duration within a Peer
	// must have sent ICMP echo requests to be considered seen recently during
	// a monitor cycle. E.g. "8s", "2m".
	PeerTimeout string
	// Webhook is an optional webhook URL to be POSTed for events. Individual
	// PeerConfigs may set their own Webhook.
	Webhook string
	// Peers is one or more PeerConfigs describing a peer to be monitored.
	Peers []PeerConfig
}

// Valid checks that a woodwatch Config is valid. If no peers are specified
// ErrTooFewPeers is returned. Each of the Peers specified will have their
// PeerConfig.Valid() function called and any errors will be returned. The
// MonitorCycle and PeerTimeout will both be parsed as time.Duration instances
// and any errors will be returned.
func (c Config) Valid() error {
	if len(c.Peers) == 0 {
		return ErrTooFewPeers
	}
	for _, pc := range c.Peers {
		if err := pc.Valid(); err != nil {
			return err
		}
	}
	if _, err := time.ParseDuration(c.MonitorCycle); err != nil {
		return err
	}
	if _, err := time.ParseDuration(c.PeerTimeout); err != nil {
		return err
	}
	return nil
}

// LoadConfig loads a woodwatch.Config from the given data bytes.
func LoadConfig(data []byte) (Config, error) {
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return c, err
	}
	return c, nil
}

// LoadConfigFile reads the data bytes from the file located at the provided
// filename and uses LoadConfig to return a woodwatch.Config from the file
// bytes.
func LoadConfigFile(filename string) (Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return Config{}, err
	}
	return LoadConfig(data)
}
