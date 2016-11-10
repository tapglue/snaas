package core

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/tapglue/snaas/platform/generate"
	"github.com/tapglue/snaas/service/app"
	"github.com/tapglue/snaas/service/connection"
	"github.com/tapglue/snaas/service/session"
	"github.com/tapglue/snaas/service/user"
)

func TestUserCreateConstrainPrivate(t *testing.T) {
	var (
		app      = testSetupUser()
		origin   = Origin{Integration: IntegrationApplication}
		sessions = session.MemService()
		users    = user.MemService()
		fn       = UserCreate(sessions, users)
	)

	u := testUser()
	u.Private = &user.Private{
		Verified: true,
	}

	_, err := fn(app, origin, u)

	if have, want := err, ErrUnauthorized; !IsUnauthorized(have) {
		t.Errorf("have %v, want %v", have, want)
	}
}

func TestUserUpdateConstrainPrivate(t *testing.T) {
	var (
		app         = testSetupUser()
		connections = connection.MemService()
		sessions    = session.MemService()
		u           = testUser()
		users       = user.MemService()
		fn          = UserUpdate(connections, sessions, users)
	)

	created, err := users.Put(app.Namespace(), u)
	if err != nil {
		t.Fatal(err)
	}

	created.Private = &user.Private{
		Type:     "brand",
		Verified: true,
	}

	_, err = fn(
		app,
		Origin{
			Integration: IntegrationApplication,
			UserID:      created.ID,
		},
		u,
		created,
	)

	if have, want := err, ErrUnauthorized; !IsUnauthorized(have) {
		t.Errorf("have %v, want %v", have, want)
	}
}

func TestPassword(t *testing.T) {
	password := "foobar"

	epw, err := passwordSecure(password)
	if err != nil {
		t.Fatal(err)
	}

	valid, err := passwordCompare(password, epw)
	if err != nil {
		t.Fatal(err)
	}

	if have, want := valid, true; have != want {
		t.Errorf("have %v, want %v", have, want)
	}
}

func testSetupUser() *app.App {
	return &app.App{
		ID: uint64(rand.Int63()),
	}
}

func testUser() *user.User {
	return &user.User{
		Email: fmt.Sprintf(
			"user%d@tapglue.test", rand.Int63(),
		),
		Enabled:  true,
		Password: generate.RandomString(8),
	}
}
