// Package woodwatch provides a server for monitoring peers by ICMP echo request
// keepalives.
package woodwatch

import (
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/cpu/woodwatch/internal/webhook"

	"golang.org/x/net/icmp"
)

var (
	// ErrServerAlreadyListening is returned from Server.Listen when the Server is
	// already listening.
	ErrServerAlreadyListening = errors.New("Listen() can only be called once")
	// ErrServerNotListening is returned from Server.Close when the Server is not
	// listening.
	ErrServerNotListening = errors.New("Close() must be called after Listen()")
	// ErrEmptyListenAddress is returned from Server.Listen when the Server's
	// listen address is empty.
	ErrEmptyListenAddress = errors.New("Listen address must not be empty")
	// ErrTooFewPeers is returned from NewServer when there aren't enough
	// peers provided.
	ErrTooFewPeers = errors.New("One or more Peers must be configured")
)

// Server is a struct for monitoring peers for keepalives received on
// a icmp.PacketConn.
type Server struct {
	// log is the Server's log.Logger instance.
	log *log.Logger
	// Verbose indicates whether all state change events should be logged and
	// dispatched or just notable ones.
	verbose bool
	// listenAddress is the address used with icmp.ListenPacket in Listen to
	// create conn.
	listenAddress string
	// conn is created in Listen with icmp.ListenPacket. ICMP messages are read
	// from conn.
	conn *icmp.PacketConn
	// peers is a list of configured peers.
	peers []*peer
	// closeChan is used to signal a close to the monitoring goroutine.
	closeChan chan bool
	// monitorCycle is the duration of time between checking if peers have timed out.
	monitorCycle time.Duration
	// peerTimeout is the duration of time the peer must have sent an ICMP echo
	// request within to be considered seen recently enough during a monitor
	// cycle.
	peerTimeout time.Duration
}

// NewServer constructs a woodwatch.Server for the given arguments and config or
// returns an error. The Server will not be running and listening for ICMP
// messages until it is explicitly started by calling Server.Listen()
func NewServer(
	log *log.Logger,
	verbose bool,
	addr string,
	c Config) (*Server, error) {
	if addr == "" {
		return nil, ErrEmptyListenAddress
	}
	if err := c.Valid(); err != nil {
		return nil, err
	}

	// Parse the monitor cycle and timeout durations.
	// NOTE(@cpu): It's safe to throw away potential error returns from
	// `time.ParseDuration` here because we checked c.Valid() and it verifies the
	// duration validities.
	monitorCycleDuration, _ := time.ParseDuration(c.MonitorCycle)
	peerTimeoutDuration, _ := time.ParseDuration(c.PeerTimeout)

	// Build peers from the PeerConfigs
	peers, err := loadPeers(c)
	if err != nil {
		return nil, err
	}

	// Log each of the peers and the initial state
	for _, p := range peers {
		log.Print(p)
	}

	return &Server{
		log:           log,
		verbose:       verbose,
		listenAddress: addr,
		peers:         peers,
		closeChan:     make(chan bool, 1),
		monitorCycle:  monitorCycleDuration,
		peerTimeout:   peerTimeoutDuration,
	}, nil
}

// Listen opens a PacketConn for the Server's listen address that will listen
// for ICMP packets. If Listen is called on a Server with an empty listen
// address it will return ErrEmptyListeningAddress. If Listen is called more
// than once it will return ErrServerAlreadyListening for all calls after the
// first.
func (s *Server) Listen() error {
	// Don't listen if there is no listen address
	if s.listenAddress == "" {
		return ErrEmptyListenAddress
	}
	// Don't listen again if the server is already listening.
	if s.conn != nil {
		return ErrServerAlreadyListening
	}

	// Start monitoring the last seen date of the peers.
	go s.checkPeersTicker()

	// Listen for packets on the server listenAddress
	var err error
	s.conn, err = icmp.ListenPacket("ip4:icmp", s.listenAddress)
	if err != nil {
		return err
	}
	s.log.Printf("server listening on ip4:icmp:%s\n", s.listenAddress)
	return s.readPacket()
}

