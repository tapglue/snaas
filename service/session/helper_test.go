package session

import (
	"math/rand"
	"reflect"
	"testing"

	"github.com/tapglue/snaas/platform/generate"
)

type prepareFunc func(t *testing.T, namespace string) Service

func testList() List {
	ss := List{}

	for i := 0; i < 6; i++ {
		ss = append(ss, testSession())
	}

	return ss
}

func testServicePut(t *testing.T, p prepareFunc) {
	var (
		enabled   = true
		namespace = "service_put"
		service   = p(t, namespace)
		session   = testSession()
	)

	created, err := service.Put(namespace, session)
	if err != nil {
		t.Fatal(err)
	}

	list, err := service.Query(namespace, QueryOptions{
		Enabled: &enabled,
		IDs: []string{
			created.ID,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(list), 1; have != want {
		t.Fatalf("have %v, want %v", have, want)
	}
	if have, want := list[0], created; !reflect.DeepEqual(have, want) {
		t.Errorf("have %v, want %v", have, want)
	}
}

func testServiecQuery(t *testing.T, p prepareFunc) {
	var (
		enabled   = true
		namespace = "service_query"
		service   = p(t, namespace)
	)

	ss, err := service.Query(namespace, QueryOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(ss), 0; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	created, err := service.Put(namespace, testSession())
	if err != nil {
		t.Fatal(err)
	}

	for _, s := range testList() {
		_, err := service.Put(namespace, s)
		if err != nil {
			t.Fatal(err)
		}
	}

	ss, err = service.Query(namespace, QueryOptions{
		DeviceIDs: []string{
			created.DeviceID,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(ss), 1; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	ss, err = service.Query(namespace, QueryOptions{
		Enabled: &enabled,
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(ss), 7; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	ss, err = service.Query(namespace, QueryOptions{
		IDs: []string{
			created.ID,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(ss), 1; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	ss, err = service.Query(namespace, QueryOptions{
		UserIDs: []uint64{
			created.UserID,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(ss), 1; have != want {
		t.Errorf("have %v, want %v", have, want)
	}
}

func testSession() *Session {
	return &Session{
		Enabled:  true,
		DeviceID: generate.RandomString(24),
		UserID:   uint64(rand.Int63()),
	}
}
