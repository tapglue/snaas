package core

import (
	"github.com/tapglue/snaas/service/app"
	"github.com/tapglue/snaas/service/connection"
	"github.com/tapglue/snaas/service/object"
	"github.com/tapglue/snaas/service/user"
)

const (
	// TypeComment identifies a comment object.
	TypeComment = "tg_comment"

	attachmentContent = "content"
)

// CommentFeed is a collection of comments with their referneced users.
type CommentFeed struct {
	Comments object.List
	UserMap  user.Map
}

// CommentCreateFunc creates a new comment on behalf of the origin uesr on the
// given Post id.
type CommentCreateFunc func(
	currentApp *app.App,
	origin Origin,
	postID uint64,
	input *object.Object,
) (*object.Object, error)

// CommentCreate creates a new comment on behalf of the origin uesr on the
// given Post id.
func CommentCreate(
	connections connection.Service,
	objects object.Service,
) CommentCreateFunc {
	return func(
		currentApp *app.App,
		origin Origin,
		postID uint64,
		input *object.Object,
	) (*object.Object, error) {
		err := constrainCommentPrivate(origin, input.Private)
		if err != nil {
			return nil, err
		}

		ps, err := objects.Query(currentApp.Namespace(), object.QueryOptions{
			ID:    &postID,
			Owned: &defaultOwned,
			Types: []string{TypePost},
		})
		if err != nil {
			return nil, err
		}

		if len(ps) == 0 {
			return nil, ErrNotFound
		}

		post := ps[0]

		if err := constrainCommentRestriction(post.Restrictions); err != nil {
			return nil, err
		}

		comment := &object.Object{
			Attachments: []object.Attachment{
				object.TextAttachment(
					attachmentContent,
					input.Attachments[0].Contents,
				),
			},
			ObjectID:   postID,
			OwnerID:    origin.UserID,
			Owned:      true,
			Private:    input.Private,
			Type:       TypeComment,
			Visibility: post.Visibility,
		}

		if err := comment.Validate(); err != nil {
			return nil, wrapError(ErrInvalidEntity, "invalid Comment: %s", err)
		}

		if err := isPostVisible(connections, currentApp, ps[0], origin.UserID); err != nil {
			return nil, err
		}

		return objects.Put(currentApp.Namespace(), comment)
	}
}

// CommentDeleteFunc flags the Comment as deleted.
type CommentDeleteFunc func(
	currentApp *app.App,
	origin uint64,
	postID uint64,
	commentID uint64,
) error

// CommentDelete flags the Comment as deleted.
func CommentDelete(
	connections connection.Service,
	objects object.Service,
) CommentDeleteFunc {
	return func(
		currentApp *app.App,
		origin uint64,
		postID uint64,
		commentID uint64,
	) error {
		cs, err := objects.Query(currentApp.Namespace(), object.QueryOptions{
			ID: &commentID,
			ObjectIDs: []uint64{
				postID,
			},
			OwnerIDs: []uint64{
				origin,
			},
			Types: []string{
				TypeComment,
			},
			Owned: &defaultOwned,
		})
		if err != nil {
			return err
		}

		// A delete should be idempotent and always succeed.
		if len(cs) != 1 {
			return nil
		}

		cs[0].Deleted = true

		_, err = objects.Put(currentApp.Namespace(), cs[0])
		if err != nil {
			return err
		}

		return nil
	}
}

// CommentListFunc returns all comemnts for the given post id.
type CommentListFunc func(
	currentApp *app.App,
	origin uint64,
	postID uint64,
	opts object.QueryOptions,
) (*CommentFeed, error)

