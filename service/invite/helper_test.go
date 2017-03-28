package invite

import (
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/tapglue/snaas/platform/generate"
)

type prepareFunc func(t *testing.T, namespace string) Service

func testServicePut(t *testing.T, p prepareFunc) {
	var (
		invite    = testInvite()
		namespace = "service_put"
		service   = p(t, namespace)
	)

	created, err := service.Put(namespace, invite)
	if err != nil {
		t.Fatal(err)
	}

	list, err := service.Query(namespace, QueryOptions{
		IDs: []uint64{
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

	created.Deleted = true

	updated, err := service.Put(namespace, created)
	if err != nil {
		t.Fatal(err)
	}

	list, err = service.Query(namespace, QueryOptions{
		IDs: []uint64{
			updated.ID,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := list[0], updated; !reflect.DeepEqual(have, want) {
		t.Errorf("have %v, want %v", have, want)
	}
}

func testServiceQuery(t *testing.T, p prepareFunc) {
	var (
		deleted   = true
		namespace = "service_query"
		service   = p(t, namespace)
		userID    = uint64(rand.Int63())
	)

	for _, i := range testList(userID) {
		_, err := service.Put(namespace, i)
		if err != nil {
			t.Fatal(err)
		}
	}

	created, err := service.Put(namespace, testInvite())
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Millisecond)

	cases := map[*QueryOptions]uint{
		&QueryOptions{}:                                9,
		&QueryOptions{Before: created.UpdatedAt}:       8,
		&QueryOptions{Deleted: &deleted}:               5,
		&QueryOptions{Keys: []string{created.Key}}:     1,
		&QueryOptions{Limit: 6}:                        6,
		&QueryOptions{IDs: []uint64{created.ID}}:       1,
		&QueryOptions{UserIDs: []uint64{userID}}:       3,
		&QueryOptions{Values: []string{created.Value}}: 1,
	}

	for opts, want := range cases {
		list, err := service.Query(namespace, *opts)
		if err != nil {
			t.Fatal(err)
		}

		if have := uint(len(list)); have != want {
			t.Errorf("have %v, want %v", have, want)
		}
	}
}

func testInvite() *Invite {
	return &Invite{
		Deleted: false,
		Key:     generate.RandomStringSafe(24),
		UserID:  uint64(rand.Int63()),
		Value:   generate.RandomStringSafe(24),
	}
}

func testList(userID uint64) List {
	is := List{}

	for i := 0; i < 5; i++ {
		i := testInvite()

		i.Deleted = true

		is = append(is, i)
	}

	for i := 0; i < 3; i++ {
		i := testInvite()

		i.UserID = userID

		is = append(is, i)
	}

	return is
}
