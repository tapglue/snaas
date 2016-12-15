package user

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"time"

	serr "github.com/tapglue/snaas/error"
	"github.com/tapglue/snaas/platform/generate"
)

type prepareFunc func(t *testing.T, namespace string) Service

func testList(platform string, socialIDs ...string) List {
	us := List{}

	for i := 0; i < 9; i++ {
		u := testUser()

		u.Deleted = true
		u.Enabled = false

		us = append(us, u)
	}

	for i := 0; i < 7; i++ {
		us = append(us, testUser())
	}

	for _, id := range socialIDs {
		u := testUser()

		u.SocialIDs = map[string]string{
			platform: id,
		}

		us = append(us, u)
	}

	return us
}

func testServiceCount(t *testing.T, p prepareFunc) {
	var (
		customID  = generate.RandomString(12)
		deleted   = true
		enabled   = true
		namespace = "service_count"
		platform  = "facebook"
		service   = p(t, namespace)
		socialIDs = []string{
			generate.RandomString(7),
			generate.RandomString(7),
			generate.RandomString(7),
			generate.RandomString(7),
			generate.RandomString(7),
		}
	)

	count, err := service.Count(namespace, QueryOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := count, 0; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	u := testUser()
	u.CustomID = customID
	u.Username = generate.RandomString(8)

	created, err := service.Put(namespace, u)
	if err != nil {
		t.Fatal(err)
	}

	for _, u := range testList(platform, socialIDs...) {
		_, err := service.Put(namespace, u)
		if err != nil {
			t.Fatal(err)
		}
	}

	count, err = service.Count(namespace, QueryOptions{
		CustomIDs: []string{
			customID,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := count, 1; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	count, err = service.Count(namespace, QueryOptions{
		Deleted: &deleted,
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := count, 9; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	count, err = service.Count(namespace, QueryOptions{
		Emails: []string{
			created.Email,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := count, 1; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	count, err = service.Count(namespace, QueryOptions{
		Enabled: &enabled,
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := count, 13; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	count, err = service.Count(namespace, QueryOptions{
		IDs: []uint64{
			created.ID,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := count, 1; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	count, err = service.Count(namespace, QueryOptions{
		SocialIDs: map[string][]string{
			platform: socialIDs,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := count, len(socialIDs); have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	count, err = service.Count(namespace, QueryOptions{
		Usernames: []string{
			created.Username,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := count, 1; have != want {
		t.Errorf("have %v, want %v", have, want)
	}
}

func testServicePut(t *testing.T, p prepareFunc) {
	var (
		enabled   = true
		namespace = "service_put"
		service   = p(t, namespace)
		user      = testUser()
	)

	created, err := service.Put(namespace, user)
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

	_, err = service.Put(namespace, &User{})
	if have, want := err, ErrInvalidUser; !IsInvalidUser(err) {
		t.Errorf("have %v, want %v", have, want)
	}

	invalidID := testUser()
	invalidID.ID = 1

	_, err = service.Put(namespace, invalidID)
	if have, want := err, ErrNotFound; !IsNotFound(err) {
		t.Errorf("have %v, want %v", have, want)
	}
}

func testServicePutEmailUnique(t *testing.T, p prepareFunc) {
	var (
		namespace = "service_put_email"
		service   = p(t, namespace)
		user      = testUser()
		lowerCase = "xla@tgpl.dev"
		mixedCase = "xlA@tgpl.dev"
	)

	user.Email = lowerCase

	_, err := service.Put(namespace, user)
	if err != nil {
		t.Fatal(err)
	}

	us, err := service.Query(namespace, QueryOptions{
		Emails: []string{
			mixedCase,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(us), 1; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	second := testUser()
	second.Email = mixedCase

	_, err = service.Put(namespace, second)
	if have, want := err, serr.ErrUserExists; !serr.IsUserExists(err) {
		t.Errorf("have %v, want %v\n%#v", have, want, err)
	}
}

func testServicePutLastRead(t *testing.T, p prepareFunc) {
	var (
		namespace = "service_put_last_read"
		service   = p(t, namespace)
		user      = testUser()
	)

	created, err := service.Put(namespace, user)
	if err != nil {
		t.Fatal(err)
	}

	format := "2006-01-02 15:04:05 UTC"

	ts, err := time.Parse(format, time.Now().Format(format))
	if err != nil {
		t.Fatal(err)
	}

	err = service.PutLastRead(namespace, created.ID, ts)
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

	created.LastRead = ts.UTC()

	if have, want := list[0], created; !reflect.DeepEqual(have, want) {
		t.Errorf("\nhave %v,\nwant %v", have, want)
	}
}

func testServicePutUsernameUnique(t *testing.T, p prepareFunc) {
	var (
		namespace = "service_put_email"
		service   = p(t, namespace)
		user      = testUser()
		lowerCase = "xla1234"
		mixedCase = "XlA1234"
	)

	user.Username = lowerCase

	_, err := service.Put(namespace, user)
	if err != nil {
		t.Fatal(err)
	}

	us, err := service.Query(namespace, QueryOptions{
		Usernames: []string{
			mixedCase,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(us), 1; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	second := testUser()
	second.Username = mixedCase

	_, err = service.Put(namespace, second)
	if have, want := err, serr.ErrUserExists; !serr.IsUserExists(err) {
		t.Errorf("have %v, want %v\n%#v", have, want, err)
	}
}

func testServiceQuery(t *testing.T, p prepareFunc) {
	var (
		customID  = generate.RandomString(12)
		deleted   = true
		enabled   = true
		namespace = "service_query"
		platform  = "twitter"
		service   = p(t, namespace)
		socialIDs = []string{
			generate.RandomString(5),
			generate.RandomString(5),
			generate.RandomString(5),
			generate.RandomString(5),
			generate.RandomString(5),
			generate.RandomString(5),
			generate.RandomString(5),
		}
		ts = testList(platform, socialIDs...)
	)

	list, err := service.Query(namespace, QueryOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(list), 0; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	u := testUser()
	u.CustomID = customID
	u.Username = generate.RandomString(8)

	created, err := service.Put(namespace, u)
	if err != nil {
		t.Fatal(err)
	}

	for _, u := range ts {
		_, err := service.Put(namespace, u)
		if err != nil {
			t.Fatal(err)
		}
	}

	cases := map[*QueryOptions]int{
		&QueryOptions{}:                                                    24,
		&QueryOptions{Before: ts[len(ts)-3].ID}:                            2,
		&QueryOptions{CustomIDs: []string{customID}}:                       1,
		&QueryOptions{Deleted: &deleted}:                                   9,
		&QueryOptions{Enabled: &enabled}:                                   15,
		&QueryOptions{IDs: []uint64{created.ID}}:                           1,
		&QueryOptions{Limit: 10}:                                           10,
		&QueryOptions{SocialIDs: map[string][]string{platform: socialIDs}}: len(socialIDs),
		&QueryOptions{Usernames: []string{created.Username}}:               1,
	}

	for opts, want := range cases {
		us, err := service.Query(namespace, *opts)
		if err != nil {
			t.Fatal(err)
		}

		if have := len(us); have != want {
			t.Errorf("have %v, want %v", have, want)
		}
	}
}

func testServiceSearch(t *testing.T, p prepareFunc) {
	var (
		namespace = "service_search"
		service   = p(t, namespace)
	)

	us, err := service.Search(namespace, QueryOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(us), 0; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	u := testUser()
	u.Firstname = generate.RandomString(12)
	u.Lastname = generate.RandomString(12)
	u.Username = generate.RandomString(8)

	created, err := service.Put(namespace, u)
	if err != nil {
		t.Fatal(err)
	}

	for _, u := range testList("") {
		_, err := service.Put(namespace, u)
		if err != nil {
			t.Fatal(err)
		}
	}

	us, err = service.Search(namespace, QueryOptions{
		Emails: []string{
			created.Email[0 : len(u.Email)-3],
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(us), 1; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	us, err = service.Search(namespace, QueryOptions{
		Firstnames: []string{
			created.Firstname[1:10],
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(us), 1; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	us, err = service.Search(namespace, QueryOptions{
		Lastnames: []string{
			created.Lastname[1:10],
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(us), 1; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	us, err = service.Search(namespace, QueryOptions{
		Enabled: &defaultEnabled,
		Usernames: []string{
			created.Username[3:7],
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(us), 1; have != want {
		t.Errorf("have %v, want %v", have, want)
	}
}

func testUser() *User {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	return &User{
		Email: fmt.Sprintf(
			"user%d@tapglue.test", r.Int63(),
		),
		Enabled:  true,
		Password: generate.RandomString(8),
		Username: generate.RandomString(8),
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
