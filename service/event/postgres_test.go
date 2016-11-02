// build +integration

package event

import (
	"flag"
	"fmt"
	"os/user"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var (
	day  = 24 * time.Hour
	week = 7 * day

	pgURL string
)

func TestPostgresCount(t *testing.T) {
	testServiceCount(func(ns string, t *testing.T) Service {
		s, _ := preparePostgres(ns, t)
		return s
	}, t)
}

func TestPostgresPut(t *testing.T) {
	testServicePut(func(ns string, t *testing.T) Service {
		s, _ := preparePostgres(ns, t)
		return s
	}, t)
}

func TestPostgresQuery(t *testing.T) {
	testServiceQuery(func(ns string, t *testing.T) Service {
		s, _ := preparePostgres(ns, t)
		return s
	}, t)
}

func preparePostgres(namespace string, t *testing.T) (Service, *sqlx.DB) {
	db, err := sqlx.Connect("postgres", pgURL)
	if err != nil {
		t.Fatal(err)
	}

	s := PostgresService(db)

	err = s.Teardown(namespace)
	if err != nil {
		t.Fatal(err)
	}

	return s, db
}

func init() {
	user, err := user.Current()
	if err != nil {
		panic(err)
	}

	d := fmt.Sprintf(
		"postgres://%s@127.0.0.1:5432/tapglue_test?sslmode=disable&connect_timeout=5",
		user.Username,
	)

	url := flag.String("postgres.url", d, "Postgres connection URL")
	flag.Parse()

	pgURL = *url
}
