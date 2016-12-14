package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/net/context"

	"github.com/tapglue/snaas/core"
	"github.com/tapglue/snaas/service/reaction"
	"github.com/tapglue/snaas/service/user"
)

// ReactionCreate creates a Reaction on the Post.
func ReactionCreate(fn core.ReactionCreateFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			currentApp  = appFromContext(ctx)
			currentUser = userFromContext(ctx)
		)

		postID, err := extractPostID(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		reactionType, err := extractReactionType(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		reaction, err := fn(currentApp, currentUser.ID, postID, reactionType)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		respondJSON(w, http.StatusCreated, &payloadReaction{reaction: reaction})
	}
}

// ReactionDelete removes an existing Reaction for the currentUser on the Post.
func ReactionDelete(fn core.ReactionDeleteFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			currentApp  = appFromContext(ctx)
			currentUser = userFromContext(ctx)
		)

		postID, err := extractPostID(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		reactionType, err := extractReactionType(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		err = fn(currentApp, currentUser.ID, postID, reactionType)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		respondJSON(w, http.StatusNoContent, nil)
	}
}

// ReactionListPost returns all reactions for a Post.
func ReactionListPost(fn core.ReactionListPostFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			currentApp  = appFromContext(ctx)
			currentUser = userFromContext(ctx)
		)

		postID, err := extractPostID(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		opts, err := extractReactionOpts(r)
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

		feed, err := fn(currentApp, currentUser.ID, postID, opts)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		if len(feed.Reactions) == 0 {
			respondJSON(w, http.StatusNoContent, nil)
			return
		}

		respondJSON(w, http.StatusOK, &payloadReactions{
			pagination: pagination(
				r,
				opts.Limit,
				reactionCursorAfter(feed.Reactions, opts.Limit),
				reactionCursorBefore(feed.Reactions, opts.Limit),
			),
			postMap:   feed.PostMap,
			reactions: feed.Reactions,
			userMap:   feed.UserMap,
		})
	}
}

// ReactionListPostByType returns all reactions for a Post.
func ReactionListPostByType(fn core.ReactionListPostFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			currentApp  = appFromContext(ctx)
			currentUser = userFromContext(ctx)
		)

		postID, err := extractPostID(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		reactionType, err := extractReactionType(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		opts, err := extractReactionOpts(r)
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

		opts.Types = []reaction.Type{
			reactionType,
		}

		feed, err := fn(currentApp, currentUser.ID, postID, opts)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		if len(feed.Reactions) == 0 {
			respondJSON(w, http.StatusNoContent, nil)
			return
		}

		respondJSON(w, http.StatusOK, &payloadReactions{
			pagination: pagination(
				r,
				opts.Limit,
				reactionCursorAfter(feed.Reactions, opts.Limit),
				reactionCursorBefore(feed.Reactions, opts.Limit),
			),
			postMap:   feed.PostMap,
			reactions: feed.Reactions,
			userMap:   feed.UserMap,
		})
	}
}

type payloadReaction struct {
	reaction *reaction.Reaction
}

func (p *payloadReaction) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID        string        `json:"id"`
		PostID    string        `json:"post_id"`
		Type      reaction.Type `json:"type"`
		CreatedAt time.Time     `json:"created_at"`
		UpdatedAt time.Time     `json:"updated_at"`
	}{
		ID:        strconv.FormatUint(p.reaction.ID, 10),
		PostID:    strconv.FormatUint(p.reaction.ObjectID, 10),
		Type:      p.reaction.Type,
		CreatedAt: p.reaction.CreatedAt,
		UpdatedAt: p.reaction.UpdatedAt,
	})
}

type payloadReactions struct {
	reactions  reaction.List
	pagination *payloadPagination
	postMap    core.PostMap
	userMap    user.Map
}

func (p *payloadReactions) MarshalJSON() ([]byte, error) {
	rs := []*payloadReaction{}

	for _, r := range p.reactions {
		rs = append(rs, &payloadReaction{reaction: r})
	}

	pm := map[string]*payloadPost{}

	for id, post := range p.postMap {
		pm[strconv.FormatUint(id, 10)] = &payloadPost{post: post}
	}

	return json.Marshal(struct {
		Pagination     *payloadPagination      `json:"paging"`
		PostMap        map[string]*payloadPost `json:"post_map"`
		PostMapCount   int                     `json:"post_map_count"`
		Reactions      []*payloadReaction      `json:"reactions"`
		ReactionsCount int                     `json:"reactions_count"`
		UserMap        *payloadUserMap         `json:"users"`
		UserCount      int                     `json:"users_count"`
	}{
		Pagination:     p.pagination,
		PostMap:        pm,
		PostMapCount:   len(pm),
		Reactions:      rs,
		ReactionsCount: len(rs),
		UserMap:        &payloadUserMap{userMap: p.userMap},
		UserCount:      len(p.userMap),
	})
}

func reactionCursorAfter(rs reaction.List, limit int) string {
	var after string

	if len(rs) != 0 {
		after = toTimeCursor(rs[0].UpdatedAt)
	}

	return after
}

func reactionCursorBefore(rs reaction.List, limit int) string {
	var before string

	if len(rs) != 0 {
		before = toTimeCursor(rs[len(rs)-1].UpdatedAt)
	}

	return before
}
