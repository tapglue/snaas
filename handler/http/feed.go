package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/net/context"

	"github.com/tapglue/snaas/core"
	"github.com/tapglue/snaas/service/event"
	"github.com/tapglue/snaas/service/user"
)

// FeedEvents returns the events of the current user driven by the social and
// interest graph.
func FeedEvents(fn core.FeedEventsFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			app         = appFromContext(ctx)
			currentUser = userFromContext(ctx)
		)

		opts, err := extractEventOpts(r)
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

		if len(feed.Events) == 0 {
			respondJSON(w, http.StatusNoContent, nil)
			return
		}

		respondJSON(w, http.StatusOK, &payloadFeedEvents{
			events: feed.Events,
			pagination: pagination(
				r,
				opts.Limit,
				eventCursorAfter(feed.Events, opts.Limit),
				eventCursorBefore(feed.Events, opts.Limit),
			),
			postMap: feed.PostMap,
			userMap: feed.UserMap,
		})
	}
}

// FeedNews returns the superset aggregration of events and posts driven by the
// social and interest graph of the current user.
func FeedNews(fn core.FeedNewsFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			app         = appFromContext(ctx)
			currentUser = userFromContext(ctx)
		)

		eventOpts, err := extractEventOpts(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		postOpts, err := extractPostOpts(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		eventOpts.Before, postOpts.Before, err = extractNewsCursor(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		limit, err := extractLimit(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		eventOpts.Limit, postOpts.Limit = limit, limit

		feed, err := fn(app, currentUser.ID, eventOpts, postOpts)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		if len(feed.Events) == 0 && len(feed.Posts) == 0 {
			respondJSON(w, http.StatusNoContent, nil)
			return
		}

		after, err := newsCursorAfter(feed.Events, feed.Posts, limit)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		before, err := newsCursorBefore(feed.Events, feed.Posts, limit)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		respondJSON(w, http.StatusOK, &payloadFeedNews{
			events:     feed.Events,
			pagination: pagination(r, limit, after, before),
			posts:      feed.Posts,
			postMap:    feed.PostMap,
			userMap:    feed.UserMap,
			lastRead:   currentUser.LastRead,
		})
	}
}

// FeedNotificationsSelf returns the events which target the origin user and
// their content.
func FeedNotificationsSelf(fn core.FeedNotificationsSelfFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			app         = appFromContext(ctx)
			currentUser = userFromContext(ctx)
		)

		opts, err := extractEventOpts(r)
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

		if len(feed.Events) == 0 {
			respondJSON(w, http.StatusNoContent, nil)
			return
		}

		respondJSON(w, http.StatusOK, &payloadFeedEvents{
			events: feed.Events,
			pagination: pagination(
				r,
				opts.Limit,
				eventCursorAfter(feed.Events, opts.Limit),
				eventCursorBefore(feed.Events, opts.Limit),
			),
			postMap: feed.PostMap,
			userMap: feed.UserMap,
		})
	}
}

