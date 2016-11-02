package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/net/context"

	"github.com/tapglue/api/core"
	"github.com/tapglue/api/service/object"
	"github.com/tapglue/api/service/user"
)

// PostCreate creates a new Post.
func PostCreate(fn core.PostCreateFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			currentApp  = appFromContext(ctx)
			deviceID    = deviceIDFromContext(ctx)
			currentUser = userFromContext(ctx)
			p           = &payloadPost{}
			tokenType   = tokenTypeFromContext(ctx)

			origin = createOrigin(deviceID, tokenType, currentUser.ID)
		)

		err := json.NewDecoder(r.Body).Decode(p)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		post, err := fn(currentApp, origin, p.post)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		respondJSON(w, http.StatusCreated, &payloadPost{post: post})
	}
}

// PostDelete flags the Post as deleted.
func PostDelete(fn core.PostDeleteFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			app         = appFromContext(ctx)
			currentUser = userFromContext(ctx)
		)

		id, err := extractPostID(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		err = fn(app, currentUser.ID, id)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		respondJSON(w, http.StatusNoContent, nil)
	}
}

// PostList returns all posts for a user as visible by the current user.
func PostList(fn core.PostListUserFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			app         = appFromContext(ctx)
			currentUser = userFromContext(ctx)
		)

		userID, err := extractUserID(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		opts, err := extractPostOpts(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		opts.Before, err = extractTimeCursorBefore(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		opts.Limit, err = extractLimit(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		feed, err := fn(app, currentUser.ID, userID, opts)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		if len(feed.Posts) == 0 {
			respondJSON(w, http.StatusNoContent, nil)
			return
		}

		respondJSON(w, http.StatusOK, &payloadPosts{
			pagination: pagination(
				r,
				opts.Limit,
				postCursorAfter(feed.Posts, opts.Limit),
				postCursorBefore(feed.Posts, opts.Limit),
			),
			posts:   feed.Posts,
			userMap: feed.UserMap,
		})
	}
}

// PostListAll returns all publicly visible posts.
func PostListAll(fn core.PostListAllFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			app         = appFromContext(ctx)
			currentUser = userFromContext(ctx)
		)

		opts, err := extractPostOpts(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		opts.Before, err = extractTimeCursorBefore(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		opts.Limit, err = extractLimit(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		feed, err := fn(app, currentUser.ID, opts)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		if len(feed.Posts) == 0 {
			respondJSON(w, http.StatusNoContent, nil)
			return
		}

		respondJSON(w, http.StatusOK, &payloadPosts{
			pagination: pagination(
				r,
				opts.Limit,
				postCursorAfter(feed.Posts, opts.Limit),
				postCursorBefore(feed.Posts, opts.Limit),
			),
			posts:   feed.Posts,
			userMap: feed.UserMap,
		})
	}
}

// PostListMe returns all posts of the current user.
func PostListMe(fn core.PostListUserFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			app         = appFromContext(ctx)
			currentUser = userFromContext(ctx)
		)

		opts, err := extractPostOpts(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		opts.Before, err = extractTimeCursorBefore(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		opts.Limit, err = extractLimit(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		feed, err := fn(app, currentUser.ID, currentUser.ID, opts)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		if len(feed.Posts) == 0 {
			respondJSON(w, http.StatusNoContent, nil)
			return
		}

		respondJSON(w, http.StatusOK, &payloadPosts{
			pagination: pagination(
				r,
				opts.Limit,
				postCursorAfter(feed.Posts, opts.Limit),
				postCursorBefore(feed.Posts, opts.Limit),
			),
			posts:   feed.Posts,
			userMap: feed.UserMap,
		})
	}
}

// PostRetrieve returns the requested Post.
func PostRetrieve(fn core.PostRetrieveFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			app         = appFromContext(ctx)
			currentUser = userFromContext(ctx)
		)

		id, err := extractPostID(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		post, err := fn(app, currentUser.ID, id)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		respondJSON(w, http.StatusOK, &payloadPost{post: post})
	}
}

// PostUpdate reaplces a post with new values.
func PostUpdate(fn core.PostUpdateFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			currentApp  = appFromContext(ctx)
			currentUser = userFromContext(ctx)
			deviceID    = deviceIDFromContext(ctx)
			p           = payloadPost{}
			tokenType   = tokenTypeFromContext(ctx)

			origin = createOrigin(deviceID, tokenType, currentUser.ID)
		)

		id, err := extractPostID(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		err = json.NewDecoder(r.Body).Decode(&p)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		updated, err := fn(
			currentApp,
			origin,
			id,
			p.post,
		)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		respondJSON(w, http.StatusOK, &payloadPost{post: updated})
	}
}

type payloadAttachment struct {
	attachment object.Attachment
}

func (p *payloadAttachment) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Content  string          `json:"content"`
		Contents object.Contents `json:"contents"`
		Name     string          `json:"name"`
		Type     string          `json:"type"`
	}{
		Content:  p.attachment.Contents[object.DefaultLanguage],
		Contents: p.attachment.Contents,
		Name:     p.attachment.Name,
		Type:     p.attachment.Type,
	})
}

