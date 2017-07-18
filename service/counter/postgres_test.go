// +build integration

package counter

import (
	"flag"
	"fmt"
	"math/rand"
	"os/user"
	"testing"

	"github.com/jmoiron/sqlx"

	"github.com/tapglue/snaas/platform/pg"
)

var pgTestURL string

func TestPostgresCountAll(t *testing.T) {
	var (
		namespace = "service_count_all"
		service   = preparePostgres(t, namespace)
		name      = "beers"

		want uint64
	)

	for i := 0; i < rand.Intn(64); i++ {
		userID, value := testUser()

		err := service.Set(namespace, name, userID, value)
		if err != nil {
			t.Fatal(err)
		}

		want += value
	}

	have, err := service.CountAll(namespace, name)
	if err != nil {
		t.Fatal(err)
	}

	if have != want {
		t.Errorf("have %v, want %v", have, want)
	}
}

func TestPostgresSet(t *testing.T) {
	var (
		namespace     = "service_set"
		service       = preparePostgres(t, namespace)
		name          = "setter"
		userID, value = testUser()
	)

	err := service.Set(namespace, name, userID, value)
	if err != nil {
		t.Fatal(err)
	}

	c, err := service.Count(namespace, name, userID)
	if err != nil {
		t.Fatal(err)
	}

	if have, want := c, value; err != nil {
		t.Errorf("have %v, want %v", have, want)
	}
}

func preparePostgres(t *testing.T, namespace string) Service {
	db, err := sqlx.Connect("postgres", pgTestURL)
	if err != nil {
		t.Fatal(err)
	}

	s := PostgresService(db)

	if err := s.Teardown(namespace); err != nil {
		t.Fatal(err)
	}

	return s
}

func testUser() (uint64, uint64) {
	return uint64(rand.Int63()), uint64(rand.Int31())
}

func init() {
	u, err := user.Current()
	if err != nil {
		panic(err)
	}

	d := fmt.Sprintf(pg.URLTest, u.Username)

	url := flag.String("postgres.url", d, "Postgres test connection URL")
	flag.Parse()

	pgTestURL = *url
}
