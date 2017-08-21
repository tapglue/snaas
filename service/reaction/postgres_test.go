// +build integration

package reaction

import (
	"flag"
	"fmt"
	"os/user"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var pgURL string

func TestPostgresCount(t *testing.T) {
	testServiceCount(preparePostgres, t)
}

func TestPostgresCountMulti(t *testing.T) {
	testServiceCountMulti(preparePostgres, t)
}

func TestPostgresPut(t *testing.T) {
	testServicePut(preparePostgres, t)
}

func TestPostgresQuery(t *testing.T) {
	testServiceQuery(preparePostgres, t)
}

func preparePostgres(t *testing.T, namespace string) Service {
	db, err := sqlx.Connect("postgres", pgURL)
	if err != nil {
		t.Fatal(err)
	}

	s := PostgresService(db)

	err = s.Teardown(namespace)
	if err != nil {
		t.Fatal(err)
	}

	return s
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
