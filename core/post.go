package core

import (
	"github.com/tapglue/snaas/service/app"
	"github.com/tapglue/snaas/service/connection"
	"github.com/tapglue/snaas/service/event"
	"github.com/tapglue/snaas/service/object"
	"github.com/tapglue/snaas/service/reaction"
	"github.com/tapglue/snaas/service/user"
)

// TypePost identifies an object as a Post.
const TypePost = "tg_post"

var defaultOwned = true

// HasReacted bundles the binary state of a user per reaction.
type HasReacted struct {
	Like  bool `json:"like"`
	Love  bool `json:"love"`
	Haha  bool `json:"haha"`
	Wow   bool `json:"wow"`
	Sad   bool `json:"sad"`
	Angry bool `json:"angry"`
}

// Post is the intermediate representation for posts.
type Post struct {
	Counts     PostCounts
	IsLiked    bool
	HasReacted HasReacted

	*object.Object
}

// PostCounts bundles all connected entity counts.
type PostCounts struct {
	Comments       uint64
	Likes          int
	ReactionCounts reaction.Counts
}

// PostFeed is the composite answer for post list methods.
type PostFeed struct {
	Posts   PostList
	UserMap user.Map
}

// PostMap is the user collection indexed by their ids.
type PostMap map[uint64]*Post

// PostList is a collection of Post.
type PostList []*Post

func (ps PostList) toMap() PostMap {
	pm := PostMap{}

	for _, post := range ps {
		pm[post.ID] = post
	}

	return pm
}

func (ps PostList) Len() int {
	return len(ps)
}

func (ps PostList) Less(i, j int) bool {
	return ps[i].CreatedAt.After(ps[j].CreatedAt)
}

func (ps PostList) Swap(i, j int) {
	ps[i], ps[j] = ps[j], ps[i]
}

// IDs returns the id of all posts in the list.
func (ps PostList) IDs() []uint64 {
	ids := []uint64{}

	for _, p := range ps {
		ids = append(ids, p.ID)
	}

	return ids
}

// OwnerIDs extracts the OwnerID of every post.
func (ps PostList) OwnerIDs() []uint64 {
	ids := []uint64{}

	for _, p := range ps {
		ids = append(ids, p.OwnerID)
	}

	return ids
}

func (ps PostList) objectIDs() []uint64 {
	ids := []uint64{}

	for _, p := range ps {
		ids = append(ids, p.ObjectID)
	}

	return ids
}

func postsFromObjects(os object.List) PostList {
	ps := PostList{}

	for _, o := range os {
		ps = append(ps, &Post{Object: o})
	}

	return ps
}

// PostCreateFunc associates the given Post with the owner and stores it.
type PostCreateFunc func(
	currentApp *app.App,
	origin Origin,
	post *Post,
) (*Post, error)

// PostCreate associates the given Post with the owner andstores it.
func PostCreate(
	objects object.Service,
) PostCreateFunc {
	return func(
		currentApp *app.App,
		origin Origin,
		post *Post,
	) (*Post, error) {
		post.OwnerID = origin.UserID
		post.Owned = defaultOwned
		post.Type = TypePost

		if err := post.Validate(); err != nil {
			return nil, wrapError(ErrInvalidEntity, "invalid Post: %s", err)
		}

		if err := constrainPostRestrictions(origin, post.Restrictions); err != nil {
			return nil, err
		}

		if err := constrainPostVisibility(origin, post.Visibility); err != nil {
			return nil, err
		}

		if err := post.Object.Validate(); err != nil {
			return nil, wrapError(ErrInvalidEntity, "%s", err)
		}

		o, err := objects.Put(currentApp.Namespace(), post.Object)
		if err != nil {
			return nil, err
		}

		return &Post{Object: o}, nil
	}
}

// PostDeleteFunc marks a Post as deleted and updates it in the service.
type PostDeleteFunc func(
	currentApp *app.App,
	origin uint64,
	id uint64,
) error

