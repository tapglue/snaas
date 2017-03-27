// +build integration

package rule

import (
	"flag"
	"fmt"
	"os/user"
	"reflect"
	"testing"

	"github.com/jmoiron/sqlx"

	"github.com/tapglue/snaas/platform/pg"
	"github.com/tapglue/snaas/platform/sns"
	"github.com/tapglue/snaas/service/connection"
	"github.com/tapglue/snaas/service/event"
	"github.com/tapglue/snaas/service/object"
	"github.com/tapglue/snaas/service/reaction"
)

var pgTestURL string

func TestPostgresPut(t *testing.T) {
	var (
		disabled  = false
		enabled   = true
		namespace = "service_put_connection"
		service   = preparePostgres(t, namespace)
	)

	rules := List{
		{
			Active: true,
			Criteria: &CriteriaConnection{
				New: &connection.QueryOptions{
					States: []connection.State{
						connection.StateConfirmed,
					},
					Types: []connection.Type{
						connection.TypeFriend,
					},
				},
				Old: &connection.QueryOptions{
					States: []connection.State{
						connection.StatePending,
					},
					Types: []connection.Type{
						connection.TypeFriend,
					},
				},
			},
			Deleted:   false,
			Ecosystem: sns.PlatformAPNS,
			Name:      "Friend confirm",
			Recipients: Recipients{
				{
					Query: map[string]string{
						"foo": "bar",
					},
					Templates: map[string]string{
						"en": "Where we mesage.",
					},
					URN: "",
				},
			},
			Type: TypeConnection,
		},
		{
			Active: true,
			Criteria: &CriteriaEvent{
				New: &event.QueryOptions{
					Enabled: &disabled,
					Types: []string{
						"signal",
					},
				},
				Old: &event.QueryOptions{
					Enabled: &enabled,
					Types: []string{
						"signal",
					},
				},
			},
			Deleted:   false,
			Ecosystem: sns.PlatformAPNS,
			Name:      "Friend confirm",
			Recipients: Recipients{
				{
					Query: map[string]string{
						"foo": "bar",
					},
					Templates: map[string]string{
						"en": "Where we mesage.",
					},
					URN: "",
				},
			},
			Type: TypeEvent,
		},
		{
			Active: true,
			Criteria: &CriteriaObject{
				New: &object.QueryOptions{
					Owned: &enabled,
					Types: []string{
						"review",
					},
					Tags: []string{
						"movie",
						"official",
					},
				},
				Old: &object.QueryOptions{
					Owned: &enabled,
					Types: []string{
						"review",
					},
					Tags: []string{
						"movie",
					},
				},
			},
			Deleted:   false,
			Ecosystem: sns.PlatformAPNS,
			Name:      "Friend confirm",
			Recipients: Recipients{
				{
					Query: map[string]string{
						"foo": "bar",
					},
					Templates: map[string]string{
						"en": "Where we mesage.",
					},
					URN: "",
				},
			},
			Type: TypeObject,
		},
		{
			Active: true,
			Criteria: &CriteriaReaction{
				New: &reaction.QueryOptions{
					Deleted: &enabled,
					Types: []reaction.Type{
						reaction.TypeLike,
					},
				},
				Old: &reaction.QueryOptions{
					Deleted: &disabled,
					Types: []reaction.Type{
						reaction.TypeLike,
					},
				},
			},
			Deleted:   false,
			Ecosystem: sns.PlatformAPNS,
			Name:      "Signal change",
			Recipients: Recipients{
				{
					Query: map[string]string{
						"foo": "bar",
					},
					Templates: map[string]string{
						"en": "Where we mesage.",
					},
					URN: "",
				},
			},
			Type: TypeReaction,
		},
	}

	for _, r := range rules {
		created, err := service.Put(namespace, r)
		if err != nil {
			t.Fatal(err)
		}

		deleted := false

		list, err := service.Query(namespace, QueryOptions{
			Deleted: &deleted,
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
			t.Errorf("\nhave %v\nwant %v", have, want)
		}

		created.Deleted = true

		_, err = service.Put(namespace, created)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestPostgresQuery(t *testing.T) {
	// var (
	// 	activate  = true
	// 	deleted   = true
	// 	namespace = "service_query"
	// 	service   = preparePostgres(t, namespace)
	// )
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
