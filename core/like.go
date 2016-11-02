package core

import (
	"sort"

	"github.com/tapglue/api/service/app"
	"github.com/tapglue/api/service/connection"
	"github.com/tapglue/api/service/event"
	"github.com/tapglue/api/service/object"
	"github.com/tapglue/api/service/user"
)

const (
	// TypeLike identifies an event as a Like.
	TypeLike = "tg_like"
)

var defaultEnabled = true

// LikeFeed is a collection of likes with their referenced users.
type LikeFeed struct {
	Likes   event.List
	PostMap PostMap
	UserMap user.Map
}

// LikeCreate checks if a like for the owner on the post exists and if not
// creates a new event for it.
type LikeCreateFunc func(
	currentApp *app.App,
	origin uint64,
	postID uint64,
) (*event.Event, error)

// Create checks if a like for the owner on the post exists and if not creates
// a new event for it.
func LikeCreate(
	connections connection.Service,
	events event.Service,
	objects object.Service,
) LikeCreateFunc {
	return func(
		currentApp *app.App,
		origin uint64,
		postID uint64,
	) (*event.Event, error) {
		ps, err := objects.Query(currentApp.Namespace(), object.QueryOptions{
			ID:    &postID,
			Owned: &defaultOwned,
			Types: []string{
				TypePost,
			},
		})
		if err != nil {
			return nil, err
		}

		if len(ps) == 0 {
			return nil, ErrNotFound
		}

		post := ps[0]

		if err := constrainLikeRestriction(post.Restrictions); err != nil {
			return nil, err
		}

		if err := isPostVisible(connections, currentApp, post, origin); err != nil {
			return nil, err
		}

		es, err := events.Query(currentApp.Namespace(), event.QueryOptions{
			ObjectIDs: []uint64{
				postID,
			},
			Owned: &defaultOwned,
			Types: []string{
				TypeLike,
			},
			UserIDs: []uint64{
				origin,
			},
		})
		if err != nil {
			return nil, err
		}

		if len(es) > 0 && es[0].Enabled == true {
			return es[0], nil
		}

		var like *event.Event

		if len(es) == 0 {
			like = &event.Event{
				Enabled:    true,
				ObjectID:   postID,
				Owned:      true,
				Type:       TypeLike,
				UserID:     origin,
				Visibility: event.Visibility(post.Visibility),
			}
		} else {
			like = es[0]
			like.Enabled = true
		}

		like, err = events.Put(currentApp.Namespace(), like)
		if err != nil {
			return nil, err
		}

		return like, nil
	}
}

// LikeDeleteFunc removes an existing like event for the given user on the post.
type LikeDeleteFunc func(
	currentApp *app.App,
	origin uint64,
	postID uint64,
) error

// LikeDelete removes an existing like event for the given user on the post.
func LikeDelete(
	connections connection.Service,
	events event.Service,
	objects object.Service,
) LikeDeleteFunc {
	return func(
		currentApp *app.App,
		origin uint64,
		postID uint64,
	) error {
		ps, err := objects.Query(currentApp.Namespace(), object.QueryOptions{
			ID:    &postID,
			Owned: &defaultOwned,
			Types: []string{
				TypePost,
			},
		})
		if err != nil {
			return err
		}

		if len(ps) != 1 {
			return ErrNotFound
		}

		if err := isPostVisible(connections, currentApp, ps[0], origin); err != nil {
			return err
		}

		es, err := events.Query(currentApp.Namespace(), event.QueryOptions{
			Enabled: &defaultEnabled,
			ObjectIDs: []uint64{
				postID,
			},
			Owned: &defaultOwned,
			Types: []string{
				TypeLike,
			},
			UserIDs: []uint64{
				origin,
			},
		})
		if err != nil {
			return err
		}

		if len(es) == 0 {
			return nil
		}

		like := es[0]
		like.Enabled = false

		like, err = events.Put(currentApp.Namespace(), like)
		if err != nil {
			return err
		}

		return nil
	}
}

// LikeListFunc returns all likes for the given post.
type LikeListFunc func(
	currentApp *app.App,
	origin uint64,
	postID uint64,
	opts event.QueryOptions,
) (*LikeFeed, error)

