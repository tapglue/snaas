package core

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"

	"golang.org/x/text/language"

	"github.com/tapglue/snaas/service/app"
	"github.com/tapglue/snaas/service/connection"
	"github.com/tapglue/snaas/service/event"
	"github.com/tapglue/snaas/service/object"
	"github.com/tapglue/snaas/service/reaction"
	"github.com/tapglue/snaas/service/rule"
	"github.com/tapglue/snaas/service/user"
)

func TestPipelineConnectionCondFrom(t *testing.T) {
	var (
		currentApp  = testApp()
		connections = connection.MemService()
		users       = user.MemService()
	)

	// Create friend target.
	target, err := users.Put(currentApp.Namespace(), testUser())
	if err != nil {
		t.Fatal(err)
	}

	// Create friend origin.
	origin, err := users.Put(currentApp.Namespace(), testUser())
	if err != nil {
		t.Fatal(err)
	}

	// Friend request.
	old, err := connections.Put(currentApp.Namespace(), &connection.Connection{
		Enabled: true,
		FromID:  origin.ID,
		State:   connection.StatePending,
		ToID:    target.ID,
		Type:    connection.TypeFriend,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Confirm request.
	new, err := connections.Put(currentApp.Namespace(), &connection.Connection{
		Enabled: true,
		FromID:  origin.ID,
		State:   connection.StateConfirmed,
		ToID:    target.ID,
		Type:    connection.TypeFriend,
	})
	if err != nil {
		t.Fatal(err)
	}

	var (
		enabled          = true
		ruleConnectionTo = &rule.Rule{
			Criteria: &rule.CriteriaConnection{
				New: &connection.QueryOptions{
					Enabled: &enabled,
					States: []connection.State{
						connection.StateConfirmed,
					},
					Types: []connection.Type{
						connection.TypeFriend,
					},
				},
				Old: &connection.QueryOptions{
					Enabled: &enabled,
					States: []connection.State{
						connection.StatePending,
					},
					Types: []connection.Type{
						connection.TypeFriend,
					},
				},
			},
			Recipients: rule.Recipients{
				{
					Query: map[string]string{
						"userFrom": "",
					},
					Templates: map[string]string{
						"en": "{{.To.Username}} accepted your friend request",
					},
					URN: "tapglue/users/{{.To.ID}}",
				},
			},
		}
	)

	want := Messages{
		{
			Messages: map[string]string{
				language.English.String(): fmt.Sprintf("%s accepted your friend request", target.Username),
			},
			Recipient: origin.ID,
			URN:       fmt.Sprintf("tapglue/users/%d", target.ID),
		},
	}

	have, err := PipelineConnection(users)(currentApp, &connection.StateChange{New: new, Old: old}, ruleConnectionTo)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(have, want) {
		t.Errorf("have %#v, want %#v", have, want)
	}
}

func TestPipelineConnectionCondTo(t *testing.T) {
	var (
		currentApp  = testApp()
		connections = connection.MemService()
		users       = user.MemService()
	)

	// Create friend target.
	target, err := users.Put(currentApp.Namespace(), testUser())
	if err != nil {
		t.Fatal(err)
	}

	// Create friend origin.
	origin, err := users.Put(currentApp.Namespace(), testUser())
	if err != nil {
		t.Fatal(err)
	}

	// Friend request.
	con, err := connections.Put(currentApp.Namespace(), &connection.Connection{
		Enabled: true,
		FromID:  origin.ID,
		State:   connection.StatePending,
		ToID:    target.ID,
		Type:    connection.TypeFriend,
	})
	if err != nil {
		t.Fatal(err)
	}

	var (
		enabled          = true
		ruleConnectionTo = &rule.Rule{
			Criteria: &rule.CriteriaConnection{
				New: &connection.QueryOptions{
					Enabled: &enabled,
					States: []connection.State{
						connection.StatePending,
					},
					Types: []connection.Type{
						connection.TypeFriend,
					},
				},
				Old: nil,
			},
			Recipients: rule.Recipients{
				{
					Query: map[string]string{
						"userTo": "",
					},
					Templates: map[string]string{
						"en": "{{.From.Username}} sent you a friend request",
					},
					URN: "tapglue/users/{{.From.ID}}",
				},
			},
		}
	)

	want := Messages{
		{
			Messages: map[string]string{
				language.English.String(): fmt.Sprintf("%s sent you a friend request", origin.Username),
			},
			Recipient: target.ID,
			URN:       fmt.Sprintf("tapglue/users/%d", origin.ID),
		},
	}

	have, err := PipelineConnection(users)(currentApp, &connection.StateChange{New: con}, ruleConnectionTo)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(have, want) {
		t.Errorf("have %#v, want %#v", have, want)
	}
}

func TestPipelineReactionCondParentOwner(t *testing.T) {
	var (
		currentApp = testApp()
		objects    = object.MemService()
		reactions  = reaction.MemService()
		users      = user.MemService()
	)

	// Creat Post Owner.
	postOwner, err := users.Put(currentApp.Namespace(), testUser())
	if err != nil {
		t.Fatal(err)
	}

	// Create Post.
	post, err := objects.Put(currentApp.Namespace(), testPost(postOwner.ID).Object)
	if err != nil {
		t.Fatal(err)
	}

	// Create liker.
	liker, err := users.Put(currentApp.Namespace(), testUser())
	if err != nil {
		t.Fatal(err)
	}

	// Create like.
	like, err := reactions.Put(currentApp.Namespace(), &reaction.Reaction{
		ObjectID: post.ID,
		OwnerID:  liker.ID,
		Type:     reaction.TypeLike,
	})
	if err != nil {
		t.Fatal(err)
	}

	var (
		deleted                 = false
		ruleReactionParentOwner = &rule.Rule{
			Criteria: &rule.CriteriaReaction{
				New: &reaction.QueryOptions{
					Deleted: &deleted,
					Types: []reaction.Type{
						reaction.TypeLike,
					},
				},
				Old: nil,
			},
			Recipients: rule.Recipients{
				{
					Query: map[string]string{
						"parentOwner": "",
					},
					Templates: map[string]string{
						"en": "{{.Owner.Username}} liked your post",
					},
					URN: "tapglue/users/{{.Owner.ID}}",
				},
			},
		}
	)

	want := Messages{
		{
			Messages: map[string]string{
				language.English.String(): fmt.Sprintf("%s liked your post", liker.Username),
			},
			Recipient: postOwner.ID,
			URN:       fmt.Sprintf("tapglue/users/%d", liker.ID),
		},
	}

	have, err := PipelineReaction(objects, users)(currentApp, &reaction.StateChange{New: like}, ruleReactionParentOwner)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(have, want) {
		t.Errorf("have %#v, want %#v", have, want)
	}
}

func TestPipelineEventCondParentOwner(t *testing.T) {
	var (
		currentApp = testApp()
		events     = event.MemService()
		objects    = object.MemService()
		users      = user.MemService()
	)

	// Creat Post Owner.
	postOwner, err := users.Put(currentApp.Namespace(), testUser())
	if err != nil {
		t.Fatal(err)
	}

	// Create Post.
	post, err := objects.Put(currentApp.Namespace(), testPost(postOwner.ID).Object)
	if err != nil {
		t.Fatal(err)
	}

	// Create liker.
	liker, err := users.Put(currentApp.Namespace(), testUser())
	if err != nil {
		t.Fatal(err)
	}

	// Create like.
	like, err := events.Put(currentApp.Namespace(), &event.Event{
		Enabled:  true,
		ObjectID: post.ID,
		Owned:    true,
		Type:     TypeLike,
		UserID:   liker.ID,
	})
	if err != nil {
		t.Fatal(err)
	}

	var (
		enabled              = true
		ruleEventParentOwner = &rule.Rule{
			Criteria: &rule.CriteriaEvent{
				New: &event.QueryOptions{
					Enabled: &enabled,
					Owned:   &enabled,
					Types: []string{
						TypeLike,
					},
				},
				Old: nil,
			},
			Recipients: rule.Recipients{
				{
					Query: map[string]string{
						"parentOwner": "",
					},
					Templates: map[string]string{
						"en": "{{.Owner.Username}} liked your post",
					},
					URN: "tapglue/users/{{.Owner.ID}}",
				},
			},
		}
	)

	want := Messages{
		{
			Messages: map[string]string{
				language.English.String(): fmt.Sprintf("%s liked your post", liker.Username),
			},
			Recipient: postOwner.ID,
			URN:       fmt.Sprintf("tapglue/users/%d", liker.ID),
		},
	}

	have, err := PipelineEvent(objects, users)(currentApp, &event.StateChange{New: like}, ruleEventParentOwner)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(have, want) {
		t.Errorf("have %#v, want %#v", have, want)
	}
}

func TestPipelineObjectCondFriends(t *testing.T) {
	var (
		currentApp  = testApp()
		connections = connection.MemService()
		objects     = object.MemService()
		users       = user.MemService()
	)

	// Creat Post Owner.
	postOwner, err := users.Put(currentApp.Namespace(), testUser())
	if err != nil {
		t.Fatal(err)
	}

	// Create Post.
	post, err := objects.Put(currentApp.Namespace(), testPost(postOwner.ID).Object)
	if err != nil {
		t.Fatal(err)
	}

	// Create frist friend.
	friend1, err := users.Put(currentApp.Namespace(), testUser())
	if err != nil {
		t.Fatal(err)
	}

	// Create first connection.
	_, err = connections.Put(currentApp.Namespace(), &connection.Connection{
		Enabled: true,
		FromID:  postOwner.ID,
		State:   connection.StateConfirmed,
		ToID:    friend1.ID,
		Type:    connection.TypeFriend,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create second friend.
	friend2, err := users.Put(currentApp.Namespace(), testUser())
	if err != nil {
		t.Fatal(err)
	}

	// Create second connection.
	_, err = connections.Put(currentApp.Namespace(), &connection.Connection{
		Enabled: true,
		FromID:  friend2.ID,
		State:   connection.StateConfirmed,
		ToID:    postOwner.ID,
		Type:    connection.TypeFriend,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create creep who is not friends with post owner.
	_, err = users.Put(currentApp.Namespace(), testUser())
	if err != nil {
		t.Fatal(err)
	}

	ruleObjectOwner := &rule.Rule{
		Criteria: &rule.CriteriaObject{
			New: &object.QueryOptions{
				Owned: &defaultOwned,
				Tags:  []string{"review"},
				Types: []string{TypePost},
			},
			Old: nil,
		},
		Recipients: rule.Recipients{
			{
				Query: map[string]string{
					"ownerFriends": "",
				},
				Templates: map[string]string{
					"en": "{{.Owner.Username}} just added a review",
				},
				URN: "tapglue/posts/{{.Object.ID}}",
			},
		},
	}

	want := Messages{
		{
			Recipient: friend2.ID,
			Messages: map[string]string{
				language.English.String(): fmt.Sprintf("%s just added a review", postOwner.Username),
			},
			URN: fmt.Sprintf("tapglue/posts/%d", post.ID),
		},
		{
			Recipient: friend1.ID,
			Messages: map[string]string{
				language.English.String(): fmt.Sprintf("%s just added a review", postOwner.Username),
			},
			URN: fmt.Sprintf("tapglue/posts/%d", post.ID),
		},
	}

	have, err := PipelineObject(
		connections,
		objects,
		users,
	)(currentApp, &object.StateChange{New: post}, ruleObjectOwner)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(have, want) {
		t.Errorf("have %#v, want %#v", have, want)
	}
}

func TestPipelineObjectCondObjectOwner(t *testing.T) {
	var (
		currentApp  = testApp()
		connections = connection.MemService()
		objects     = object.MemService()
		users       = user.MemService()
	)

	// Creat Post Owner.
	postOwner, err := users.Put(currentApp.Namespace(), testUser())
	if err != nil {
		t.Fatal(err)
	}

	// Create Post.
	post, err := objects.Put(currentApp.Namespace(), testPost(postOwner.ID).Object)
	if err != nil {
		t.Fatal(err)
	}

	// Create frist commenter.
	commenter1, err := users.Put(currentApp.Namespace(), testUser())
	if err != nil {
		t.Fatal(err)
	}

	// Create first comment.
	_, err = objects.Put(currentApp.Namespace(), testComment(commenter1.ID, post))
	if err != nil {
		t.Fatal(err)
	}

	// Create second commenter.
	commenter2, err := users.Put(currentApp.Namespace(), testUser())
	if err != nil {
		t.Fatal(err)
	}

	// Create second comment.
	_, err = objects.Put(currentApp.Namespace(), testComment(commenter2.ID, post))
	if err != nil {
		t.Fatal(err)
	}

	// Create final commenter.
	commenter3, err := users.Put(currentApp.Namespace(), testUser())
	if err != nil {
		t.Fatal(err)
	}

	// Craete final comment, which we test against.
	comment3, err := objects.Put(currentApp.Namespace(), testComment(commenter3.ID, post))
	if err != nil {
		t.Fatal(err)
	}

	ruleObjectOwner := &rule.Rule{
		Criteria: &rule.CriteriaObject{
			New: &object.QueryOptions{
				Owned: &defaultOwned,
				Types: []string{TypeComment},
			},
			Old: nil,
		},
		Recipients: rule.Recipients{
			{
				Query: map[string]string{
					"objectOwner": `{ "object_ids": [ {{.Parent.ID}} ], "owned": true, "types": [ "tg_comment" ]}`,
				},
				Templates: map[string]string{
					"en": "{{.Owner.Username}} also commented on {{.ParentOwner.Username}}s post",
				},
				URN: "tapglue/posts/{{.Parent.ID}}/comments/{{.Object.ID}}",
			},
		},
	}

	want := Messages{
		{
			Recipient: commenter2.ID,
			Messages: map[string]string{
				language.English.String(): fmt.Sprintf("%s also commented on %ss post", commenter3.Username, postOwner.Username),
			},
			URN: fmt.Sprintf("tapglue/posts/%d/comments/%d", post.ID, comment3.ID),
		},
		{
			Recipient: commenter1.ID,
			Messages: map[string]string{
				language.English.String(): fmt.Sprintf("%s also commented on %ss post", commenter3.Username, postOwner.Username),
			},
			URN: fmt.Sprintf("tapglue/posts/%d/comments/%d", post.ID, comment3.ID),
		},
	}

	have, err := PipelineObject(
		connections,
		objects,
		users,
	)(currentApp, &object.StateChange{New: comment3}, ruleObjectOwner)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(have, want) {
		t.Errorf("have %#v, want %#v", have, want)
	}
}

func TestPipelineObjectCondOwner(t *testing.T) {
	var (
		currentApp  = testApp()
		connections = connection.MemService()
		objects     = object.MemService()
		users       = user.MemService()
	)

	// Creat Post Owner.
	postOwner, err := users.Put(currentApp.Namespace(), testUser())
	if err != nil {
		t.Fatal(err)
	}

	// Create Post.
	post, err := objects.Put(currentApp.Namespace(), testPost(postOwner.ID).Object)
	if err != nil {
		t.Fatal(err)
	}

	// Create commenter.
	commenter, err := users.Put(currentApp.Namespace(), testUser())
	if err != nil {
		t.Fatal(err)
	}

	// Create comment.
	comment, err := objects.Put(currentApp.Namespace(), testComment(commenter.ID, post))
	if err != nil {
		t.Fatal(err)
	}

	ruleObjectOwner := &rule.Rule{
		Criteria: &rule.CriteriaObject{
			New: &object.QueryOptions{
				Owned: &defaultOwned,
				Types: []string{TypeComment},
			},
			Old: nil,
		},
		Recipients: rule.Recipients{
			{
				Query: map[string]string{
					"parentOwner": "",
				},
				Templates: map[string]string{
					"en": "{{.Owner.Username}} commented on your post",
				},
				URN: "tapglue/posts/{{.Parent.ID}}/comments/{{.Object.ID}}",
			},
		},
	}

	want := Messages{
		{
			Recipient: postOwner.ID,
			Messages: map[string]string{
				language.English.String(): fmt.Sprintf("%s commented on your post", commenter.Username),
			},
			URN: fmt.Sprintf("tapglue/posts/%d/comments/%d", post.ID, comment.ID),
		},
	}

	have, err := PipelineObject(
		connections,
		objects,
		users,
	)(currentApp, &object.StateChange{New: comment}, ruleObjectOwner)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(have, want) {
		t.Errorf("have %#v, want %#v", have, want)
	}
}

func testApp() *app.App {
	return &app.App{
		ID: uint64(rand.Int63()),
	}
}
