// Package webhook describes woodwatch events and webhook URLs they can be
// POSTed to.
package webhook

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime"
	"time"
)

var (
	// ErrEmptyEventTitle is returned from Hook.Dispatch when the provided Event
	// has no title.
	ErrEmptyEventTitle = errors.New("Event Title must not be empty")
	// ErrEmptyNewState is returned from Hook.Dispatch when the provided Event
	// has no NewState.
	ErrEmptyNewState = errors.New("Event NewState must not be empty")
	// ErrEmptyPrevState is returned from Hook.Dispatch when the provided Event
	// has no PrevState.
	ErrEmptyPrevState = errors.New("Event PrevState must not be empty")
)

// Hook is a URL for Event's to be POSTed to as JSON objects.
type Hook string

// Event is a struct for describing a state change event observed for
// a woodwatch Peer.
type Event struct {
	// Title is the title of the event.
	Title string `json:"title"`
	// Text is a textual description of the event.
	Text string `json:"text"`
	// Timestamp is when the event occurred
	Timestamp time.Time `json:"timestamp"`
	// LastSeen is when the Peer last sent an ICMP echo request that was received
	// by the woodwatch server.
	LastSeen time.Time `json:"lastSeen"`
	// NewState is the state the Peer is now in.
	NewState string `json:"newState"`
	// PrevState is the state the Peer was previously in.
	PrevState string `json:"prevState"`
}

// Valid checks that an Event has a Title, a NewState and a PrevState. Otherwise
// ErrEmptyTitle, ErrEmptyNewState or ErrEmptyPrevState is returned.
func (e Event) Valid() error {
	if e.Title == "" {
		return ErrEmptyEventTitle
	}
	if e.NewState == "" {
		return ErrEmptyNewState
	}
	if e.PrevState == "" {
		return ErrEmptyPrevState
	}
	return nil
}

// Dispatch POSTs the provided Event to the Hook URL as a JSON object. Errors
// with the event, marshaling, POSTing, or from the server are presently
// ignored.
//
// TODO(@cpu): Figure out error handling for things that go wrong during dispatch.
func (h Hook) Dispatch(e Event) {
	if err := e.Valid(); err != nil {
		return
	}

	eventBytes, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return
	}

	req, err := http.NewRequest("POST", string(h), bytes.NewBuffer(eventBytes))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf(
		"cpu.woodwatch 0.0.1 (%s; %s)",
		runtime.GOOS, runtime.GOARCH))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	_, _ = ioutil.ReadAll(resp.Body)
}
