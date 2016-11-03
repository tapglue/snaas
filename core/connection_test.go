package core

import (
	"testing"

	"github.com/tapglue/snaas/service/connection"
)

func TestValidateConTransition(t *testing.T) {
	cases := map[*connection.Connection]*connection.Connection{
		// Different FromID
		{
			FromID: 1,
			ToID:   2,
			State:  connection.StatePending,
			Type:   connection.TypeFriend,
		}: {
			FromID: 2,
			ToID:   1,
			State:  connection.StatePending,
			Type:   connection.TypeFriend,
		},
		// Different ToID
		{
			FromID: 1,
			ToID:   2,
			State:  connection.StatePending,
			Type:   connection.TypeFriend,
		}: {
			FromID: 1,
			ToID:   3,
			State:  connection.StatePending,
			Type:   connection.TypeFriend,
		},
		// Different Type
		{
			FromID: 1,
			ToID:   2,
			State:  connection.StatePending,
			Type:   connection.TypeFriend,
		}: {
			FromID: 1,
			ToID:   2,
			State:  connection.StatePending,
			Type:   connection.TypeFollow,
		},
		// rejected -> confirmed
		{
			FromID: 1,
			ToID:   2,
			State:  connection.StateRejected,
			Type:   connection.TypeFriend,
		}: {
			FromID: 1,
			ToID:   2,
			State:  connection.StateConfirmed,
			Type:   connection.TypeFriend,
		},
		// rejected -> pending
		{
			FromID: 1,
			ToID:   2,
			State:  connection.StateRejected,
			Type:   connection.TypeFriend,
		}: {
			FromID: 1,
			ToID:   2,
			State:  connection.StatePending,
			Type:   connection.TypeFriend,
		},
		// confirmed -> pending
		{
			FromID: 1,
			ToID:   2,
			State:  connection.StateConfirmed,
			Type:   connection.TypeFriend,
		}: {
			FromID: 1,
			ToID:   2,
			State:  connection.StatePending,
			Type:   connection.TypeFriend,
		},
	}

	for old, new := range cases {
		err := validateConTransition(old, new)
		if have, want := err, ErrInvalidEntity; !IsInvalidEntity(err) {
			t.Errorf("have %v, want %v", have, want)
		}
	}
}
