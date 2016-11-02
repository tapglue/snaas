package core

import (
	"math/rand"
	"reflect"
	"testing"

	"github.com/tapglue/api/service/connection"

	"github.com/tapglue/api/service/event"

	"github.com/tapglue/api/service/app"
	"github.com/tapglue/api/service/object"
	"github.com/tapglue/api/service/user"
)

func TestPostCreate(t *testing.T) {
	var (
		app, owner = testSetupPost()
		objects    = object.MemService()
		post       = &Post{
			Object: &object.Object{
				Attachments: []object.Attachment{
					object.TextAttachment("body", object.Contents{
						"en": "Test body.",
					}),
				},
				Tags: []string{
					"review",
				},
				Visibility: object.VisibilityPublic,
			},
		}
		fn = PostCreate(objects)
	)

	created, err := fn(
		app,
		Origin{
			Integration: IntegrationApplication,
			UserID:      owner.ID,
		},
		post,
	)
	if err != nil {
		t.Fatal(err)
	}

	rs, err := objects.Query(app.Namespace(), object.QueryOptions{
		ID:    &created.ID,
		Owned: &defaultOwned,
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(rs), 1; have != want {
		t.Fatalf("have %v, want %v", have, want)
	}

	if have, want := rs[0], created.Object; !reflect.DeepEqual(have, want) {
		t.Errorf("have %v, want %v", have, want)
	}
}

func TestPostCreateConstrainVisibility(t *testing.T) {
	var (
		app, owner = testSetupPost()
		objects    = object.MemService()
		post       = &Post{
			Object: &object.Object{
				Visibility: object.VisibilityGlobal,
			},
		}
		fn = PostCreate(objects)
	)

	_, err := fn(
		app,
		Origin{
			Integration: IntegrationApplication,
			UserID:      owner.ID,
		},
		post,
	)

	if have, want := err, ErrUnauthorized; !IsUnauthorized(have) {
		t.Errorf("have %v, want %v", have, want)
	}
}

func TestPostDelete(t *testing.T) {
	var (
		app, owner = testSetupPost()
		objects    = object.MemService()
		post       = testPost(owner.ID)
		fn         = PostDelete(objects)
	)

	created, err := objects.Put(app.Namespace(), post.Object)
	if err != nil {
		t.Fatal(err)
	}

	err = fn(app, owner.ID+1, created.ID)
	if have, want := err, ErrUnauthorized; !IsUnauthorized(err) {
		t.Errorf("have %v, want %v", have, want)
	}

	err = fn(app, owner.ID, created.ID)
	if err != nil {
		t.Fatal(err)
	}

	os, err := objects.Query(app.Namespace(), object.QueryOptions{
		Deleted: true,
		ID:      &created.ID,
		Owned:   &defaultOwned,
		Types: []string{
			TypePost,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(os), 1; have != want {
		t.Fatalf("have %v, want %v", have, want)
	}

	err = fn(app, owner.ID, created.ID)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPostListAll(t *testing.T) {
	var (
		app, owner = testSetupPost()
		events     = event.MemService()
		objects    = object.MemService()
		users      = user.MemService()
		fn         = PostListAll(events, objects, users)
	)

	feed, err := fn(app, owner.ID, object.QueryOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(feed.Posts), 0; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	for _, post := range testPostSet(owner.ID) {
		_, err = objects.Put(app.Namespace(), post)
		if err != nil {
			t.Fatal(err)
		}
	}

	feed, err = fn(app, owner.ID, object.QueryOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(feed.Posts), 3; have != want {
		t.Errorf("have %v, want %v", have, want)
	}
}

func TestPostListUser(t *testing.T) {
	var (
		app, owner  = testSetupPost()
		connections = connection.MemService()
		events      = event.MemService()
		objects     = object.MemService()
		users       = user.MemService()
		fn          = PostListUser(connections, events, objects, users)
	)

	feed, err := fn(app, owner.ID, owner.ID, object.QueryOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(feed.Posts), 0; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	for _, post := range testPostSet(owner.ID) {
		_, err = objects.Put(app.Namespace(), post)
		if err != nil {
			t.Fatal(err)
		}
	}

	feed, err = fn(app, owner.ID, owner.ID, object.QueryOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(feed.Posts), 3; have != want {
		t.Errorf("have %v, want %v", have, want)
	}
}

func TestPostRetrieve(t *testing.T) {
	var (
		app, owner  = testSetupPost()
		connections = connection.MemService()
		events      = event.MemService()
		objects     = object.MemService()
		post        = testPost(owner.ID)
		fn          = PostRetrieve(connections, events, objects)
	)

	created, err := objects.Put(app.Namespace(), post.Object)
	if err != nil {
		t.Fatal(err)
	}

	r, err := fn(app, owner.ID, created.ID)
	if err != nil {
		t.Fatal(err)
	}

	if have, want := r.Object, created; !reflect.DeepEqual(have, want) {
		t.Fatalf("have %v, want %v", have, want)
	}

	_, err = fn(app, owner.ID, created.ID-1)
	if have, want := err, ErrNotFound; have != want {
		t.Errorf("have %v, want %v", have, want)
	}
}

func TestPostUpdate(t *testing.T) {
	var (
		app, owner = testSetupPost()
		objects    = object.MemService()
		post       = testPost(owner.ID)
		fn         = PostUpdate(objects)
	)

	created, err := objects.Put(app.Namespace(), post.Object)
	if err != nil {
		t.Fatal(err)
	}

	created.OwnerID = 0

	_, err = fn(
		app,
		Origin{
			Integration: IntegrationApplication,
			UserID:      owner.ID,
		},
		created.ID,
		&Post{Object: created},
	)
	if err != nil {
		t.Fatal(err)
	}

	ps, err := objects.Query(app.Namespace(), object.QueryOptions{
		ID:    &created.ID,
		Owned: &defaultOwned,
		Types: []string{
			TypePost,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(ps), 1; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	updated := ps[0]

	if have, want := updated.OwnerID, post.OwnerID; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	if have, want := updated.Visibility, post.Visibility; have != want {
		t.Errorf("have %v, want %v", have, want)
	}
}

func TestPostUpdateConstrainVisibility(t *testing.T) {
	var (
		app, owner = testSetupPost()
		origin     = Origin{
			Integration: IntegrationApplication,
			UserID:      owner.ID,
		}
		objects = object.MemService()
		post    = testPost(owner.ID)
		fn      = PostUpdate(objects)
	)

	created, err := objects.Put(app.Namespace(), post.Object)
	if err != nil {
		t.Fatal(err)
	}

	created.Visibility = object.VisibilityGlobal

	post = &Post{Object: created}

	_, err = fn(app, origin, created.ID, post)

	if have, want := err, ErrUnauthorized; !IsUnauthorized(have) {
		t.Errorf("have %v, want %v", have, want)
	}
}

func TestPostUpdateMissing(t *testing.T) {
	var (
		app, owner = testSetupPost()
		objects    = object.MemService()
		post       = testPost(owner.ID)
		fn         = PostUpdate(objects)
	)

	_, err := fn(
		app,
		Origin{
			Integration: IntegrationApplication,
			UserID:      owner.ID,
		},
		post.ID,
		post,
	)
	if have, want := err, ErrNotFound; have != want {
		t.Errorf("have %v, want %v", have, want)
	}
}

func testPost(ownerID uint64) *Post {
	return &Post{
		Object: &object.Object{
			Attachments: []object.Attachment{
				object.TextAttachment("body", object.Contents{
					"en": "Test body.",
				}),
			},
			OwnerID: ownerID,
			Owned:   true,
			Tags: []string{
				"review",
			},
			Type:       TypePost,
			Visibility: object.VisibilityPublic,
		},
	}
}

func testPostSet(ownerID uint64) []*object.Object {
	return []*object.Object{
		{
			OwnerID:    ownerID,
			Owned:      true,
			Type:       TypePost,
			Visibility: object.VisibilityConnection,
		},
		{
			OwnerID:    ownerID + 1,
			Owned:      true,
			Type:       TypePost,
			Visibility: object.VisibilityPublic,
		},
		{
			OwnerID:    ownerID - 1,
			Owned:      true,
			Type:       TypePost,
			Visibility: object.VisibilityPublic,
		},
		{
			OwnerID:    ownerID,
			Owned:      true,
			Type:       TypePost,
			Visibility: object.VisibilityPublic,
		},
		{
			OwnerID:    ownerID,
			Owned:      true,
			Type:       TypePost,
			Visibility: object.VisibilityPrivate,
		},
	}
}

func testSetupPost() (*app.App, *user.User) {
	return &app.App{
			ID:    uint64(rand.Int63()),
			OrgID: uint64(rand.Int63()),
		}, &user.User{
			ID: uint64(rand.Int63()),
		}

}