// CommentList returns all comemnts for the given post id.
func CommentList(
	connections connection.Service,
	objects object.Service,
	users user.Service,
) CommentListFunc {
	return func(
		currentApp *app.App,
		origin uint64,
		postID uint64,
		opts object.QueryOptions,
	) (*CommentFeed, error) {
		ps, err := objects.Query(currentApp.Namespace(), object.QueryOptions{
			ID:    &postID,
			Owned: &defaultOwned,
			Types: []string{TypePost},
		})
		if err != nil {
			return nil, err
		}

		if len(ps) == 0 {
			return nil, ErrNotFound
		}

		if err := isPostVisible(connections, currentApp, ps[0], origin); err != nil {
			return nil, err
		}

		cs, err := objects.Query(currentApp.Namespace(), object.QueryOptions{
			Before: opts.Before,
			Limit:  opts.Limit,
			ObjectIDs: []uint64{
				postID,
			},
			Types: []string{
				TypeComment,
			},
			Owned: &defaultOwned,
		})
		if err != nil {
			return nil, err
		}

		um, err := user.MapFromIDs(users, currentApp.Namespace(), cs.OwnerIDs()...)
		if err != nil {
			return nil, err
		}

		for _, u := range um {
			err = enrichRelation(connections, currentApp, origin, u)
			if err != nil {
				return nil, err
			}
		}

		return &CommentFeed{Comments: cs, UserMap: um}, nil
	}
}

// CommentRetrieveFunc returns the comment for the given id.
type CommentRetrieveFunc func(
	currentApp *app.App,
	origin uint64,
	postID, commentID uint64,
) (*object.Object, error)

// CommentRetrieve returns the comment for the given id.
func CommentRetrieve(
	objects object.Service,
) CommentRetrieveFunc {
	return func(
		currentApp *app.App,
		origin uint64,
		postID, commentID uint64,
	) (*object.Object, error) {
		cs, err := objects.Query(currentApp.Namespace(), object.QueryOptions{
			ID: &commentID,
			ObjectIDs: []uint64{
				postID,
			},
			OwnerIDs: []uint64{
				origin,
			},
			Types: []string{
				TypeComment,
			},
			Owned: &defaultOwned,
		})
		if err != nil {
			return nil, err
		}

		if len(cs) != 1 {
			return nil, ErrNotFound
		}

		return cs[0], nil
	}
}

// CommentUpdateFunc replaces the given comment with new values.
type CommentUpdateFunc func(
	currentApp *app.App,
	origin Origin,
	postID, commentID uint64,
	new *object.Object,
) (*object.Object, error)

// CommentUpdate replaces the given comment with new values.
func CommentUpdate(
	objects object.Service,
) CommentUpdateFunc {
	return func(
		currentApp *app.App,
		origin Origin,
		postID, commentID uint64,
		new *object.Object,
	) (*object.Object, error) {
		err := constrainCommentPrivate(origin, new.Private)
		if err != nil {
			return nil, err
		}

		cs, err := objects.Query(currentApp.Namespace(), object.QueryOptions{
			ID: &commentID,
			ObjectIDs: []uint64{
				postID,
			},
			OwnerIDs: []uint64{
				origin.UserID,
			},
			Owned: &defaultOwned,
			Types: []string{
				TypeComment,
			},
		})
		if err != nil {
			return nil, err
		}

		if len(cs) != 1 {
			return nil, ErrNotFound
		}

		old := cs[0]

		old.Attachments = []object.Attachment{
			object.TextAttachment(
				attachmentContent,
				new.Attachments[0].Contents,
			),
		}

		if origin.IsBackend() && new.Private != nil {
			old.Private = new.Private
		}

		return objects.Put(currentApp.Namespace(), old)
	}
}

// IsComment indicates if Object is a comment.
func IsComment(o *object.Object) bool {
	if o.Type != TypeComment {
		return false
	}

	return o.Owned
}

func constrainCommentPrivate(origin Origin, private *object.Private) error {
	if !origin.IsBackend() && private != nil {
		return wrapError(ErrUnauthorized,
			"private can only be set by backend integration",
		)
	}

	return nil
}

func constrainCommentRestriction(restrictions *object.Restrictions) error {
	if restrictions != nil && restrictions.Comment {
		return wrapError(
			ErrUnauthorized,
			"comments not allowed for this post",
		)
	}

	return nil
}
