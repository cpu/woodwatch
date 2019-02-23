package woodwatch

import (
	"testing"

	"golang.org/x/net/icmp"
)

// TestListenErrors tests that calling Listen() in invalid ways generates the
// correct errors.
func TestListenErrors(t *testing.T) {
	testCases := []struct {
		Name        string
		Addr        string
		Conn        *icmp.PacketConn
		ExpectedErr error
	}{
		{
			Name:        "Empty listen address",
			ExpectedErr: ErrEmptyListenAddress,
		},
		{
			Name:        "Non-nil PacketConn",
			Addr:        "whatever",
			Conn:        &icmp.PacketConn{},
			ExpectedErr: ErrServerAlreadyListening,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			s := Server{
				listenAddress: tc.Addr,
				conn:          tc.Conn,
			}
			if err := s.Listen(); err == nil {
				t.Fatalf("expected err from Listen(), got nil\n")
			} else if err != tc.ExpectedErr {
				t.Errorf("expected err to be %v, was %v\n", tc.ExpectedErr, err)
			}
		})
	}
}

// TestCloseError tests that calling Close() on an already closed instance fails
// with the expected error.
func TestCloseError(t *testing.T) {
	// Create a server with a nil conn.
	s := Server{}
	// Calling Close() with a nil PacketConn should error and it should be an
	// instance of ErrServerNotListening
	if err := s.Close(); err == nil {
		t.Fatalf("expected err from Close(), got nil\n")
	} else if err != ErrServerNotListening {
		t.Errorf("expected err to be ErrServerNotListening, was %v\n", err)
	}
}

// TestNewServerError tests that calling NewServer with invalid args fails with
// the expected errors.
func TestNewServerError(t *testing.T) {
	testCases := []struct {
		Name          string
		ListenAddress string
		ExpectedError error
	}{
		{
			Name:          "Empty listen address",
			ExpectedError: ErrEmptyListenAddress,
		},
		{
			Name:          "Too few peers",
			ListenAddress: "whatever",
			ExpectedError: ErrTooFewPeers,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			if _, err := NewServer(nil, false, tc.ListenAddress, Config{}); err == nil {
				t.Fatalf("expected err from NewServer(), got nil\n")
			} else if err != tc.ExpectedError {
				t.Errorf("expected err to be %v, was %v\n", tc.ExpectedError, err)
			}
		})
	}
}
