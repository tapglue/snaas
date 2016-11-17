package http

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"github.com/tapglue/snaas/service/app"
	"github.com/tapglue/snaas/service/connection"
	"github.com/tapglue/snaas/service/event"
	"github.com/tapglue/snaas/service/object"
	"github.com/tapglue/snaas/service/user"
)

const (
	cursorTimeFormat = time.RFC3339Nano

	keyCommentID    = "commentID"
	keyCursorAfter  = "after"
	keyCursorBefore = "before"
	keyLimit        = "limit"
	keyPostID       = "postID"
	keyState        = "state"
	keyUserID       = "userID"
	keyWhere        = "where"

	limitDefault = 25
	limitMax     = 50

	refFmt = "%s://%s%s?limit=%d&%s"
)

var cursorEncoding = base64.URLEncoding.WithPadding(base64.NoPadding)

type payloadCursors struct {
	After  string `json:"after"`
	Before string `json:"before"`
}

type payloadPagination struct {
	after  string
	before string
	limit  int
	req    *http.Request
}

func pagination(
	req *http.Request,
	limit int,
	after, before string,
) *payloadPagination {
	return &payloadPagination{
		after:  after,
		before: before,
		limit:  limit,
		req:    req,
	}
}

func (p *payloadPagination) MarshalJSON() ([]byte, error) {
	var (
		next     = ""
		previous = ""
		scheme   = "http"
	)

	if p.req.TLS != nil {
		scheme = "https"
	}

	if p.after != "" {
		next = fmt.Sprintf(
			refFmt,
			scheme,
			p.req.Host,
			p.req.URL.Path,
			p.limit,
			fmt.Sprintf("%s=%s", keyCursorAfter, p.after),
		)
	}

	if p.before != "" {
		previous = fmt.Sprintf(
			refFmt,
			scheme,
			p.req.Host,
			p.req.URL.Path,
			p.limit,
			fmt.Sprintf("%s=%s", keyCursorBefore, p.before),
		)
	}

	f := struct {
		Cursors  payloadCursors `json:"cursors"`
		Next     string         `json:"next"`
		Previous string         `json:"previous"`
	}{
		Cursors: payloadCursors{
			After:  p.after,
			Before: p.before,
		},
		Next:     next,
		Previous: previous,
	}

	return json.Marshal(&f)
}

func extractAppOpts(r *http.Request) (app.QueryOptions, error) {
	return app.QueryOptions{}, nil
}

func extractCommentID(r *http.Request) (uint64, error) {
	return strconv.ParseUint(mux.Vars(r)[keyCommentID], 10, 64)
}

func extractCommentOpts(r *http.Request) (object.QueryOptions, error) {
	return object.QueryOptions{}, nil
}

func extractConnectionOpts(r *http.Request) (connection.QueryOptions, error) {
	return connection.QueryOptions{}, nil
}

type condition struct {
	EQ string   `json:"eq"`
	IN []string `json:"in"`
}

type eventCondition struct {
	Object *struct {
		ID   *condition `json:"id,omitempty"`
		Type *condition `json:"type,omitempty"`
	} `json:"object,omitempty"`
	Type *condition `json:"type,omitempty"`
}

func extractEventOpts(r *http.Request) (event.QueryOptions, error) {
	var (
		cond  = eventCondition{}
		opts  = event.QueryOptions{}
		param = r.URL.Query().Get(keyWhere)
	)

	if param == "" {
		return opts, nil
	}

	err := json.Unmarshal([]byte(param), &cond)
	if err != nil {
		return opts, err
	}

	if cond.Object != nil && cond.Object.ID != nil {
		if cond.Object.ID.EQ != "" {
			opts.ExternalObjectIDs = []string{
				cond.Object.ID.EQ,
			}
		}

		if cond.Object.ID.IN != nil {
			opts.ExternalObjectIDs = cond.Object.ID.IN
		}
	}

	if cond.Object != nil && cond.Object.Type != nil {
		if cond.Object.Type.EQ != "" {
			opts.ExternalObjectTypes = []string{
				cond.Object.Type.EQ,
			}
		}

		if cond.Object.Type.IN != nil {
			opts.ExternalObjectTypes = cond.Object.Type.IN
		}
	}

	if cond.Type != nil {
		if cond.Type.EQ != "" {
			opts.Types = []string{
				cond.Type.EQ,
			}
		}

		if cond.Type.IN != nil {
			opts.Types = cond.Type.IN
		}
	}

	return opts, nil
}

func extractIDCursorBefore(r *http.Request) (uint64, error) {
	var (
		id    uint64 = 0
		param        = r.URL.Query().Get(keyCursorBefore)
	)

	if param == "" {
		return id, nil
	}

	cursor, err := cursorEncoding.DecodeString(param)
	if err != nil {
		return id, err
	}

	return strconv.ParseUint(string(cursor), 10, 64)
}

func extractLikeOpts(r *http.Request) (event.QueryOptions, error) {
	return event.QueryOptions{}, nil
}

func extractLimit(r *http.Request) (int, error) {
	param := r.URL.Query().Get(keyLimit)

	if param == "" {
		return limitDefault, nil
	}

	limit, err := strconv.Atoi(param)
	if err != nil {
		return 0, err
	}

	if limit > limitMax {
		return limitMax, nil
	}

	return limit, nil
}

func extractPostID(r *http.Request) (uint64, error) {
	return strconv.ParseUint(mux.Vars(r)[keyPostID], 10, 64)
}

func extractPostOpts(r *http.Request) (object.QueryOptions, error) {
	var (
		opts  = object.QueryOptions{}
		param = r.URL.Query().Get(keyWhere)
		w     = struct {
			Post *postWhere `json:"post"`
		}{}
	)

	if param == "" {
		return opts, nil
	}

	err := json.Unmarshal([]byte(param), &w)
	if err != nil {
		return opts, fmt.Errorf("error in where param: %s", err)
	}

	if w.Post != nil && w.Post.Tags != nil {
		opts.Tags = w.Post.Tags
	}

	return opts, nil
}

func extractState(r *http.Request) connection.State {
	return connection.State(mux.Vars(r)[keyState])
}

func extractTimeCursorBefore(r *http.Request) (time.Time, error) {
	var (
		before = time.Now()
		param  = r.URL.Query().Get(keyCursorBefore)
	)

	if param == "" {
		return before, nil
	}

	cursor, err := cursorEncoding.DecodeString(param)
	if err != nil {
		return before, err
	}

	return time.Parse(cursorTimeFormat, string(cursor))
}

func extractUserID(r *http.Request) (uint64, error) {
	return strconv.ParseUint(mux.Vars(r)[keyUserID], 10, 64)
}

func extractUserOpts(r *http.Request) (user.QueryOptions, error) {
	return user.QueryOptions{}, nil
}

func toIDCursor(id uint64) string {
	return cursorEncoding.EncodeToString([]byte(strconv.FormatUint(id, 10)))
}

func toTimeCursor(t time.Time) string {
	return cursorEncoding.EncodeToString([]byte(t.Format(cursorTimeFormat)))
}
