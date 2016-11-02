package http

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"github.com/tapglue/api/service/app"
	"github.com/tapglue/api/service/connection"
	"github.com/tapglue/api/service/event"
	"github.com/tapglue/api/service/object"
	"github.com/tapglue/api/service/user"
	v04_core "github.com/tapglue/multiverse/v04/core"
)

const (
	cursorTimeFormat = time.RFC3339Nano
	defaultLimit     = 100
	keyCommentID     = "commentID"
	keyCursorAfter   = "after"
	keyCursorBefore  = "before"
	keyLimit         = "limit"
	keyPostID        = "postID"
	keyState         = "state"
	keyUserID        = "userID"
	keyWhere         = "where"
	maxLimit         = 100

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

func extractEventOpts(r *http.Request) (event.QueryOptions, error) {
	var (
		opts  = event.QueryOptions{}
		param = r.URL.Query().Get(keyWhere)
	)

	if param == "" {
		return opts, nil
	}

	cond, errs := v04_core.NewEventFilter(param)
	if errs != nil {
		return opts, errs[0]
	}

	if cond == nil {
		return opts, nil
	}

	if cond.Object != nil && cond.Object.ID != nil {
		if cond.Object.ID.Eq != nil {
			id, err := parseID(cond.Object.ID.Eq)
			if err != nil {
				return opts, err
			}

			opts.ExternalObjectIDs = []string{
				id,
			}
		}

		if cond.Object.ID.In != nil {
			opts.ExternalObjectIDs = []string{}

			for _, input := range cond.Object.ID.In {
				id, err := parseID(input)
				if err != nil {
					return opts, err
				}

				opts.ExternalObjectIDs = append(opts.ExternalObjectIDs, id)
			}
		}
	}

	if cond.Object != nil && cond.Object.Type != nil {
		if cond.Object.Type.Eq != nil {
			t, ok := cond.Object.Type.Eq.(string)
			if !ok {
				return opts, fmt.Errorf("error in where param")
			}

			opts.ExternalObjectTypes = []string{
				t,
			}
		}

		if cond.Object.Type.In != nil {
			opts.ExternalObjectTypes = []string{}

			for _, input := range cond.Object.Type.In {
				t, ok := input.(string)
				if !ok {
					return opts, fmt.Errorf("error in where param")
				}

				opts.ExternalObjectTypes = append(opts.ExternalObjectTypes, t)
			}
		}
	}

	if cond.Type != nil {
		if cond.Type.Eq != nil {
			t, ok := cond.Type.Eq.(string)
			if !ok {
				return opts, fmt.Errorf("error in where param")
			}

			opts.Types = []string{
				t,
			}
		}

		if cond.Type.In != nil {
			opts.Types = []string{}

			for _, input := range cond.Type.In {
				t, ok := input.(string)
				if !ok {
					return opts, fmt.Errorf("error in where param")
				}

				opts.Types = append(opts.Types, t)
			}
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
		return defaultLimit, nil
	}

	limit, err := strconv.Atoi(param)
	if err != nil {
		return 0, err
	}

	if limit > maxLimit {
		return maxLimit, nil
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
