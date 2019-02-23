package woodwatch

import (
	"testing"
)

// TestNewPeerError tests that calling newPeer with a bad CIDR network
// string will produce an error.
func TestNewPeerError(t *testing.T) {
	if _, err := newPeer("bad CIDR", "", 0, 0, nil); err == nil {
		t.Fatalf("expected err from newPeer with bad CIDR, got nil\n")
	}
}

// TestPeerString tests that calling peer.String() returns the expected string
// description of the peer.
func TestPeerString(t *testing.T) {
	p, err := newPeer(
		"TestPeer",
		"192.168.1.0/24",
		0, 0, nil)
	if err != nil {
		t.Fatalf("newPeer returned %v expected nil", err)
	}
	expected := "Peer TestPeer - Network 192.168.1.0/24 - State Down"
	if p.String() != expected {
		t.Errorf("Expected p.String() to be %q was %q", expected, p.String())
	}
}

func TestLoadPeers(t *testing.T) {
	exampleHookA := "example.org"
	exampleHookB := "example.com"

	type expectedPeer struct {
		Name          string
		UpThreshold   uint
		DownThreshold uint
		Webhook       string
	}
	testCases := []struct {
		Name          string
		Conf          Config
		ExpectedError error
		ExpectedPeers []expectedPeer
	}{
		{
			Name:          "Invalid config",
			Conf:          Config{},
			ExpectedError: ErrTooFewPeers,
		},
		{
			Name: "Global config",
			Conf: Config{
				UpThreshold:   10,
				DownThreshold: 11,
				MonitorCycle:  "2s",
				PeerTimeout:   "2s",
				Webhook:       exampleHookA,
				Peers: []PeerConfig{
					{
						Name:    "First",
						Network: "192.168.1.0/24",
					},
					{
						Name:    "Second",
						Network: "192.168.1.0/24",
					},
				},
			},
			ExpectedPeers: []expectedPeer{
				{
					Name:          "First",
					UpThreshold:   10,
					DownThreshold: 11,
					Webhook:       exampleHookA,
				},
				{
					Name:          "Second",
					UpThreshold:   10,
					DownThreshold: 11,
					Webhook:       exampleHookA,
				},
			},
		},
		{
			Name: "Peer override config",
			Conf: Config{
				UpThreshold:   10,
				DownThreshold: 11,
				MonitorCycle:  "2s",
				PeerTimeout:   "2s",
				Webhook:       exampleHookA,
				Peers: []PeerConfig{
					{
						Name:          "First",
						Network:       "192.168.1.0/24",
						DownThreshold: 99,
						Webhook:       exampleHookB,
					},
					{
						Name:        "Second",
						Network:     "192.168.1.0/24",
						UpThreshold: 128,
					},
				},
			},
			ExpectedPeers: []expectedPeer{
				{
					Name:          "First",
					UpThreshold:   10,
					DownThreshold: 99,
					Webhook:       exampleHookB,
				},
				{
					Name:          "Second",
					UpThreshold:   128,
					DownThreshold: 11,
					Webhook:       exampleHookA,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			peers, err := loadPeers(tc.Conf)
			if err != tc.ExpectedError {
				t.Fatalf("expected loadPeers() to return err %v got %v",
					tc.ExpectedError, err)
			}
			if len(peers) != len(tc.ExpectedPeers) {
				t.Fatalf("expected %d peers from loadPeers(), got %d",
					len(tc.ExpectedPeers), len(peers))
			}
			for i, p := range peers {
				expected := tc.ExpectedPeers[i]
				if p.Name != expected.Name {
					t.Errorf("expected %dth peer to have name %q had %q",
						i, expected.Name, p.Name)
				}
				if p.upThreshold != expected.UpThreshold {
					t.Errorf("expected %dth peer to have upThreshold %d had %d",
						i, expected.UpThreshold, p.upThreshold)
				}
				if p.downThreshold != expected.DownThreshold {
					t.Errorf("expected %dth peer to have downThreshold %d had %d",
						i, expected.DownThreshold, p.downThreshold)
				}
				if string(*p.Webhook) != expected.Webhook {
					t.Errorf("expected %dth peer to have Webhook %s had %s",
						i, expected.Webhook, string(*p.Webhook))
				}
			}
		})
	}
}
