package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/net/context"

	"github.com/tapglue/snaas/core"
	"github.com/tapglue/snaas/service/object"
	"github.com/tapglue/snaas/service/user"
)

// CommentCreate creates a new Comment on behalf of the current user.
func CommentCreate(fn core.CommentCreateFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			currentApp  = appFromContext(ctx)
			currentUser = userFromContext(ctx)
			deviceID    = deviceIDFromContext(ctx)
			p           = &payloadComment{}
			tokenType   = tokenTypeFromContext(ctx)

			origin = createOrigin(deviceID, tokenType, currentUser.ID)
		)

		postID, err := extractPostID(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		err = json.NewDecoder(r.Body).Decode(p)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		comment, err := fn(currentApp, origin, postID, p.comment)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		respondJSON(w, http.StatusCreated, &payloadComment{comment: comment})
	}
}

// CommentDelete flags the comment as deleted.
func CommentDelete(fn core.CommentDeleteFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			app         = appFromContext(ctx)
			currentUser = userFromContext(ctx)
		)

		postID, err := extractPostID(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		commentID, err := extractCommentID(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		err = fn(app, currentUser.ID, postID, commentID)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		respondJSON(w, http.StatusNoContent, nil)
	}
}

// CommentList returns all comments for the given a Post.
func CommentList(fn core.CommentListFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			app         = appFromContext(ctx)
			currentUser = userFromContext(ctx)
		)

		postID, err := extractPostID(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		opts, err := extractCommentOpts(r)
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

		feed, err := fn(app, currentUser.ID, postID, opts)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		if len(feed.Comments) == 0 {
			respondJSON(w, http.StatusNoContent, nil)
		}

		respondJSON(w, http.StatusOK, &payloadComments{
			comments: feed.Comments,
			pagination: pagination(
				r,
				opts.Limit,
				commentCursorAfter(feed.Comments, opts.Limit),
				commentCursorBefore(feed.Comments, opts.Limit),
			),
			userMap: feed.UserMap,
		})
	}
}

// CommentRetrieve return the comment for the requested id.
func CommentRetrieve(fn core.CommentRetrieveFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			app         = appFromContext(ctx)
			currentUser = userFromContext(ctx)
		)

		postID, err := extractPostID(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		commentID, err := extractCommentID(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		comment, err := fn(app, currentUser.ID, postID, commentID)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		respondJSON(w, http.StatusOK, &payloadComment{comment: comment})
	}
}

// CommentUpdate replaces the value for a comment with the new values.
func CommentUpdate(fn core.CommentUpdateFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			currentApp  = appFromContext(ctx)
			currentUser = userFromContext(ctx)
			deviceID    = deviceIDFromContext(ctx)
			p           = &payloadComment{}
			tokenType   = tokenTypeFromContext(ctx)

			origin = createOrigin(deviceID, tokenType, currentUser.ID)
		)

		postID, err := extractPostID(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		commentID, err := extractCommentID(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		err = json.NewDecoder(r.Body).Decode(p)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		comment, err := fn(
			currentApp,
			origin,
			postID,
			commentID,
			p.comment,
		)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		respondJSON(w, http.StatusOK, &payloadComment{comment: comment})
	}
}

type payloadComment struct {
	contents object.Contents
	comment  *object.Object
}

func (p *payloadComment) MarshalJSON() ([]byte, error) {
	c := p.comment

	return json.Marshal(struct {
		Content   string          `json:"content"`
		Contents  object.Contents `json:"contents"`
		ID        string          `json:"id"`
		PostID    string          `json:"post_id"`
		Private   *object.Private `json:"private,omitempty"`
		UserID    string          `json:"user_id"`
		CreatedAt time.Time       `json:"created_at"`
		UpdatedAt time.Time       `json:"updated_at"`
	}{
		Content:   c.Attachments[0].Contents[object.DefaultLanguage],
		Contents:  c.Attachments[0].Contents,
		ID:        strconv.FormatUint(c.ID, 10),
		PostID:    strconv.FormatUint(c.ObjectID, 10),
		Private:   c.Private,
		UserID:    strconv.FormatUint(c.OwnerID, 10),
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	})
}

func (p *payloadComment) UnmarshalJSON(raw []byte) error {
	f := struct {
		Content  string            `json:"content"`
		Contents map[string]string `json:"contents"`
		Private  *object.Private   `json:"private,omitempty"`
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

	p.comment = &object.Object{
		Attachments: []object.Attachment{
			{
				Contents: f.Contents,
			},
		},
		Private: f.Private,
	}

	return nil
}

type payloadComments struct {
	comments   object.List
	pagination *payloadPagination
	userMap    user.Map
}

func (p *payloadComments) MarshalJSON() ([]byte, error) {
	cs := []*payloadComment{}

	for _, comment := range p.comments {
		cs = append(cs, &payloadComment{comment: comment})
	}

	return json.Marshal(struct {
		Comments      []*payloadComment  `json:"comments"`
		CommentsCount int                `json:"comments_count"`
		Pagination    *payloadPagination `json:"paging"`
		UserMap       *payloadUserMap    `json:"users"`
		UsersCount    int                `json:"users_count"`
	}{
		Comments:      cs,
		CommentsCount: len(cs),
		Pagination:    p.pagination,
		UserMap:       &payloadUserMap{userMap: p.userMap},
		UsersCount:    len(p.userMap),
	})
}

func commentCursorAfter(cs object.List, limit int) string {
	var after string

	if len(cs) > 0 {
		after = toTimeCursor(cs[0].CreatedAt)
	}

	return after
}

func commentCursorBefore(cs object.List, limit int) string {
	var before string

	if len(cs) > 0 {
		before = toTimeCursor(cs[len(cs)-1].CreatedAt)
	}

	return before
}
