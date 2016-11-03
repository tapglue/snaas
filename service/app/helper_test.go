package app

import (
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/tapglue/snaas/platform/generate"
)

type prepareFunc func(t *testing.T, namespace string) Service

func testApp() *App {
	return &App{
		BackendToken: generate.RandomString(12),
		Description:  generate.RandomString(36),
		InProduction: false,
		Enabled:      true,
		Name:         generate.RandomString(8),
		OrgID:        1,
		PublicID:     generate.RandomString(16),
		PublicOrgID:  generate.RandomString(16),
		Token:        generate.RandomString(8),
	}
}

func testList() List {
	as := List{}

	for i := 0; i < 9; i++ {
		as = append(as, testApp())
	}

	for i := 0; i < 5; i++ {
		a := testApp()
		a.OrgID = 2

		as = append(as, a)
	}

	return as
}

func testServicePut(t *testing.T, p prepareFunc) {
	var (
		app       = testApp()
		enabled   = true
		namespace = "service_put"
		service   = p(t, namespace)
	)

	created, err := service.Put(namespace, app)
	if err != nil {
		t.Fatal(err)
	}

	list, err := service.Query(namespace, QueryOptions{
		Enabled: &enabled,
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

	created.Enabled = false

	updated, err := service.Put(namespace, created)
	if err != nil {
		t.Fatal(err)
	}

	list, err = service.Query(namespace, QueryOptions{
		Enabled: &created.Enabled,
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
	if have, want := list[0], updated; !reflect.DeepEqual(have, want) {
		t.Errorf("have %v, want %v", have, want)
	}
}

func testServiceQuery(t *testing.T, p prepareFunc) {
	var (
		namespace = "service_query"
		service   = p(t, namespace)
	)

	list, err := service.Query(namespace, QueryOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(list), 0; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	app := testApp()

	created, err := service.Put(namespace, app)
	if err != nil {
		t.Fatal(err)
	}

	for _, a := range testList() {
		_, err := service.Put(namespace, a)
		if err != nil {
			t.Fatal(err)
		}
	}

	cases := map[*QueryOptions]int{
		&QueryOptions{}:                                              15,
		&QueryOptions{OrgIDs: []uint64{2}}:                           5,
		&QueryOptions{BackendTokens: []string{created.BackendToken}}: 1,
		&QueryOptions{IDs: []uint64{created.ID}}:                     1,
		&QueryOptions{PublicIDs: []string{created.PublicID}}:         1,
		&QueryOptions{Tokens: []string{created.Token}}:               1,
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

func init() {
	rand.Seed(time.Now().UnixNano())
}
