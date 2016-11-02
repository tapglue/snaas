package connection

import (
	"math/rand"
	"reflect"
	"testing"
	"time"
)

type prepareFunc func(t *testing.T, namespace string) Service

func testServiceCount(t *testing.T, p prepareFunc) {
	var (
		namespace = "service_count"
		service   = p(t, namespace)
		from      = uint64(rand.Int63())
		to        = uint64(rand.Int63())
		disabled  = false
		start     = time.Now()
	)

	for _, c := range testList(from, to, start) {
		_, err := service.Put(namespace, c)
		if err != nil {
			t.Fatal(err)
		}
	}

	cases := map[*QueryOptions]int{
		&QueryOptions{}: 36,
		&QueryOptions{Before: start.Add(-(time.Hour + time.Minute))}: 10,
		&QueryOptions{Enabled: &disabled}:                            5,
		&QueryOptions{FromIDs: []uint64{from}}:                       12,
		&QueryOptions{States: []State{StateConfirmed}}:               18,
		&QueryOptions{ToIDs: []uint64{to}}:                           13,
		&QueryOptions{Types: []Type{TypeFriend}}:                     29,
	}

	for opts, want := range cases {
		have, err := service.Count(namespace, *opts)
		if err != nil {
			t.Fatal(err)
		}

		if have != want {
			t.Errorf("have %v, want %v", have, want)
		}
	}
}

func testList(from, to uint64, start time.Time) List {
	cs := List{}

	for i := 0; i < 7; i++ {
		cs = append(cs, &Connection{
			Enabled: true,
			FromID:  from,
			State:   StateConfirmed,
			ToID:    uint64(rand.Int63()),
			Type:    TypeFollow,
		})
	}

	for i := 0; i < 5; i++ {
		cs = append(cs, &Connection{
			Enabled: false,
			FromID:  from,
			State:   StatePending,
			ToID:    uint64(rand.Int63()),
			Type:    TypeFriend,
		})
	}

	for i := 0; i < 13; i++ {
		cs = append(cs, &Connection{
			Enabled: true,
			FromID:  uint64(rand.Int63()),
			State:   StateRejected,
			ToID:    to,
			Type:    TypeFriend,
		})
	}

	for i := 1; i < 12; i++ {
		cs = append(cs, &Connection{
			Enabled:   true,
			FromID:    uint64(rand.Int63()),
			State:     StateConfirmed,
			ToID:      uint64(rand.Int63()),
			Type:      TypeFriend,
			CreatedAt: start.Add(-(time.Duration(i) * time.Hour)),
			UpdatedAt: start.Add(-(time.Duration(i) * time.Hour)),
		})
	}

	return cs
}

func testServicePut(t *testing.T, p prepareFunc) {
	var (
		namespace = "service_put"
		service   = p(t, namespace)
		con       = &Connection{
			Enabled: true,
			FromID:  uint64(rand.Int63()),
			ToID:    uint64(rand.Int63()),
			Type:    TypeFollow,
			State:   StatePending,
		}
	)

	created, err := service.Put(namespace, con)
	if err != nil {
		t.Fatal(err)
	}

	cs, err := service.Query(namespace, QueryOptions{
		Enabled: &con.Enabled,
		FromIDs: []uint64{con.FromID},
		Types:   []Type{con.Type},
		ToIDs:   []uint64{con.ToID},
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(cs), 1; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	if have, want := cs[0], created; !reflect.DeepEqual(have, want) {
		t.Errorf("have %v, want %v", have, want)
	}

	created.State = StateConfirmed

	updated, err := service.Put(namespace, created)
	if err != nil {
		t.Fatal(err)
	}

	cs, err = service.Query(namespace, QueryOptions{
		Enabled: &con.Enabled,
		FromIDs: []uint64{con.FromID},
		Types:   []Type{con.Type},
		ToIDs:   []uint64{con.ToID},
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(cs), 1; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	if have, want := cs[0], updated; !reflect.DeepEqual(have, want) {
		t.Errorf("have %v, want %v", have, want)
	}
}

func testServicePutInvalid(t *testing.T, p prepareFunc) {
	var (
		namespace = "service_put_invalid"
		service   = p(t, namespace)
	)

	// missing FromID
	_, err := service.Put(namespace, &Connection{})
	if !IsInvalidConnection(err) {
		t.Errorf("expected error: %s", ErrInvalidConnection)
	}

	// missing ToID
	_, err = service.Put(namespace, &Connection{
		FromID: uint64(rand.Int63()),
	})
	if !IsInvalidConnection(err) {
		t.Errorf("expected error: %s", ErrInvalidConnection)
	}

	// missing State
	_, err = service.Put(namespace, &Connection{
		FromID: uint64(rand.Int63()),
		ToID:   uint64(rand.Int63()),
	})
	if !IsInvalidConnection(err) {
		t.Errorf("expected error: %s", ErrInvalidConnection)
	}

	// missing Type
	_, err = service.Put(namespace, &Connection{
		FromID: uint64(rand.Int63()),
		ToID:   uint64(rand.Int63()),
		State:  StateConfirmed,
	})
	if !IsInvalidConnection(err) {
		t.Errorf("expected error: %s", ErrInvalidConnection)
	}
}

func testServiceQuery(t *testing.T, p prepareFunc) {
	var (
		namespace = "service_query"
		service   = p(t, namespace)
		from      = uint64(rand.Int63())
		to        = uint64(rand.Int63())
		disabled  = false
		start     = time.Now()
	)

	for _, c := range testList(from, to, time.Now()) {
		_, err := service.Put(namespace, c)
		if err != nil {
			t.Fatal(err)
		}
	}

	cases := map[*QueryOptions]int{
		&QueryOptions{}: 36,
		&QueryOptions{Before: start.Add(-(time.Hour + time.Minute))}: 10,
		&QueryOptions{Enabled: &disabled}:                            5,
		&QueryOptions{FromIDs: []uint64{from}}:                       12,
		&QueryOptions{Limit: 10}:                                     10,
		&QueryOptions{States: []State{StateConfirmed}}:               18,
		&QueryOptions{ToIDs: []uint64{to}}:                           13,
		&QueryOptions{Types: []Type{TypeFriend}}:                     29,
	}

	for opts, want := range cases {
		cs, err := service.Query(namespace, *opts)
		if err != nil {
			t.Fatal(err)
		}

		if have := len(cs); have != want {
			t.Errorf("have %v, want %v", have, want)
		}
	}
}
