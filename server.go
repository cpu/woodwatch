// Package woodwatch provides things
// TODO(@cpu): Write this package comment
package woodwatch

import (
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	"golang.org/x/net/icmp"

	"github.com/looplab/fsm"
)

const (
	// monitorCycle determines how often the server checks each source's last seen
	// field.
	monitorCycle = time.Second * 2

	// sourceTimeout is the max duration between ICMP messages before a source is
	// considered offline.
	sourceTimeout = time.Second * 5
)

var (
	// ServerAlreadyListeningErr is returned from Server.Listen when the Server is
	// already listening.
	ServerAlreadyListeningErr = errors.New("Listen() can only be called once")
	// ServerNotListeningErr is returned from Server.Close when the Server is not
	// listening.
	ServerNotListeningErr = errors.New("Close() must be called after Listen()")
	// EmptyListenAddressErr is returned from Server.Listen when the Server's
	// listen address is empty.
	EmptyListenAddressErr = errors.New("Listen address must not be empty")
	// TooFewSourcesErr is returned from NewServer when there aren't enough
	// sources provided.
	TooFewSourcesErr = errors.New("One or more Sources must be configured")
)

const (
	// StateUnknown represents a Source that hasn't been seen yet.
	StateUnknown string = "Unknown"
	// StateOffline represents a Source that hasn't been seen within the timeout
	// duration.
	StateOffline string = "Offline"
	// StateOnline represents a Source that is online and has been seen within the
	// timeout duration.
	StateOnline string = "Online"
)

// Source is a thing
// TODO(@cpu): Write the Source struct comment
type Source struct {
	// Name is the friendly display name for the Source. E.g. "Comcast", "Cocego".
	Name string
	// Network is the IP network that the given Source will send ICMP messages from.
	Network *net.IPNet

	// stateMachine implements state transitions for the source.
	stateMachine *fsm.FSM

	// lastSeen is the time the Source last sent an ICMP message.
	lastSeen time.Time
}

// NewSource does a thing.
// TODO(@cpu): Write the NewSource func comment.
func NewSource(name string, network string) (*Source, error) {
	_, parsedNetwork, err := net.ParseCIDR(network)
	if err != nil {
		return nil, err
	}
	fsm := fsm.NewFSM(
		StateUnknown,
		fsm.Events{
			{
				Name: "ping",
				Src:  []string{StateUnknown, StateOnline, StateOffline},
				Dst:  StateOnline,
			},
			{
				Name: "timeout",
				Src:  []string{StateUnknown, StateOnline, StateOffline},
				Dst:  StateOffline,
			},
		},
		fsm.Callbacks{},
	)
	return &Source{
		Name:         name,
		Network:      parsedNetwork,
		stateMachine: fsm,
	}, nil
}

// Server is a thing.
// TODO(@cpu): Write the Server struct comment
type Server struct {
	// log is the Server's log.Logger instance.
	log *log.Logger
	// listenAddress is the address used with icmp.ListenPacket in Listen to
	// create conn.
	listenAddress string
	// conn is created in Listen with icmp.ListenPacket.
	conn *icmp.PacketConn
	// sources is a list of configured traffic sources.
	sources []*Source
	// closeChan is used to signal a close to the monitoring goroutine.
	closeChan chan bool
}

// NewServer creates a thing
// TODO(@cpu): Write the NewServer func comment
func NewServer(log *log.Logger, addr string, sources []*Source) (*Server, error) {
	if addr == "" {
		return nil, EmptyListenAddressErr
	}
	if len(sources) == 0 {
		return nil, TooFewSourcesErr
	}
	return &Server{
		log:           log,
		listenAddress: addr,
		sources:       sources,
		closeChan:     make(chan bool, 1),
	}, nil
}

// Listen opens a PacketConn for the Server's listen address that will listen
// for ICMP packets. If Listen is called on a Server with an empty listen
// address it will return EmptyListeningAddressErr. If Listen is called more
// than once it will return ServerAlreadyListeningErr for all calls after the
// first.
func (s *Server) Listen() error {
	// Don't listen if there is no listen address
	if s.listenAddress == "" {
		return EmptyListenAddressErr
	}
	// Don't listen again if the server is already listening.
	if s.conn != nil {
		return ServerAlreadyListeningErr
	}

	// Start monitoring the last seen date of the traffic sources.
	go s.checkSourcesTicker()

	// Listen for packets on the server listenAddress
	var err error
	s.conn, err = icmp.ListenPacket("ip4:icmp", s.listenAddress)
	if err != nil {
		return err
	}
	s.log.Printf("server listening on ip4:icmp:%s\n", s.listenAddress)
	return s.readPacket()
}

// checkSourcesTicker will call checkSource for each of the Server's configured
// Sources once per monitorCycle until the Server's Close function is called.
func (s *Server) checkSourcesTicker() {
	ticker := time.NewTicker(monitorCycle)
	for {
		select {
		case _ = <-s.closeChan:
			s.log.Printf("stopping monitoring\n")
			return
		case _ = <-ticker.C:
			for _, src := range s.sources {
				s.checkSource(src)
			}
		}
	}
}

// checkSource checks if the given Source's last seen date is within an
// acceptable time range.
func (s *Server) checkSource(src *Source) {
	// if there is no src or statemachine return early
	if src == nil || src.stateMachine == nil {
		return
	}

	fmt.Printf("Source %q (last seen %q) is %q\n",
		src.Name, src.lastSeen, src.stateMachine.Current())

	// if the source has been seen less than sourceTimeout ago return early.
	now := time.Now()
	if now.Sub(src.lastSeen) < sourceTimeout {
		return
	}

	src.stateMachine.Event("timeout")
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
		s.updateSource(srcIP)
	}
	return nil
}

// updateSource iterates the Server's configured sources checking if any of the
// source networks contain the given address. The first matching source will
// have its last seen field set to the current time.
func (s *Server) updateSource(addr net.Addr) {
	parsedIP := net.ParseIP(addr.String())

	var matchedSource *Source
	for _, src := range s.sources {
		// TODO(@cpu): Add a debug flag to control printing the following commented
		// out debug Printf.
		/*
			fmt.Printf("Source %q Network %q contains %q - %v\n",
				src.Name, src.Network, parsedIP, src.Network.Contains(parsedIP))
		*/
		if src.Network.Contains(parsedIP) {
			matchedSource = src
			break
		}
	}

	if matchedSource == nil {
		s.log.Printf("no configured source matched %q", addr)
	} else {
		s.log.Printf("ip %q updated lastseen for %s\n", addr, matchedSource.Name)
		matchedSource.lastSeen = time.Now()
		matchedSource.stateMachine.Event("ping")
	}
}

// Close closes the Server's PacketConn and stops listening for ICMP messages on
// the Server's listen address. If Close is called before Listen it will return
// ServerNotListeningError.
func (s *Server) Close() error {
	if s.conn == nil {
		return ServerNotListeningErr
	}
	// Signal the monitoring go routine to close
	s.closeChan <- true
	// Close the underlying PacketConn. This will cause the `ReadFrom` in the
	// infinite for loop in `Serve` to immediately read a *net.OpError from using
	// the closed connection. Its a good enough "clean" exit mechanism for me!
	return s.conn.Close()
}
