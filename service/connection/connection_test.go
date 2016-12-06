package connection

import "testing"

func TestConnectionMatchOpts(t *testing.T) {
	var (
		disabled = false
		enabled  = true
		c        = &Connection{
			Enabled: true,
			FromID:  1,
			State:   StateConfirmed,
			ToID:    2,
			Type:    TypeFriend,
		}
		cases = map[*QueryOptions]bool{
			nil: true,
			&QueryOptions{Enabled: &disabled}:              false,
			&QueryOptions{Enabled: &enabled}:               true,
			&QueryOptions{States: []State{StatePending}}:   false,
			&QueryOptions{States: []State{StateConfirmed}}: true,
			&QueryOptions{Types: []Type{TypeFollow}}:       false,
			&QueryOptions{Types: []Type{TypeFriend}}:       true,
		}
	)

	for opts, want := range cases {
		if have := c.MatchOpts(opts); have != want {
			t.Errorf("have %v, want %v", have, want)
		}
	}
}