// PostDelete marks a Post as deleted and updates it in the service.
func PostDelete(
	objects object.Service,
) PostDeleteFunc {
	return func(
		currentApp *app.App,
		origin uint64,
		id uint64,
	) error {
		os, err := objects.Query(currentApp.Namespace(), object.QueryOptions{
			ID:    &id,
			Owned: &defaultOwned,
			Types: []string{
				TypePost,
			},
		})
		if err != nil {
			return err
		}

		// A delete should be idempotent and always succeed.
		if len(os) == 0 {
			return nil
		}

		post := os[0]

		if post.OwnerID != origin {
			return wrapError(ErrUnauthorized, "not allowed to delete post")
		}

		post.Deleted = true

		_, err = objects.Put(currentApp.Namespace(), post)
		if err != nil {
			return err
		}

		return nil
	}
}

// PostFetchFunc returns the Post for the given id.
type PostFetchFunc func(currentApp *app.App, id uint64) (*Post, error)

// PostFetch returns the Post for the given id.
func PostFetch(objects object.Service) PostFetchFunc {
	return func(currentApp *app.App, id uint64) (*Post, error) {
		os, err := objects.Query(currentApp.Namespace(), object.QueryOptions{
			ID:    &id,
			Owned: &defaultOwned,
			Types: []string{
				TypePost,
			},
		})
		if err != nil {
			return nil, err
		}

		if len(os) != 1 {
			return nil, ErrNotFound
		}

		return &Post{Object: os[0]}, nil
	}
}

// PostListAllFunc returns all objects which are of type post.
type PostListAllFunc func(
	currentApp *app.App,
	origin uint64,
	opts object.QueryOptions,
) (*PostFeed, error)

// PostListAll returns all objects which are of type post.
func PostListAll(
	connections connection.Service,
	events event.Service,
	objects object.Service,
	reactions reaction.Service,
	users user.Service,
) PostListAllFunc {
	return func(
		currentApp *app.App,
		origin uint64,
		opts object.QueryOptions,
	) (*PostFeed, error) {
		opts.Owned = &defaultOwned
		opts.Types = []string{TypePost}
		opts.Visibilities = []object.Visibility{
			object.VisibilityPublic,
			object.VisibilityGlobal,
		}

		os, err := objects.Query(currentApp.Namespace(), opts)
		if err != nil {
			return nil, err
		}

		ps := postsFromObjects(os)

		err = enrichCounts(events, objects, reactions, currentApp, ps)
		if err != nil {
			return nil, err
		}

		err = enrichHasReacted(reactions, currentApp, origin, ps)
		if err != nil {
			return nil, err
		}

		err = enrichIsLiked(events, currentApp, origin, ps)
		if err != nil {
			return nil, err
		}

		um, err := user.MapFromIDs(users, currentApp.Namespace(), ps.OwnerIDs()...)
		if err != nil {
			return nil, err
		}

		for _, u := range um {
			err = enrichRelation(connections, currentApp, origin, u)
			if err != nil {
				return nil, err
			}
		}

		return &PostFeed{
			Posts:   ps,
			UserMap: um,
		}, nil
	}
}

// PostListUserFunc returns all posts for the given user id as visible by the
// connection user id.
type PostListUserFunc func(
	currentApp *app.App,
	origin uint64,
	userID uint64,
	opts object.QueryOptions,
) (*PostFeed, error)

