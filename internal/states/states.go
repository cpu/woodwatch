// Package states provides peer state tracking.
package states

import "fmt"

const (
	down  = "Down"
	up    = "Up"
	maybe = "Maybe"
)

// PeerState is an interface describing a peer that responds to heartbeats by
// changing state. All PeerStates can represent their current state as
// a string. State changes can be marked as notable or not notable by returning
// a bool from the heartbeat function alongside the new state.
//
// TODO(@cpu): Make this an internal export
type PeerState interface {
	// Heartbeat is called every check cycle to indicate if the peer was seen
	// recently or not. A Heartbeat function should return the new PeerState
	// and a bool to indicate if change was noteworthy. Noteworthy-ness is defined
	// by the states themselves but generally should only be true for changes
	// from an intermediate state to the next state and not from an intermediate
	// state to a return state or a state to itself.
	Heartbeat(seen bool) (PeerState, bool)
	// String describes the PeerState's current state as a string.
	String() string
}

// NewPeer returns a PeerState that will transition states based on the
// provided thresholds. The returned PeerState represents a down connection that
// must receive downThreshold seen events to transition to up.
//
// TODO(@cpu): Make this an internal export
// TODO(@cpu): Describe lifecycle based on upThreshold/downThreshold
func NewPeer(upThreshold, downThreshold uint) PeerState {
	// NOTE(@cpu): By default we start in down state
	return downState{
		limits: limits{
			upThreshold:   upThreshold,
			downThreshold: downThreshold,
		},
	}
}

// limits is a struct for holding the upThreshold and downThreshold used by
// PeerStates.
type limits struct {
	// upThreshold is how many cycles the peer needs to be sending ICMP echo
	// requests without timeout before it is considered up.
	upThreshold uint
	// downThreshold is how many cycles the peer needs to miss sending ICMP echo
	// requests before it is considered down.
	downThreshold uint
}

// upState describes the state when the Peer is up and **has** been sending ICMP
// echo requests within the timeout reliably for some time. If the peer begins
// to timeout the upState will transition to the maybeDownState without
// considering it a notable event.
type upState struct {
	limits
}

// Heartbeat for upState stays in downState until there is a timeout, then it
// makes an unnotable transition to the maybeDownState.
func (s upState) Heartbeat(seen bool) (PeerState, bool) {
	if !seen {
		return maybeDownState(s.limits), false
	}
	return s, false
}

// String for upState returns up.
func (s upState) String() string {
	return up
}

// downState describes the state when the Peer is down and **has not** been
// sending ICMP echo requests within the timeout reliably for some time. If the
// peer begins to send echoes within the timeout again the downState will
// transition to the maybeUpState without considering it a notable event.
type downState struct {
	limits
}

// Heartbeat for downState stays in downState until the peer stops timing out,
// then it makes an unnotable transition to the maybeUpState.
func (s downState) Heartbeat(seen bool) (PeerState, bool) {
	if seen {
		return maybeUpState(s.limits), false
	}
	return s, false
}

// String for downState returns down.
func (s downState) String() string {
	return down
}

// maybeState describes a state when the Peer is maybe up or maybe down and
// we're counting up to some threshold, potentially resetting to a return state
// as an unnotable event, before finally considering the Peer in a new state as
// a notable event.
type maybeState struct {
	// name is the name of the state for use in String()
	name string
	// returnSeen indicates whether the maybe state should return to the
	// returnState on seen events or not seen events.
	returnSeen bool
	// returnState is a state to make an unnotable return to when the heartbeat
	// observation is equal to returnSeen. It is the "reset" state for the maybe
	// state in progress.
	returnState PeerState
	// nextState is a state to make a notable return to when there are threshold
	// correct observations in a row. It is the "target" state for the maybe state
	// in progress.
	nextState PeerState
	// count is how many correct observations for towards the threshold have been
	// seen so far.
	count uint
	// threshold is how many correct observations need to be seen before a notable
	// return to the nextState is made by the heartbeat function.
	threshold uint
}

// Heartbeat for the maybestate will reset to the returnState (without false
// notable bool) if the observation matches the returnSeen bool. Otherwise the
// count will be incremented. If the count is incremented greater than or equal
// to the threshold then the nextState is returned (with a true notable bool).
func (s maybeState) Heartbeat(seen bool) (PeerState, bool) {
	if (s.returnSeen && !seen) || (!s.returnSeen && seen) {
		return s.returnState, false
	}
	s.count++
	if s.count >= s.threshold {
		return s.nextState, true
	}
	return s, false
}

// String for a maybeState returns a description of the current state.
func (s maybeState) String() string {
	return fmt.Sprintf("%s (%d of %d)", s.name, s.count+1, s.threshold)
}

// maybeUpState constructs a maybeState that will reset to the downState without
// it being notable if timeouts occur. If no consecutive timeouts occur for
// upThreshold heartbeats then the maybeUpState's heartbeat will make a notable
// transition to the upState.
func maybeUpState(lim limits) maybeState {
	return maybeState{
		name:        fmt.Sprintf("%s %s", maybe, up),
		returnSeen:  true,
		returnState: downState{lim},
		nextState:   upState{lim},
		threshold:   lim.upThreshold,
	}
}

// maybeDownState constructs a maybeState that will reset to the upState without
// it being notable if a timeouts doesn't occur. If consecutive timeouts **do**
// occur for downThreshold heartbeats then the maybeDownState's heartbeat will
// make a notable transition to the downState.
func maybeDownState(lim limits) maybeState {
	return maybeState{
		name:        fmt.Sprintf("%s %s", maybe, down),
		returnSeen:  false,
		returnState: upState{lim},
		nextState:   downState{lim},
		threshold:   lim.downThreshold,
	}
}
