package session

import "testing"

func TestValidate(t *testing.T) {
	ss := List{
		{},
		{ID: "1234"},
	}

	for _, s := range ss {
		if have, want := s.Validate(), ErrInvalidSession; !IsInvalidSession(have) {
			t.Errorf("have %v, want %v", have, want)
		}
	}
}