// PostListUser returns all posts for the given user id as visible by the
// connection user id.
func PostListUser(
	connections connection.Service,
	events event.Service,
	objects object.Service,
	reactions reaction.Service,
	users user.Service,
) PostListUserFunc {
	return func(
		currentApp *app.App,
		origin uint64,
		userID uint64,
		opts object.QueryOptions,
	) (*PostFeed, error) {
		vs := []object.Visibility{
			object.VisibilityPublic,
			object.VisibilityGlobal,
		}

		// Check relation and include connection visibility.
		if origin != userID {
			r, err := queryRelation(connections, currentApp, origin, userID)
			if err != nil {
				return nil, err
			}

			if r.isFriend || r.isFollowing {
				vs = append(vs, object.VisibilityConnection)
			}
		}

		// We want all visibilities if the connection and target are the same.
		if origin == userID {
			vs = append(vs, object.VisibilityConnection, object.VisibilityPrivate)
		}

		opts.OwnerIDs = []uint64{userID}
		opts.Owned = &defaultOwned
		opts.Types = []string{TypePost}
		opts.Visibilities = vs

		os, err := objects.Query(currentApp.Namespace(), opts)
		if err != nil {
			return nil, err
		}

		ps := postsFromObjects(os)

		err = enrichCounts(events, objects, reactions, currentApp, ps)
		if err != nil {
			return nil, err
		}

		err = enrichHasReacted(reactions, currentApp, origin, ps)
		if err != nil {
			return nil, err
		}

		um, err := user.MapFromIDs(users, currentApp.Namespace(), ps.OwnerIDs()...)
		if err != nil {
			return nil, err
		}

		for _, u := range um {
			err = enrichRelation(connections, currentApp, origin, u)
			if err != nil {
				return nil, err
			}
		}

		return &PostFeed{
			Posts:   ps,
			UserMap: um,
		}, nil
	}
}

// PostRetrieveFunc returns the Post for the given id.
type PostRetrieveFunc func(
	currentApp *app.App,
	origin uint64,
	id uint64,
) (*Post, error)

// PostRetrieve returns the Post for the given id.
func PostRetrieve(
	connections connection.Service,
	events event.Service,
	objects object.Service,
	reactions reaction.Service,
) PostRetrieveFunc {
	return func(
		currentApp *app.App,
		origin uint64,
		id uint64,
	) (*Post, error) {
		os, err := objects.Query(currentApp.Namespace(), object.QueryOptions{
			ID:    &id,
			Owned: &defaultOwned,
			Types: []string{
				TypePost,
			},
		})
		if err != nil {
			return nil, err
		}

		if len(os) != 1 {
			return nil, ErrNotFound
		}

		if err := isPostVisible(connections, currentApp, os[0], origin); err != nil {
			return nil, err
		}

		post := &Post{Object: os[0]}

		err = enrichCounts(events, objects, reactions, currentApp, PostList{post})
		if err != nil {
			return nil, err
		}

		err = enrichHasReacted(reactions, currentApp, origin, PostList{post})
		if err != nil {
			return nil, err
		}

		err = enrichIsLiked(events, currentApp, origin, PostList{post})
		if err != nil {
			return nil, err
		}

		return post, nil
	}
}

// PostUpdateFunc stores the post with the new values.
type PostUpdateFunc func(
	currentApp *app.App,
	origin Origin,
	id uint64,
	post *Post,
) (*Post, error)