// LikeList returns all likes for the given post.
func LikeList(
	connections connection.Service,
	events event.Service,
	objects object.Service,
	users user.Service,
) LikeListFunc {
	return func(
		currentApp *app.App,
		origin uint64,
		postID uint64,
		opts event.QueryOptions,
	) (*LikeFeed, error) {
		ps, err := objects.Query(currentApp.Namespace(), object.QueryOptions{
			ID:    &postID,
			Owned: &defaultOwned,
			Types: []string{
				TypePost,
			},
		})
		if err != nil {
			return nil, err
		}

		if len(ps) != 1 {
			return nil, ErrNotFound
		}

		if err := isPostVisible(connections, currentApp, ps[0], origin); err != nil {
			return nil, err
		}

		es, err := events.Query(currentApp.Namespace(), event.QueryOptions{
			Before:  opts.Before,
			Enabled: &defaultEnabled,
			Limit:   opts.Limit,
			ObjectIDs: []uint64{
				postID,
			},
			Owned: &defaultOwned,
			Types: []string{
				TypeLike,
			},
		})

		um, err := user.MapFromIDs(users, currentApp.Namespace(), es.UserIDs()...)
		if err != nil {
			return nil, err
		}

		return &LikeFeed{
			Likes:   es,
			UserMap: um,
		}, nil
	}
}

// LikesUserFunc returns all Likes for the given user.
type LikesUserFunc func(
	app *app.App,
	origin, userID uint64,
	opts event.QueryOptions,
) (*LikeFeed, error)

// LikesUserFunc returns all Likes for the given user.
func LikesUser(
	connections connection.Service,
	events event.Service,
	objects object.Service,
	users user.Service,
) LikesUserFunc {
	return func(
		currentApp *app.App,
		origin, userID uint64,
		opts event.QueryOptions,
	) (*LikeFeed, error) {
		opts.Enabled = &defaultEnabled
		opts.Types = []string{TypeLike}
		opts.UserIDs = []uint64{userID}

		r, err := queryRelation(connections, currentApp, origin, userID)
		if err != nil {
			return nil, err
		}

		opts.Visibilities = eventVisibilitiesForRelation(r)

		ls, err := events.Query(currentApp.Namespace(), opts)
		if err != nil {
			return nil, err
		}

		ps, err := extractPosts(objects, currentApp, ls)
		if err != nil {
			return nil, err
		}

		ls = filter(ls, conditionPostMissing(ps.toMap()))

		sort.Sort(ls)

		if len(ls) > opts.Limit {
			ls = ls[:opts.Limit]
		}

		ps = postsByEvents(ls, ps.toMap())

		err = enrichCounts(events, objects, currentApp, ps)
		if err != nil {
			return nil, err
		}

		if !r.isSelf {
			err := enrichIsLiked(events, currentApp, origin, ps)
			if err != nil {
				return nil, err
			}
		}

		um, err := fillupUsersForEvents(users, currentApp, origin, user.Map{}, ls)
		if err != nil {
			return nil, err
		}

		um, err = fillupUsersForPosts(users, currentApp, origin, um, ps)
		if err != nil {
			return nil, err
		}

		return &LikeFeed{
			Likes:   ls,
			PostMap: ps.toMap(),
			UserMap: um,
		}, nil
	}
}

func constrainLikeRestriction(restrictions *object.Restrictions) error {
	if restrictions != nil && restrictions.Like {
		return wrapError(
			ErrUnauthorized,
			"likes not allowed for this post",
		)
	}

	return nil
}

func eventVisibilitiesForRelation(r *relation) []event.Visibility {
	if r.isSelf {
		return []event.Visibility{
			event.VisibilityPrivate,
			event.VisibilityConnection,
			event.VisibilityPublic,
			event.VisibilityGlobal,
		}
	}

	if r.isFollowing || r.isFriend {
		return []event.Visibility{
			event.VisibilityConnection,
			event.VisibilityPublic,
			event.VisibilityGlobal,
		}
	}

	return []event.Visibility{
		event.VisibilityPublic,
		event.VisibilityGlobal,
	}
}

func postsByEvents(es event.List, pm PostMap) PostList {
	ps := PostList{}

	for _, event := range es {
		p, ok := pm[event.ObjectID]
		if ok {
			ps = append(ps, p)
		}
	}

	return ps
}
