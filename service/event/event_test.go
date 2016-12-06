package event

import "testing"

func TestEventMatchOpts(t *testing.T) {
	var (
		disabled = false
		enabled  = true
		e        = &Event{
			Enabled: true,
			Owned:   true,
			Type:    "signal",
		}
		cases = map[*QueryOptions]bool{
			nil: true,
			&QueryOptions{Enabled: &disabled}:            false,
			&QueryOptions{Enabled: &enabled}:             true,
			&QueryOptions{Owned: &disabled}:              false,
			&QueryOptions{Owned: &enabled}:               true,
			&QueryOptions{Types: []string{"not-signal"}}: false,
			&QueryOptions{Types: []string{"signal"}}:     true,
		}
	)

	for opts, want := range cases {
		if have := e.MatchOpts(opts); have != want {
			t.Errorf("have %v, want %v", have, want)
		}
	}
}