// PostUpdate stores the post with the new values.
func PostUpdate(
	objects object.Service,
) PostUpdateFunc {
	return func(
		currentApp *app.App,
		origin Origin,
		id uint64,
		post *Post,
	) (*Post, error) {
		if err := constrainPostRestrictions(origin, post.Restrictions); err != nil {
			return nil, err
		}

		ps, err := objects.Query(currentApp.Namespace(), object.QueryOptions{
			ID: &id,
			OwnerIDs: []uint64{
				origin.UserID,
			},
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

		// Preserve information.
		p := ps[0]
		p.Attachments = post.Attachments
		p.Tags = post.Tags
		p.Visibility = post.Visibility

		if post.Restrictions != nil {
			p.Restrictions = post.Restrictions
		}

		err = constrainPostVisibility(origin, p.Visibility)
		if err != nil {
			return nil, err
		}

		if err := p.Validate(); err != nil {
			return nil, wrapError(ErrInvalidEntity, "%s", err)
		}

		o, err := objects.Put(currentApp.Namespace(), p)
		if err != nil {
			return nil, err
		}

		return &Post{Object: o}, nil
	}
}

// IsPost indicates if object is a Post.
func IsPost(o *object.Object) bool {
	if o.Type != TypePost {
		return false
	}

	return o.Owned
}

func constrainPostRestrictions(origin Origin, restrictions *object.Restrictions) error {
	if !origin.IsBackend() && restrictions != nil {
		return wrapError(
			ErrUnauthorized,
			"restrictions can only be set via backend integration",
		)
	}

	return nil
}

func constrainPostVisibility(origin Origin, visibility object.Visibility) error {
	if !origin.IsBackend() && visibility == object.VisibilityGlobal {
		return wrapError(
			ErrUnauthorized,
			"global visibility can only set by backend integration",
		)
	}

	return nil
}

func enrichCounts(
	events event.Service,
	objects object.Service,
	reactions reaction.Service,
	currentApp *app.App,
	ps PostList,
) error {
	commentsMap, err := objects.CountMulti(currentApp.Namespace(), ps.IDs()...)
	if err != nil {
		return err
	}

	reactionsMap, err := reactions.CountMulti(currentApp.Namespace(), reaction.QueryOptions{
		Deleted:   &defaultDeleted,
		ObjectIDs: ps.IDs(),
	})
	if err != nil {
		return err
	}

	for _, p := range ps {
		p.Counts = PostCounts{
			Comments:       commentsMap[p.ID].Comments,
			ReactionCounts: reactionsMap[p.ID],
		}
	}

	return nil
}

func enrichHasReacted(
	reactions reaction.Service,
	currentApp *app.App,
	origin uint64,
	ps PostList,
) error {
	for _, p := range ps {
		rs, err := reactions.Query(currentApp.Namespace(), reaction.QueryOptions{
			Deleted: &defaultDeleted,
			ObjectIDs: []uint64{
				p.ID,
			},
			OwnerIDs: []uint64{
				origin,
			},
		})
		if err != nil {
			return nil
		}

		hasReacted := HasReacted{}

		for _, r := range rs {
			switch r.Type {
			case reaction.TypeLike:
				hasReacted.Like = true
			case reaction.TypeLove:
				hasReacted.Love = true
			case reaction.TypeHaha:
				hasReacted.Haha = true
			case reaction.TypeWow:
				hasReacted.Wow = true
			case reaction.TypeSad:
				hasReacted.Sad = true
			case reaction.TypeAngry:
				hasReacted.Angry = true
			}
		}

		p.HasReacted = hasReacted
	}

	return nil
}

func enrichIsLiked(
	events event.Service,
	currentApp *app.App,
	userID uint64,
	ps PostList,
) error {
	for _, p := range ps {
		es, err := events.Query(currentApp.Namespace(), event.QueryOptions{
			Enabled: &defaultEnabled,
			ObjectIDs: []uint64{
				p.ID,
			},
			Types: []string{
				TypeLike,
			},
			UserIDs: []uint64{
				userID,
			},
		})
		if err != nil {
			return err
		}

		if len(es) == 1 {
			p.IsLiked = true
		}
	}

	return nil
}

// isPostVisible given a post validates that the origin is allowed to see the
// post.
func isPostVisible(
	connections connection.Service,
	currentApp *app.App,
	post *object.Object,
	origin uint64,
) error {
	if origin == post.OwnerID {
		return nil
	}

	switch post.Visibility {
	case object.VisibilityGlobal, object.VisibilityPublic:
		return nil
	case object.VisibilityPrivate:
		return ErrNotFound
	}

	r, err := queryRelation(connections, currentApp, origin, post.OwnerID)
	if err != nil {
		return err
	}

	if !r.isFriend && !r.isFollowing {
		return ErrNotFound
	}

	return nil
}
