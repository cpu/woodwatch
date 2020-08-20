package woodwatch

import (
	"strings"
	"testing"
)

func TestPeerConfigValid(t *testing.T) {
	testCases := []struct {
		Name          string
		InputName     string
		InputNetwork  string
		ExpectedError error
	}{
		{
			Name:          "Empty peer name",
			ExpectedError: ErrNoPeerName,
		},
		{
			Name:          "Empty network",
			InputName:     "not-empty",
			ExpectedError: ErrNoPeerNetwork,
		},
		{
			Name:         "Valid peer",
			InputName:    "not-empty",
			InputNetwork: "not-empty",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			p := PeerConfig{
				Name:    tc.InputName,
				Network: tc.InputNetwork,
			}
			if err := p.Valid(); err != tc.ExpectedError {
				t.Errorf("expected Valid() to return %v, got %v",
					tc.ExpectedError, err)
			}
		})
	}
}

func TestConfigValid(t *testing.T) {
	validPeers := []PeerConfig{
		{
			Name:    "test",
			Network: "test",
		},
	}
	testCases := []struct {
		Name                       string
		Peers                      []PeerConfig
		MonitorCycle               string
		PeerTimeout                string
		ExpectedErrorMessagePrefix string
	}{
		{
			Name:                       "No peers",
			ExpectedErrorMessagePrefix: ErrTooFewPeers.Error(),
		},
		{
			Name:                       "Invalid peer",
			Peers:                      []PeerConfig{{}},
			ExpectedErrorMessagePrefix: ErrNoPeerName.Error(),
		},
		{
			Name:                       "Invalid monitor cycle",
			Peers:                      validPeers,
			MonitorCycle:               "aaaa",
			ExpectedErrorMessagePrefix: "time: invalid duration",
		},
		{
			Name:                       "Invalid peer timeout",
			Peers:                      validPeers,
			MonitorCycle:               "1m",
			PeerTimeout:                "aaaa",
			ExpectedErrorMessagePrefix: "time: invalid duration",
		},
		{
			Name:         "Valid config",
			MonitorCycle: "1m",
			PeerTimeout:  "10s",
			Peers:        validPeers,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			c := Config{
				Peers:        tc.Peers,
				MonitorCycle: tc.MonitorCycle,
				PeerTimeout:  tc.PeerTimeout,
			}
			if err := c.Valid(); err != nil && tc.ExpectedErrorMessagePrefix == "" {
				t.Errorf("expected Valid() to return nil err, got %v", err)
			} else if err == nil && tc.ExpectedErrorMessagePrefix != "" {
				t.Errorf("expected Valid() to return error starting with %q got nil",
					tc.ExpectedErrorMessagePrefix)
			} else if err != nil &&
				!strings.HasPrefix(err.Error(), tc.ExpectedErrorMessagePrefix) {
				t.Errorf("expected Valid() to return error starting with %q, got %v",
					tc.ExpectedErrorMessagePrefix, err)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	var exampleConfig = `
	{
		"UpThreshold": 10,
		"DownThreshold": 10,
		"Peers": [
			{
				"Name": "ISP A",
				"Network": "8.8.8.0/24",
				"DownThreshold": 2
			},
			{
				"Name": "ISP B",
				"Network": "1.1.1.0/24",
				"UpThreshold": 2
			},
			{
				"Name": "ISP C",
				"Network": "192.168.1.0/24",
				"UpThreshold": 3,
				"DownThreshold": 4
			}
		]
	}
`
	testCases := []struct {
		Name           string
		Input          []byte
		ExpectedErrMsg string
	}{
		{
			Name:           "Invalid input",
			Input:          []byte("{"),
			ExpectedErrMsg: "unexpected end of JSON input",
		},
		{
			Name:  "Empty input",
			Input: []byte("{}"),
		},
		{
			Name:  "Example config",
			Input: []byte(exampleConfig),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			_, err := LoadConfig(tc.Input)
			if err != nil && tc.ExpectedErrMsg == "" {
				t.Fatalf("Expected no err, got %v", err)
			} else if err == nil && tc.ExpectedErrMsg != "" {
				t.Fatalf("Expected err %q got nil", tc.ExpectedErrMsg)
			} else if err != nil && err.Error() != tc.ExpectedErrMsg {
				t.Fatalf("Expected err %q got %q", tc.ExpectedErrMsg, err.Error())
			}
		})
	}
}