func (p *payloadAttachment) UnmarshalJSON(raw []byte) error {
	f := struct {
		Content  string          `json:"content"`
		Contents object.Contents `json:"contents"`
		Name     string          `json:"name"`
		Type     string          `json:"type"`
	}{}

	err := json.Unmarshal(raw, &f)
	if err != nil {
		return err
	}

	if f.Contents == nil {
		if f.Content == "" {
			return ErrBadRequest
		}

		f.Contents = object.Contents{
			object.DefaultLanguage: f.Content,
		}
	}

	p.attachment = object.Attachment{
		Contents: f.Contents,
		Name:     f.Name,
		Type:     f.Type,
	}

	return nil
}

type payloadPost struct {
	post *core.Post
}

func (p *payloadPost) MarshalJSON() ([]byte, error) {
	ps := []*payloadAttachment{}

	for _, a := range p.post.Attachments {
		ps = append(ps, &payloadAttachment{attachment: a})
	}

	return json.Marshal(struct {
		Attachments  []*payloadAttachment `json:"attachments"`
		Counts       postCounts           `json:"counts"`
		CreatedAt    time.Time            `json:"created_at,omitempty"`
		ID           string               `json:"id"`
		IsLiked      bool                 `json:"is_liked"`
		Restrictions *object.Restrictions `json:"restrictions,omitempty"`
		Tags         []string             `json:"tags,omitempty"`
		UpdatedAt    time.Time            `json:"updated_at,omitempty"`
		UserID       string               `json:"user_id"`
		Visibility   object.Visibility    `json:"visibility"`
	}{
		Attachments: ps,
		Counts: postCounts{
			Comments: p.post.Counts.Comments,
			Likes:    p.post.Counts.Likes,
		},
		CreatedAt:    p.post.CreatedAt,
		ID:           strconv.FormatUint(p.post.ID, 10),
		IsLiked:      p.post.IsLiked,
		Restrictions: p.post.Restrictions,
		Tags:         p.post.Tags,
		UpdatedAt:    p.post.UpdatedAt,
		UserID:       strconv.FormatUint(p.post.OwnerID, 10),
		Visibility:   p.post.Visibility,
	})
}

func (p *payloadPost) UnmarshalJSON(raw []byte) error {
	f := struct {
		Attachments  []*payloadAttachment `json:"attachments"`
		Restrictions *object.Restrictions `json:"restrictions,omitempty"`
		Tags         []string             `json:"tags,omitempty"`
		Visibility   object.Visibility    `json:"visibility"`
	}{}

	err := json.Unmarshal(raw, &f)
	if err != nil {
		return err
	}

	as := []object.Attachment{}

	for _, a := range f.Attachments {
		as = append(as, a.attachment)
	}

	p.post = &core.Post{Object: &object.Object{}}
	p.post.Attachments = as
	p.post.Restrictions = f.Restrictions
	p.post.Tags = f.Tags
	p.post.Visibility = f.Visibility

	return nil
}

type payloadPosts struct {
	pagination *payloadPagination
	posts      core.PostList
	userMap    user.Map
}

func (p *payloadPosts) MarshalJSON() ([]byte, error) {
	ps := []*payloadPost{}

	for _, post := range p.posts {
		ps = append(ps, &payloadPost{post: post})
	}

	return json.Marshal(struct {
		Pagination *payloadPagination `json:"paging"`
		Posts      []*payloadPost     `json:"posts"`
		PostsCount int                `json:"posts_count"`
		UserMap    *payloadUserMap    `json:"users"`
		UserCount  int                `json:"users_count"`
	}{
		Pagination: p.pagination,
		Posts:      ps,
		PostsCount: len(ps),
		UserMap:    &payloadUserMap{userMap: p.userMap},
		UserCount:  len(p.userMap),
	})
}

type postCounts struct {
	Comments int `json:"comments"`
	Likes    int `json:"likes"`
}

type postFields struct {
	Attachments []object.Attachment `json:"attachments"`
	Counts      postCounts          `json:"counts"`
	CreatedAt   time.Time           `json:"created_at,omitempty"`
	ID          string              `json:"id"`
	IsLiked     bool                `json:"is_liked"`
	Tags        []string            `json:"tags,omitempty"`
	UpdatedAt   time.Time           `json:"updated_at,omitempty"`
	UserID      string              `json:"user_id"`
	Visibility  object.Visibility   `json:"visibility"`
}

func postCursorAfter(ps core.PostList, limit int) string {
	var after string

	if len(ps) > 0 {
		after = toTimeCursor(ps[0].CreatedAt)
	}

	return after
}

func postCursorBefore(ps core.PostList, limit int) string {
	var before string

	if len(ps) > 0 {
		before = toTimeCursor(ps[len(ps)-1].CreatedAt)
	}

	return before
}