// checkPeersTicker will call checkPeer for each of the Server's configured
// peers once per monitorCycle until the Server's Close function is called.
func (s *Server) checkPeersTicker() {
	ticker := time.NewTicker(s.monitorCycle)
	for {
		select {
		case <-s.closeChan:
			s.log.Printf("stopping monitoring\n")
			return
		case <-ticker.C:
			for _, src := range s.peers {
				s.checkPeer(src)
			}
		}
	}
}

// checkPeer checks if the given peer's last seen date is within an
// acceptable time range.
func (s *Server) checkPeer(p *peer) {
	// defensive check - shouldn't happen.
	if p == nil {
		return
	}
	p.lastSeenMu.RLock()
	defer p.lastSeenMu.RUnlock()

	// Check if the peer has been seen within the peerTimeout
	var seen bool
	if time.Since(p.lastSeen) < s.peerTimeout {
		seen = true
	}

	// Call the heartbeat function of the peer's current state with the
	// observation to produce a new state.
	oldState := p.state.String()
	var noteworthy bool
	p.state, noteworthy = p.state.Heartbeat(seen)
	newState := p.state.String()

	prettyLastSeen := p.lastSeen.Format("2006-01-02 03:04:05 PM -0700")
	event := webhook.Event{
		Timestamp: time.Now(),
		LastSeen:  p.lastSeen,
		Title:     fmt.Sprintf("Peer %s is %s", p.Name, newState),
		Text: fmt.Sprintf("%s (last seen %s) was previously %s and is now %s",
			p.Name, prettyLastSeen, oldState, newState),
		NewState:  newState,
		PrevState: oldState,
	}

	dispatch := func() {
		if p.Webhook != nil {
			go p.Webhook.Dispatch(event)
		}
		s.log.Print(event.Title)
	}

	if noteworthy {
		// If the event was noteworthy dispatch it.
		dispatch()
	} else if oldState != newState && s.verbose {
		// If the event was a state change and we're being verbose then dispatch it
		// even though it isn't noteworthy.
		dispatch()
	}
}

// readPacket will read an ICMP packet from the server's PacketConn connection
// and update the first source that matches the source IP of the sender.
func (s *Server) readPacket() error {
	// Process messages until an error from ReadFrom occurs. Notably this will
	// happen when the Server's Close function is called and the underlying
	// PacketConn is closed.
	for {
		var buf []byte
		_, srcIP, err := s.conn.ReadFrom(buf)
		if err != nil {
			return err
		}
		s.updatePeer(srcIP)
	}
}

// updatePeer iterates the Server's configured peers checking if any of the
// peer networks contain the given address. The first matching peer will
// have its last seen field set to the current time.
func (s *Server) updatePeer(addr fmt.Stringer) {
	parsedIP := net.ParseIP(addr.String())

	var matchedPeer *peer
	for _, p := range s.peers {
		if p.Network.Contains(parsedIP) {
			matchedPeer = p
			break
		}
	}

	if matchedPeer == nil {
		if s.verbose {
			s.log.Printf("no configured peer matched %q", addr)
		}
		return
	}

	if s.verbose {
		s.log.Printf("ip %q updated lastseen for %s\n", addr, matchedPeer.Name)
	}
	matchedPeer.lastSeenMu.Lock()
	defer matchedPeer.lastSeenMu.Unlock()
	matchedPeer.lastSeen = time.Now()
}

// Close closes the Server's PacketConn and stops listening for ICMP messages on
// the Server's listen address. If Close is called before Listen it will return
// ErrServerNotListening.
func (s *Server) Close() error {
	if s.conn == nil {
		return ErrServerNotListening
	}
	// Signal the monitoring go routine to close
	s.closeChan <- true
	// Close the underlying PacketConn. This will cause the `ReadFrom` in the
	// infinite for loop in `Serve` to immediately read a *net.OpError from using
	// the closed connection. Its a good enough "clean" exit mechanism for me!
	return s.conn.Close()
}
