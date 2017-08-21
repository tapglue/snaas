package core

import (
	"github.com/tapglue/snaas/service/app"
	"github.com/tapglue/snaas/service/connection"
	"github.com/tapglue/snaas/service/event"
	"github.com/tapglue/snaas/service/object"
	"github.com/tapglue/snaas/service/reaction"
	"github.com/tapglue/snaas/service/user"
)

// ReactionCreateFunc checks if a Reaction of the given type already exists on
// the post for the origin.
type ReactionCreateFunc func(
	currentApp *app.App,
	origin uint64,
	postID uint64,
	reactionType reaction.Type,
) (*reaction.Reaction, error)

// ReactionCreate checks if a Reaction of the given type already exists on the
// post for the origin.
func ReactionCreate(
	connections connection.Service,
	objects object.Service,
	reactions reaction.Service,
) ReactionCreateFunc {
	return func(
		currentApp *app.App,
		origin uint64,
		postID uint64,
		reactionType reaction.Type,
	) (*reaction.Reaction, error) {
		p, err := PostFetch(objects)(currentApp, postID)
		if err != nil {
			return nil, err
		}

		if err := constrainLikeRestriction(p.Restrictions); err != nil {
			return nil, err
		}

		if err := isPostVisible(connections, currentApp, p.Object, origin); err != nil {
			return nil, err
		}

		rs, err := reactions.Query(currentApp.Namespace(), reaction.QueryOptions{
			ObjectIDs: []uint64{
				postID,
			},
			OwnerIDs: []uint64{
				origin,
			},
			Types: []reaction.Type{
				reactionType,
			},
		})
		if err != nil {
			return nil, err
		}

		if len(rs) == 1 && rs[0].Deleted == false {
			return rs[0], nil
		}

		var r *reaction.Reaction

		if len(rs) == 1 {
			r = rs[0]
			r.Deleted = false
		} else {
			r = &reaction.Reaction{
				Deleted:  false,
				ObjectID: postID,
				OwnerID:  origin,
				Type:     reactionType,
			}
		}

		return reactions.Put(currentApp.Namespace(), r)
	}
}

// ReactionDeleteFunc remvoes an existing Reaction from the Post.
type ReactionDeleteFunc func(
	currentApp *app.App,
	origin, postID uint64,
	reactionType reaction.Type,
) error

// ReactionDelete remvoes an existing Reaction from the Post.
func ReactionDelete(
	connections connection.Service,
	objects object.Service,
	reactions reaction.Service,
) ReactionDeleteFunc {
	return func(
		currentApp *app.App,
		origin, postID uint64,
		reactionType reaction.Type,
	) error {
		p, err := PostFetch(objects)(currentApp, postID)
		if err != nil {
			return err
		}

		if err := isPostVisible(connections, currentApp, p.Object, origin); err != nil {
			return err
		}

		rs, err := reactions.Query(currentApp.Namespace(), reaction.QueryOptions{
			Deleted: &defaultDeleted,
			ObjectIDs: []uint64{
				postID,
			},
			OwnerIDs: []uint64{
				origin,
			},
			Types: []reaction.Type{
				reactionType,
			},
		})
		if err != nil {
			return err
		}

		if len(rs) == 0 {
			return nil
		}

		reaction := rs[0]
		reaction.Deleted = true

		_, err = reactions.Put(currentApp.Namespace(), reaction)

		return err
	}
}

// ReactionListPostFunc returns all reactions for a Post.
type ReactionListPostFunc func(
	currentApp *app.App,
	origin, postID uint64,
	opts reaction.QueryOptions,
) (*ReactionFeed, error)

// ReactionListPost returns all reactions for a Post.
func ReactionListPost(
	connections connection.Service,
	events event.Service,
	objects object.Service,
	reactions reaction.Service,
	users user.Service,
) ReactionListPostFunc {
	return func(
		currentApp *app.App,
		origin, postID uint64,
		opts reaction.QueryOptions,
	) (*ReactionFeed, error) {
		p, err := PostFetch(objects)(currentApp, postID)
		if err != nil {
			return nil, err
		}

		if err := isPostVisible(connections, currentApp, p.Object, origin); err != nil {
			return nil, err
		}

		rs, err := reactions.Query(currentApp.Namespace(), reaction.QueryOptions{
			Before:  opts.Before,
			Deleted: &defaultDeleted,
			Limit:   opts.Limit,
			ObjectIDs: []uint64{
				postID,
			},
		})
		if err != nil {
			return nil, err
		}

		um, err := user.MapFromIDs(users, currentApp.Namespace(), append(rs.OwnerIDs(), p.OwnerID)...)
		if err != nil {
			return nil, err
		}

		for _, u := range um {
			err := enrichRelation(connections, currentApp, origin, u)
			if err != nil {
				return nil, err
			}
		}

		err = enrichCounts(events, objects, reactions, currentApp, PostList{p})
		if err != nil {
			return nil, err
		}

		return &ReactionFeed{
			PostMap: PostMap{
				p.ID: p,
			},
			Reactions: rs,
			UserMap:   um,
		}, nil
	}
}

// ReactionFeed is a collection of Reactions with their referenced Users and
// Posts.
type ReactionFeed struct {
	Reactions reaction.List
	PostMap   PostMap
	UserMap   user.Map
}
