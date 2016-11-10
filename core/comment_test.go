package core

import (
	"math/rand"
	"reflect"
	"testing"

	"github.com/tapglue/snaas/service/app"
	"github.com/tapglue/snaas/service/connection"
	"github.com/tapglue/snaas/service/object"
	"github.com/tapglue/snaas/service/user"
)

func TestCommentCreate(t *testing.T) {
	var (
		app, owner  = testSetupComment()
		connections = connection.MemService()
		origin      = Origin{
			Integration: IntegrationApplication,
			UserID:      owner.ID,
		}
		objects = object.MemService()
		fn      = CommentCreate(connections, objects)
	)

	post, err := objects.Put(app.Namespace(), testPost(owner.ID).Object)
	if err != nil {
		t.Fatal(err)
	}

	created, err := fn(app, origin, post.ID, testComment(owner.ID, post))
	if err != nil {
		t.Fatal(err)
	}

	cs, err := objects.Query(app.Namespace(), object.QueryOptions{
		ID:    &created.ID,
		Owned: &defaultOwned,
		Types: []string{
			TypeComment,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(cs), 1; have != want {
		t.Fatalf("have %v, want %v", have, want)
	}

	if have, want := cs[0], created; !reflect.DeepEqual(have, want) {
		t.Errorf("have %v, want %v", have, want)
	}

	created.Attachments[0] = object.Attachment{
		Contents: object.Contents{
			"en": "Do not like.",
		},
	}

	_, err = fn(app, origin, 0, created)
	if have, want := err, ErrNotFound; have != want {
		t.Fatalf("have %v, want %v", have, want)
	}
}

func TestCommentCreateConstrainPrivate(t *testing.T) {
	var (
		app, owner  = testSetupComment()
		connections = connection.MemService()
		origin      = Origin{
			Integration: IntegrationApplication,
			UserID:      owner.ID,
		}
		objects = object.MemService()
		fn      = CommentCreate(connections, objects)
	)

	post, err := objects.Put(app.Namespace(), testPost(owner.ID).Object)
	if err != nil {
		t.Fatal(err)
	}

	comment := testComment(owner.ID, post)
	comment.Private = &object.Private{
		Visible: true,
	}

	_, err = fn(app, origin, post.ID, comment)

	if have, want := err, ErrUnauthorized; !IsUnauthorized(have) {
		t.Errorf("have %v, want %v", have, want)
	}
}

func TestCommentDelete(t *testing.T) {
	var (
		app, owner  = testSetupComment()
		connections = connection.MemService()
		objects     = object.MemService()
		fn          = CommentDelete(connections, objects)
	)

	post, err := objects.Put(app.Namespace(), testPost(owner.ID).Object)
	if err != nil {
		t.Fatal(err)
	}

	created, err := objects.Put(app.Namespace(), testComment(owner.ID, post))
	if err != nil {
		t.Fatal(err)
	}

	err = fn(app, owner.ID, post.ID, created.ID)
	if err != nil {
		t.Fatal(err)
	}

	cs, err := objects.Query(app.Namespace(), object.QueryOptions{
		Deleted: true,
		ID:      &created.ID,
		Owned:   &defaultOwned,
		Types: []string{
			TypeComment,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(cs), 1; have != want {
		t.Fatalf("have %v, want %v", have, want)
	}

	err = fn(app, owner.ID, post.ID, created.ID)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCommentList(t *testing.T) {
	var (
		app, owner  = testSetupComment()
		connections = connection.MemService()
		objects     = object.MemService()
		users       = user.MemService()
		fn          = CommentList(connections, objects, users)
	)

	post, err := objects.Put(app.Namespace(), testPost(owner.ID).Object)
	if err != nil {
		t.Fatal(err)
	}

	list, err := fn(app, owner.ID, post.ID, object.QueryOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(list.Comments), 0; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	for _, comment := range testCommentSet(owner.ID, post) {
		_, err = objects.Put(app.Namespace(), comment)
		if err != nil {
			t.Fatal(err)
		}
	}

	list, err = fn(app, owner.ID, post.ID, object.QueryOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(list.Comments), 5; have != want {
		t.Errorf("have %v, want %v", have, want)
	}
}

func TestCommentRetrieve(t *testing.T) {
	var (
		app, owner = testSetupComment()
		objects    = object.MemService()
		fn         = CommentRetrieve(objects)
	)

	post, err := objects.Put(app.Namespace(), testPost(owner.ID).Object)
	if err != nil {
		t.Fatal(err)
	}

	created, err := objects.Put(app.Namespace(), testComment(owner.ID, post))
	if err != nil {
		t.Fatal(err)
	}

	r, err := fn(app, owner.ID, post.ID, created.ID)
	if err != nil {
		t.Fatal(err)
	}

	if have, want := r, created; !reflect.DeepEqual(have, want) {
		t.Fatalf("have %v, want %v", have, want)
	}

	_, err = fn(app, owner.ID, post.ID, created.ID-1)
	if have, want := err, ErrNotFound; have != want {
		t.Errorf("have %v, want %v", have, want)
	}
}

func TestCommentUpdate(t *testing.T) {
	var (
		app, owner = testSetupComment()
		origin     = Origin{
			Integration: IntegrationApplication,
			UserID:      owner.ID,
		}
		objects = object.MemService()
		fn      = CommentUpdate(objects)
	)

	post, err := objects.Put(app.Namespace(), testPost(owner.ID).Object)
	if err != nil {
		t.Fatal(err)
	}

	_, err = fn(app, origin, post.ID, 0, testComment(owner.ID, post))
	if have, want := err, ErrNotFound; have != want {
		t.Fatalf("have %v, want %v", have, want)
	}

	created, err := objects.Put(app.Namespace(), testComment(owner.ID, post))
	if err != nil {
		t.Fatal(err)
	}

	updated, err := fn(app, origin, post.ID, created.ID, testComment(owner.ID, post))
	if err != nil {
		t.Fatal(err)
	}

	cs, err := objects.Query(app.Namespace(), object.QueryOptions{
		ID:    &created.ID,
		Owned: &defaultOwned,
		Types: []string{
			TypeComment,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(cs), 1; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	if have, want := cs[0], updated; !reflect.DeepEqual(have, want) {
		t.Errorf("have %v, want %v", have, want)
	}
}

func TestCommentUpdateConstrainPrivate(t *testing.T) {
	var (
		app, owner = testSetupComment()
		origin     = Origin{
			Integration: IntegrationApplication,
			UserID:      owner.ID,
		}
		objects = object.MemService()
		fn      = CommentUpdate(objects)
	)

	post, err := objects.Put(app.Namespace(), testPost(owner.ID).Object)
	if err != nil {
		t.Fatal(err)
	}

	created, err := objects.Put(app.Namespace(), testComment(owner.ID, post))
	if err != nil {
		t.Fatal(err)
	}

	created.Private = &object.Private{
		Visible: true,
	}

	_, err = fn(app, origin, post.ID, created.ID, created)

	if have, want := err, ErrUnauthorized; !IsUnauthorized(have) {
		t.Errorf("have %v, want %v", have, want)
	}
}

func testComment(ownerID uint64, post *object.Object) *object.Object {
	return &object.Object{
		Attachments: []object.Attachment{
			object.TextAttachment("content", object.Contents{
				"en": "Do like.",
			}),
		},
		ObjectID:   post.ID,
		OwnerID:    ownerID,
		Owned:      true,
		Type:       TypeComment,
		Visibility: post.Visibility,
	}
}

func testCommentSet(ownerID uint64, post *object.Object) []*object.Object {
	return []*object.Object{
		{
			Attachments: []object.Attachment{
				object.TextAttachment("content", object.Contents{
					"en": "Do like.",
				}),
			},
			ObjectID:   post.ID,
			OwnerID:    ownerID,
			Owned:      true,
			Type:       TypeComment,
			Visibility: post.Visibility,
		},
		{
			Attachments: []object.Attachment{
				object.TextAttachment("content", object.Contents{
					"en": "Do like.",
				}),
			},
			ObjectID:   post.ID,
			OwnerID:    ownerID + 1,
			Owned:      true,
			Type:       TypeComment,
			Visibility: post.Visibility,
		},
		{
			Attachments: []object.Attachment{
				object.TextAttachment("content", object.Contents{
					"en": "Do like.",
				}),
			},
			ObjectID:   post.ID,
			OwnerID:    ownerID - 1,
			Owned:      true,
			Type:       TypeComment,
			Visibility: post.Visibility,
		},
		{
			Attachments: []object.Attachment{
				object.TextAttachment("content", object.Contents{
					"en": "Do like.",
				}),
			},
			ObjectID:   post.ID,
			OwnerID:    ownerID,
			Owned:      true,
			Type:       TypeComment,
			Visibility: post.Visibility,
		},
		{
			Attachments: []object.Attachment{
				object.TextAttachment("content", object.Contents{
					"en": "Do like.",
				}),
			},
			ObjectID:   post.ID,
			OwnerID:    ownerID,
			Owned:      true,
			Type:       TypeComment,
			Visibility: post.Visibility,
		},
	}
}

func testSetupComment() (*app.App, *user.User) {
	return &app.App{
			ID: uint64(rand.Int63()),
		}, &user.User{
			ID: uint64(rand.Int63()),
		}
}
