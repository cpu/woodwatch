package states

import (
	"fmt"
	"testing"
)

// TestNewPeer tests that the NewPeer function returns the correct downState
// with the provided thresholds.
func TestNewPeer(t *testing.T) {
	testCases := []struct {
		Up            uint
		Down          uint
		ExpectedState downState
	}{
		{
			Up:   1,
			Down: 10,
			ExpectedState: downState{
				limits{
					upThreshold:   1,
					downThreshold: 10,
				},
			},
		},
		{
			Up:   99,
			Down: 0,
			ExpectedState: downState{
				limits{
					upThreshold:   99,
					downThreshold: 0,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(
			fmt.Sprintf("NewPeer(%d, %d)", tc.Up, tc.Down),
			func(t *testing.T) {
				state := NewPeer(tc.Up, tc.Down)
				if _, ok := state.(downState); !ok {
					t.Fatalf("expected NewPeer(%d,%d) to be downState not %T",
						tc.Up, tc.Down, state)
				}
			})
	}
}

// statePair structs describe an observation and the expected next state and
// noteworthy bool pair.
type statePair struct {
	observation bool
	newState    string
	noteworthy  bool
}

// TestPeerStates tests the Up/Down PeerStates and their
// transitions/noteworthyness.
func TestPeerStates(t *testing.T) {
	maybeDesc := func(x string, i, max uint) string {
		return fmt.Sprintf("%s %s (%d of %d)", maybe, x, i, max)
	}

	lim := limits{
		upThreshold:   3,
		downThreshold: 2,
	}

	testCases := []struct {
		Name         string
		InitialState PeerState
		Expected     []statePair
	}{
		{
			Name:         "Up stays up",
			InitialState: upState{lim},
			Expected: []statePair{
				{true, up, false},
				{true, up, false},
				{true, up, false},
			},
		},
		{
			Name:         "Up stays up within threshold",
			InitialState: upState{lim},
			Expected: []statePair{
				{true, up, false},
				{false, maybeDesc(down, 1, lim.downThreshold), false},
				{true, up, false},
				{false, maybeDesc(down, 1, lim.downThreshold), false},
				{true, up, false},
			},
		},
		{
			Name:         "Up goes down after threshold",
			InitialState: upState{lim},
			Expected: []statePair{
				{true, up, false},
				{false, maybeDesc(down, 1, lim.downThreshold), false},
				{false, maybeDesc(down, 2, lim.downThreshold), false},
				{false, down, true},
			},
		},
		{
			Name:         "Down stays down",
			InitialState: downState{lim},
			Expected: []statePair{
				{false, down, false},
				{false, down, false},
				{false, down, false},
			},
		},
		{
			Name:         "Down stays down within threshold",
			InitialState: downState{lim},
			Expected: []statePair{
				{false, down, false},
				{true, maybeDesc(up, 1, lim.upThreshold), false},
				{false, down, false},
				{true, maybeDesc(up, 1, lim.upThreshold), false},
				{true, maybeDesc(up, 2, lim.upThreshold), false},
				{false, down, false},
			},
		},
		{
			Name:         "Down goes up after threshold",
			InitialState: downState{lim},
			Expected: []statePair{
				{false, down, false},
				{true, maybeDesc(up, 1, lim.upThreshold), false},
				{true, maybeDesc(up, 2, lim.upThreshold), false},
				{true, maybeDesc(up, 3, lim.upThreshold), false},
				{true, up, true},
			},
		},
		{
			Name:         "Full down to up to down cycle",
			InitialState: downState{lim},
			Expected: []statePair{
				{true, maybeDesc(up, 1, lim.upThreshold), false},
				{true, maybeDesc(up, 2, lim.upThreshold), false},
				{true, maybeDesc(up, 3, lim.upThreshold), false},
				{true, up, true},
				{false, maybeDesc(down, 1, lim.downThreshold), false},
				{false, maybeDesc(down, 2, lim.downThreshold), false},
				{false, down, true},
			},
		},
		{
			Name:         "Full up to down to up cycle",
			InitialState: upState{lim},
			Expected: []statePair{
				{false, maybeDesc(down, 1, lim.downThreshold), false},
				{false, maybeDesc(down, 2, lim.downThreshold), false},
				{false, down, true},
				{true, maybeDesc(up, 1, lim.upThreshold), false},
				{true, maybeDesc(up, 2, lim.upThreshold), false},
				{true, maybeDesc(up, 3, lim.upThreshold), false},
				{true, up, true},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			var noteworthy bool
			state := tc.InitialState
			for i, pair := range tc.Expected {
				state, noteworthy = state.Heartbeat(pair.observation)
				if state.String() != pair.newState {
					t.Errorf("after observation %d (%v) state was %q not %q",
						i, pair.observation, state, pair.newState)
				}
				if noteworthy != pair.noteworthy {
					t.Errorf("after observation %d (%v) noteworthy bool was %v not %v",
						i, pair.observation, noteworthy, pair.noteworthy)
				}
			}
		})
	}
}
