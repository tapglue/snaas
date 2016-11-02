package core

import (
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/tapglue/api/service/connection"
	"github.com/tapglue/api/service/event"
	"github.com/tapglue/api/service/user"
)

func TestAffiliation(t *testing.T) {
	var (
		from = uint64(123)
		to   = uint64(321)
		a    = affiliations{
			&connection.Connection{
				FromID: from,
				ToID:   to,
				Type:   connection.TypeFollow,
			}: &user.User{
				ID: to,
			},
			&connection.Connection{
				FromID: to,
				ToID:   from,
				Type:   connection.TypeFollow,
			}: &user.User{
				ID: from,
			},
			&connection.Connection{
				FromID: from,
				ToID:   to,
				Type:   connection.TypeFriend,
			}: &user.User{
				ID: from,
			},
		}
	)

	if have, want := len(a.connections()), 3; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	if have, want := len(a.followers(from)), 1; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	if have, want := len(a.followings(from)), 1; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	if have, want := len(a.filterFollowers(from)), 2; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	if have, want := len(a.friends(from)), 1; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	if have, want := len(a.userIDs()), 2; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	if have, want := len(a.users()), 2; have != want {
		t.Errorf("have %v, want %v", have, want)
	}
}

func TestCollect(t *testing.T) {
	es, err := collect(
		testSourceLen(2),
		testSourceLen(7),
		testSourceLen(4),
	)
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(es), 13; have != want {
		t.Errorf("have %v, want %v", have, want)
	}
}

func TestCollectError(t *testing.T) {
	_, err := collect(testSourceError)
	if err == nil {
		t.Error("want collect to error")
	}
}

func TestConditionDuplicate(t *testing.T) {
	es, err := testSourceLen(10)()
	if err != nil {
		t.Fatal(err)
	}

	es = append(es, &event.Event{
		ID: 5,
	})

	es = filter(es, conditionDuplicate())

	if have, want := len(es), 10; have != want {
		t.Errorf("have %v, want %v", have, want)
	}
}

func TestConditionObjectMissing(t *testing.T) {
	es, err := testSourceLen(10)()
	if err != nil {
		t.Fatal(err)
	}

	pm := PostMap{
		1: {},
		6: {},
	}

	es = filter(es, conditionPostMissing(pm))

	if have, want := len(es), 3; have != want {
		t.Errorf("have %v, want %v", have, want)
	}
}

func TestFilter(t *testing.T) {
	es, err := testSourceLen(10)()
	if err != nil {
		t.Fatal(err)
	}

	es = filter(es, testConditionEven)

	if have, want := len(es), 5; have != want {
		t.Errorf("have %v, want %v", have, want)
	}
}

func TestSourceConnection(t *testing.T) {
	var (
		from = uint64(rand.Int63())
		to   = uint64(rand.Int63())
		cs   = connection.List{
			{
				State:     connection.StateConfirmed,
				Type:      connection.TypeFriend,
				FromID:    from,
				ToID:      to,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			},
			{
				State:     connection.StatePending,
				Type:      connection.TypeFollow,
				FromID:    from,
				ToID:      to,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			},
			{
				State:     connection.StateRejected,
				Type:      connection.TypeFollow,
				FromID:    from,
				ToID:      to,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			},
			{
				State:     connection.StateConfirmed,
				Type:      connection.TypeFollow,
				FromID:    from,
				ToID:      to,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			},
			{
				State:     connection.StateConfirmed,
				Type:      connection.TypeFollow,
				FromID:    to,
				ToID:      from,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			},
		}
	)

	es, err := sourceConnection(cs, event.QueryOptions{})()
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(es), 3; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	if have, want := es[0].Type, event.TypeFollow; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	if have, want := es[2].Type, event.TypeFriend; have != want {
		t.Errorf("have %v, want %v", have, want)
	}
}

func testConditionEven(idx int, event *event.Event) bool {
	return idx%2 == 0
}

func testSourceLen(n int) source {
	return func() (event.List, error) {
		es := event.List{}

		for i := 0; i < n; i++ {
			es = append(es, &event.Event{
				ID:       uint64(i + 1),
				ObjectID: uint64(i),
				Target: &event.Target{
					ID:   strconv.FormatUint(uint64(i+1), 10),
					Type: event.TargetUser,
				},
				UserID: uint64(i + 1),
			})
		}

		return es, nil
	}
}

func testSourceError() (event.List, error) {
	return nil, fmt.Errorf("something went wrong")
}
