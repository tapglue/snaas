package reaction

import (
	"math/rand"
	"reflect"
	"testing"
	"time"
)

type prepareFunc func(t *testing.T, namespace string) Service

func testServiceCount(p prepareFunc, t *testing.T) {
	var (
		deleted   = true
		objectID  = uint64(rand.Int63())
		ownerID   = uint64(rand.Int63())
		namespace = "service_count"
		service   = p(t, namespace)
	)

	for _, r := range testList(objectID, ownerID) {
		_, err := service.Put(namespace, r)
		if err != nil {
			t.Fatal(err)
		}
	}

	cases := map[*QueryOptions]uint{
		&QueryOptions{}:                                  51, // All
		&QueryOptions{Deleted: &deleted}:                 5,  // Deleted
		&QueryOptions{ObjectIDs: []uint64{objectID}}:     3,  // By Object
		&QueryOptions{OwnerIDs: []uint64{ownerID}}:       11, // By Owner
		&QueryOptions{Types: []Type{TypeLike}}:           26, // Likes
		&QueryOptions{Types: []Type{TypeLove}}:           9,  // Loves
		&QueryOptions{Types: []Type{TypeHaha}}:           3,  // Hahas
		&QueryOptions{Types: []Type{TypeWow}}:            7,  // Wows
		&QueryOptions{Types: []Type{TypeSad}}:            1,  // Sads
		&QueryOptions{Types: []Type{TypeAngry}}:          5,  // Angries
		&QueryOptions{Types: []Type{TypeLike, TypeHaha}}: 29, // Combined
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

func testServiceCountMulti(p prepareFunc, t *testing.T) {
	var (
		objectIDs = []uint64{
			uint64(rand.Int63()),
			uint64(rand.Int63()),
			uint64(rand.Int63()),
		}
		ownerID   = uint64(rand.Int63())
		namespace = "service_count_multi"
		service   = p(t, namespace)
	)

	for _, oid := range objectIDs {
		for _, r := range testList(oid, ownerID) {
			r.ObjectID = oid

			_, err := service.Put(namespace, r)
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	want := CountsMap{}

	for _, oid := range objectIDs {
		want[oid] = Counts{
			Angry: 5,
			Haha:  3,
			Like:  21,
			Love:  9,
			Sad:   1,
			Wow:   7,
		}
	}

	have, err := service.CountMulti(namespace, QueryOptions{
		ObjectIDs: objectIDs,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(have, want) {
		t.Errorf("\nhave %v\nwant %v", have, want)
	}
}

func testServicePut(p prepareFunc, t *testing.T) {
	var (
		deleted   = true
		enabled   = false
		namespace = "service_put"
		service   = p(t, namespace)
	)

	created, err := service.Put(namespace, testReactionLike())
	if err != nil {
		t.Fatal(err)
	}

	list, err := service.Query(namespace, QueryOptions{
		Deleted: &enabled,
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(list), 1; have != want {
		t.Fatalf("have %v, want %v", have, want)
	}

	if have, want := list[0], created; !reflect.DeepEqual(have, want) {
		t.Fatalf("\nhave %v\nwant %v", have, want)
	}

	created.Deleted = true

	updated, err := service.Put(namespace, created)
	if err != nil {
		t.Fatal(err)
	}

	list, err = service.Query(namespace, QueryOptions{
		Deleted: &deleted,
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(list), 1; have != want {
		t.Fatalf("have %v, want %v", have, want)
	}

	if have, want := list[0], updated; !reflect.DeepEqual(have, want) {
		t.Fatalf("\nhave %v\nwant %v", have, want)
	}
}

func testServiceQuery(p prepareFunc, t *testing.T) {
	var (
		deleted   = true
		objectID  = uint64(rand.Int63())
		ownerID   = uint64(rand.Int63())
		namespace = "service_query"
		service   = p(t, namespace)
	)

	for _, r := range testList(objectID, ownerID) {
		_, err := service.Put(namespace, r)
		if err != nil {
			t.Fatal(err)
		}
	}

	created, err := service.Put(namespace, testReactionLike())
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Millisecond)

	cases := map[*QueryOptions]uint{
		&QueryOptions{}:                                  52, // All
		&QueryOptions{Before: created.UpdatedAt}:         51, // Deleted
		&QueryOptions{Deleted: &deleted}:                 5,  // Deleted
		&QueryOptions{IDs: []uint64{created.ID}}:         1,
		&QueryOptions{Limit: 11}:                         11, // Deleted
		&QueryOptions{ObjectIDs: []uint64{objectID}}:     3,  // By Object
		&QueryOptions{OwnerIDs: []uint64{ownerID}}:       11, // By Owner
		&QueryOptions{Types: []Type{TypeLike}}:           27, // Likes
		&QueryOptions{Types: []Type{TypeLove}}:           9,  // Loves
		&QueryOptions{Types: []Type{TypeHaha}}:           3,  // Hahas
		&QueryOptions{Types: []Type{TypeWow}}:            7,  // Wows
		&QueryOptions{Types: []Type{TypeSad}}:            1,  // Sads
		&QueryOptions{Types: []Type{TypeAngry}}:          5,  // Angries
		&QueryOptions{Types: []Type{TypeLike, TypeHaha}}: 30, // Combined
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

func testList(objectID, ownerID uint64) List {
	rs := List{}

	for i := 0; i < 7; i++ {
		rs = append(rs, testReactionLike())
	}

	for i := 0; i < 5; i++ {
		r := testReactionLike()

		r.Deleted = true

		rs = append(rs, r)
	}

	for i := 0; i < 3; i++ {
		r := testReactionLike()

		r.ObjectID = objectID

		rs = append(rs, r)
	}

	for i := 0; i < 11; i++ {
		r := testReactionLike()

		r.OwnerID = ownerID

		rs = append(rs, r)
	}

	for i := 0; i < 9; i++ {
		r := testReactionLike()

		r.Type = TypeLove

		rs = append(rs, r)
	}

	for i := 0; i < 3; i++ {
		r := testReactionLike()

		r.Type = TypeHaha

		rs = append(rs, r)
	}

	for i := 0; i < 7; i++ {
		r := testReactionLike()

		r.Type = TypeWow

		rs = append(rs, r)
	}

	for i := 0; i < 1; i++ {
		r := testReactionLike()

		r.Type = TypeSad

		rs = append(rs, r)
	}

	for i := 0; i < 5; i++ {
		r := testReactionLike()

		r.Type = TypeAngry

		rs = append(rs, r)
	}

	return rs
}

func testReactionLike() *Reaction {
	return &Reaction{
		ObjectID: uint64(rand.Int63()),
		OwnerID:  uint64(rand.Int63()),
		Type:     TypeLike,
	}
}
