package platform

import (
	"math/rand"
	"reflect"
	"testing"

	"github.com/tapglue/snaas/platform/generate"
	"github.com/tapglue/snaas/platform/sns"
)

type prepareFunc func(t *testing.T, namespace string) Service

func testServicePut(t *testing.T, p prepareFunc) {
	var (
		platform  = testPlatform()
		namespace = "service_put"
		service   = p(t, namespace)
	)

	created, err := service.Put(namespace, platform)
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

	overwrite := testPlatform()
	overwrite.CreatedAt = list[0].CreatedAt
	overwrite.ID = list[0].ID

	updated, err := service.Put(namespace, overwrite)
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
		activated = true
		deleted   = true
		input     = testPlatform()
		namespace = "service_query"
		service   = p(t, namespace)
	)

	ps, err := service.Query(namespace, QueryOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(ps), 0; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	input.Active = true

	created, err := service.Put(namespace, input)
	if err != nil {
		t.Fatal(err)
	}

	for _, p := range testList() {
		_, err := service.Put(namespace, p)
		if err != nil {
			t.Fatal(err)
		}
	}

	cases := map[*QueryOptions]int{
		&QueryOptions{}:                                24,
		&QueryOptions{Active: &activated}:              1,
		&QueryOptions{ARNs: []string{created.ARN}}:     1,
		&QueryOptions{AppIDs: []uint64{created.AppID}}: 1,
		&QueryOptions{Deleted: &deleted}:               5,
		&QueryOptions{Ecosystems: []sns.Platform{IOS}}: 11,
		&QueryOptions{IDs: []uint64{created.ID}}:       1,
	}
	for opts, want := range cases {
		list, err := service.Query(namespace, *opts)
		if err != nil {
			t.Fatal(err)
		}

		if have := len(list); have != want {
			t.Errorf("have %v, want %v", have, want)
		}
	}
}

func testList() List {
	ps := List{}

	for i := 0; i < 7; i++ {
		ps = append(ps, testPlatform())
	}

	for i := 0; i < 5; i++ {
		p := testPlatform()
		p.Deleted = true

		ps = append(ps, p)
	}

	for i := 0; i < 11; i++ {
		p := testPlatform()
		p.Ecosystem = IOS

		ps = append(ps, p)
	}

	return ps
}

func testPlatform() *Platform {
	return &Platform{
		Active:    false,
		AppID:     uint64(rand.Int63()),
		ARN:       generate.RandomStringSafe(24),
		Ecosystem: Android,
		Name:      generate.RandomStringSafe(8),
		Scheme:    generate.RandomStringSafe(4),
	}
}
