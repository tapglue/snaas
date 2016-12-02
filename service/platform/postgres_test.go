// +build integration

package platform

import (
	"flag"
	"fmt"
	"os/user"

	"github.com/tapglue/snaas/platform/pg"

	"github.com/jmoiron/sqlx"

	"testing"
)

var pgTestURL string

func TestPostgresPut(t *testing.T) {
	testServicePut(t, preparePostgres)
}

func TestPostgresQuery(t *testing.T) {
	testServiceQuery(t, preparePostgres)
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