// FeedPosts returns the posts of the current user driven by the social and
// interest graph.
func FeedPosts(fn core.FeedPostsFunc) Handler {
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

		respondJSON(w, http.StatusOK, &payloadFeedPosts{
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

type payloadFeedEvents struct {
	pagination *payloadPagination
	events     event.List
	postMap    core.PostMap
	userMap    user.Map
}

func (p *payloadFeedEvents) MarshalJSON() ([]byte, error) {
	es := []*payloadEvent{}

	for _, e := range p.events {
		es = append(es, &payloadEvent{event: e})
	}

	pm := map[string]*payloadPost{}

	for id, post := range p.postMap {
		pm[strconv.FormatUint(id, 10)] = &payloadPost{post: post}
	}

	return json.Marshal(struct {
		Events       []*payloadEvent         `json:"events"`
		EventsCount  int                     `json:"events_count"`
		Pagination   *payloadPagination      `json:"paging"`
		PostMap      map[string]*payloadPost `json:"post_map"`
		PostMapCount int                     `json:"post_map_count"`
		Users        *payloadUserMap         `json:"users"`
		UsersCount   int                     `json:"users_count"`
	}{
		Events:       es,
		EventsCount:  len(es),
		Pagination:   p.pagination,
		PostMap:      pm,
		PostMapCount: len(pm),
		Users:        &payloadUserMap{userMap: p.userMap},
		UsersCount:   len(p.userMap),
	})
}

type payloadFeedNews struct {
	events     event.List
	pagination *payloadPagination
	posts      core.PostList
	postMap    core.PostMap
	userMap    user.Map
	lastRead   time.Time
}

func (p *payloadFeedNews) MarshalJSON() ([]byte, error) {
	var (
		es           = []*payloadEvent{}
		unreadEvents = 0
	)

	for _, ev := range p.events {
		es = append(es, &payloadEvent{event: ev})

		if ev.CreatedAt.After(p.lastRead) {
			unreadEvents++
		}
	}

	var (
		ps          = []*payloadPost{}
		unreadPosts = 0
	)

	for _, post := range p.posts {
		ps = append(ps, &payloadPost{post: post})

		if post.CreatedAt.After(p.lastRead) {
			unreadPosts++
		}
	}

	pm := map[string]*payloadPost{}

	for id, post := range p.postMap {
		pm[strconv.FormatUint(id, 10)] = &payloadPost{post: post}
	}

	return json.Marshal(struct {
		Events            []*payloadEvent         `json:"events"`
		EventsCount       int                     `json:"events_count"`
		EventsCountUnread int                     `json:"events_count_unread"`
		Pagination        *payloadPagination      `json:"paging"`
		Posts             []*payloadPost          `json:"posts"`
		PostsCount        int                     `json:"posts_count"`
		PostsCountUnread  int                     `json:"posts_count_unread"`
		PostMap           map[string]*payloadPost `json:"post_map"`
		PostMapCount      int                     `json:"post_map_count"`
		UserMap           *payloadUserMap         `json:"users"`
		UserCount         int                     `json:"users_count"`
	}{
		Events:            es,
		EventsCount:       len(es),
		EventsCountUnread: unreadEvents,
		Pagination:        p.pagination,
		Posts:             ps,
		PostsCount:        len(ps),
		PostsCountUnread:  unreadPosts,
		PostMap:           pm,
		PostMapCount:      len(pm),
		UserMap:           &payloadUserMap{userMap: p.userMap},
		UserCount:         len(p.userMap),
	})
}

type payloadFeedPosts struct {
	pagination *payloadPagination
	posts      core.PostList
	userMap    user.Map
}

func (p *payloadFeedPosts) MarshalJSON() ([]byte, error) {
	ps := []*payloadPost{}

	for _, p := range p.posts {
		ps = append(ps, &payloadPost{post: p})
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

type postWhere struct {
	Tags []string `json:"tags"`
}

type newsCursor struct {
	Events time.Time `json:"events"`
	Posts  time.Time `json:"posts"`
}

func extractNewsCursor(r *http.Request) (time.Time, time.Time, error) {
	var (
		before = time.Now().UTC()
		cursor = &newsCursor{
			Events: before,
			Posts:  before,
		}
		param = r.URL.Query().Get(keyCursorBefore)
	)

	if param == "" {
		return before, before, nil
	}

	raw, err := cursorEncoding.DecodeString(param)
	if err != nil {
		return before, before, err
	}

	err = json.Unmarshal(raw, cursor)
	if err != nil {
		return before, before, err
	}

	return cursor.Events, cursor.Posts, nil
}

func newsCursorAfter(
	es event.List,
	ps core.PostList,
	limit int,
) (string, error) {
	cursor := newsCursor{}

	if len(es) > 0 {
		cursor.Events = es[0].CreatedAt
	}

	if len(ps) > 0 {
		cursor.Posts = ps[0].CreatedAt
	}

	r, err := json.Marshal(&cursor)
	if err != nil {
		return "", err
	}

	return cursorEncoding.EncodeToString(r), nil
}

func newsCursorBefore(
	es event.List,
	ps core.PostList,
	limit int,
) (string, error) {
	cursor := newsCursor{}

	if len(es) > 0 {
		cursor.Events = es[len(es)-1].CreatedAt
	}

	if len(ps) > 0 {
		cursor.Posts = ps[len(ps)-1].CreatedAt
	}

	r, err := json.Marshal(&cursor)
	if err != nil {
		return "", err
	}

	return cursorEncoding.EncodeToString(r), nil
}
