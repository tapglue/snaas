package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/net/context"

	"github.com/tapglue/snaas/core"
	"github.com/tapglue/snaas/service/connection"
	"github.com/tapglue/snaas/service/user"
)

// ConnectionByState returns all connections for a user for a certain state.
func ConnectionByState(fn core.ConnectionByStateFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			app         = appFromContext(ctx)
			currentUser = userFromContext(ctx)
			state       = extractState(r)
		)

		feed, err := fn(app, currentUser.ID, state)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		if len(feed.Connections) == 0 {
			respondJSON(w, http.StatusNoContent, nil)
			return
		}

		respondJSON(w, http.StatusOK, &payloadConnections{
			cons:    feed.Connections,
			origin:  currentUser.ID,
			userMap: feed.UserMap,
		})
	}
}

// ConnectionDelete flags the given connection as disabled.
func ConnectionDelete(fn core.ConnectionDeleteFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			app         = appFromContext(ctx)
			currentUser = userFromContext(ctx)
		)

		toID, err := strconv.ParseUint(mux.Vars(r)["toID"], 10, 64)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		con := &connection.Connection{
			FromID: currentUser.ID,
			ToID:   toID,
			State:  connection.StatePending,
			Type:   connection.Type(mux.Vars(r)["type"]),
		}

		if err := con.Validate(); err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		err = fn(app, con)
		if err != nil {
			if core.IsInvalidEntity(err) {
				respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			} else {
				respondError(w, 0, err)
			}

			return
		}

		respondJSON(w, http.StatusNoContent, nil)
	}
}

// ConnectionFollowers returns the list of users who follow the user with the id.
func ConnectionFollowers(fn core.ConnectionFollowersFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			app         = appFromContext(ctx)
			currentUser = userFromContext(ctx)
		)

		userID, err := extractUserID(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, "invalid user id"))
			return
		}

		opts, err := extractConnectionOpts(r)
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

		if len(feed.Users) == 0 {
			respondJSON(w, http.StatusNoContent, nil)
			return
		}

		respondJSON(w, http.StatusOK, &payloadUsers{
			pagination: pagination(
				r,
				opts.Limit,
				connectionCursorAfter(feed.Connections, opts.Limit),
				connectionCursorBefore(feed.Connections, opts.Limit),
			),
			users: feed.Users,
		})
	}
}

// ConnectionFollowersMe returns the list of users who follow the user with the id.
func ConnectionFollowersMe(fn core.ConnectionFollowersFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			app         = appFromContext(ctx)
			currentUser = userFromContext(ctx)
		)

		opts, err := extractConnectionOpts(r)
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

		if len(feed.Users) == 0 {
			respondJSON(w, http.StatusNoContent, nil)
			return
		}

		respondJSON(w, http.StatusOK, &payloadUsers{
			pagination: pagination(
				r,
				opts.Limit,
				connectionCursorAfter(feed.Connections, opts.Limit),
				connectionCursorBefore(feed.Connections, opts.Limit),
			),
			users: feed.Users,
		})
	}
}

// ConnectionFollowings returns the list of users the current user is following.
func ConnectionFollowings(fn core.ConnectionFollowingsFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			app         = appFromContext(ctx)
			currentUser = userFromContext(ctx)
		)

		userID, err := extractUserID(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, "invalid user id"))
			return
		}

		opts, err := extractConnectionOpts(r)
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

		if len(feed.Users) == 0 {
			respondJSON(w, http.StatusNoContent, nil)
			return
		}

		respondJSON(w, http.StatusOK, &payloadUsers{
			pagination: pagination(
				r,
				opts.Limit,
				connectionCursorAfter(feed.Connections, opts.Limit),
				connectionCursorBefore(feed.Connections, opts.Limit),
			),
			users: feed.Users,
		})
	}
}

// ConnectionFollowingsMe returns the list of users the current user is following.
func ConnectionFollowingsMe(fn core.ConnectionFollowingsFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			app         = appFromContext(ctx)
			currentUser = userFromContext(ctx)
		)

		opts, err := extractConnectionOpts(r)
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

		if len(feed.Users) == 0 {
			respondJSON(w, http.StatusNoContent, nil)
			return
		}

		respondJSON(w, http.StatusOK, &payloadUsers{
			pagination: pagination(
				r,
				opts.Limit,
				connectionCursorAfter(feed.Connections, opts.Limit),
				connectionCursorBefore(feed.Connections, opts.Limit),
			),
			users: feed.Users,
		})
	}
}

// ConnectionFriends returns the list of users the current user is friends with.
func ConnectionFriends(fn core.ConnectionFriendsFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			app         = appFromContext(ctx)
			currentUser = userFromContext(ctx)
		)

		userID, err := extractUserID(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, "invalid user id"))
			return
		}

		opts, err := extractConnectionOpts(r)
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

		if len(feed.Users) == 0 {
			respondJSON(w, http.StatusNoContent, nil)
			return
		}

		respondJSON(w, http.StatusOK, &payloadUsers{
			pagination: pagination(
				r,
				opts.Limit,
				connectionCursorAfter(feed.Connections, opts.Limit),
				connectionCursorBefore(feed.Connections, opts.Limit),
			),
			users: feed.Users,
		})
	}
}

// ConnectionFriendsMe returns the list of users the current user is friends with.
func ConnectionFriendsMe(fn core.ConnectionFriendsFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			app         = appFromContext(ctx)
			currentUser = userFromContext(ctx)
		)

		opts, err := extractConnectionOpts(r)
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

		if len(feed.Users) == 0 {
			respondJSON(w, http.StatusNoContent, nil)
			return
		}

		respondJSON(w, http.StatusOK, &payloadUsers{
			pagination: pagination(
				r,
				opts.Limit,
				connectionCursorAfter(feed.Connections, opts.Limit),
				connectionCursorBefore(feed.Connections, opts.Limit),
			),
			users: feed.Users,
		})
	}
}

// ConnectionSocial takes a list of connection ids and creates connections for
// the given user.
func ConnectionSocial(fn core.ConnectionCreateSocialFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			app         = appFromContext(ctx)
			currentUser = userFromContext(ctx)
			p           = payloadSocial{}
		)

		err := json.NewDecoder(r.Body).Decode(&p)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		if len(p.ConnectionIDs) == 0 {
			respondJSON(w, http.StatusNoContent, nil)
			return
		}

		opts, err := extractUserOpts(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		opts.Before, err = extractIDCursorBefore(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		opts.Limit, err = extractLimit(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		opts.SocialIDs = map[string][]string{
			p.Platform: p.ConnectionIDs,
		}

		us, err := fn(
			app,
			currentUser.ID,
			p.Type,
			p.State,
			opts,
		)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		if len(us) == 0 {
			respondJSON(w, http.StatusNoContent, nil)
			return
		}

		respondJSON(w, http.StatusOK, &payloadUsers{
			pagination: pagination(
				r,
				opts.Limit,
				userCursorAfter(us, opts.Limit),
				userCursorBefore(us, opts.Limit),
			),
			users: us,
		})
	}
}

// ConnectionUpdate stores a new connection or updates the state of an exisitng
// Connection.
func ConnectionUpdate(fn core.ConnectionUpdateFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			app         = appFromContext(ctx)
			currentUser = userFromContext(ctx)
			p           = payloadConnection{}
		)

		err := json.NewDecoder(r.Body).Decode(&p)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		p.con.FromID = currentUser.ID

		if err := p.con.Validate(); err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		con, err := fn(app, p.con)
		if err != nil {
			if core.IsInvalidEntity(err) {
				respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			} else {
				respondError(w, 0, err)
			}

			return
		}

		respondJSON(w, http.StatusOK, &payloadConnection{con: con})
	}
}

type payloadConnection struct {
	con *connection.Connection
}

func (p *payloadConnection) MarshalJSON() ([]byte, error) {
	f := struct {
		FromID       uint64    `json:"user_from_id"`
		FromIDString string    `json:"user_from_id_string"`
		ToID         uint64    `json:"user_to_id"`
		ToIDString   string    `json:"user_to_id_string"`
		State        string    `json:"state"`
		Type         string    `json:"type"`
		CreatedAt    time.Time `json:"created_at"`
		UpdatedAt    time.Time `json:"updated_at"`
	}{
		FromID:    p.con.FromID,
		ToID:      p.con.ToID,
		State:     string(p.con.State),
		Type:      string(p.con.Type),
		CreatedAt: p.con.CreatedAt,
		UpdatedAt: p.con.UpdatedAt,
	}

	f.FromIDString = strconv.FormatUint(p.con.FromID, 10)
	f.ToIDString = strconv.FormatUint(p.con.ToID, 10)

	return json.Marshal(&f)
}

func (p *payloadConnection) UnmarshalJSON(raw []byte) error {
	f := struct {
		ToID       uint64 `json:"user_to_id"`
		ToIDString string `json:"user_to_id_string"`
		State      string `json:"state"`
		Type       string `json:"type"`
	}{}

	err := json.Unmarshal(raw, &f)
	if err != nil {
		return err
	}

	p.con = &connection.Connection{
		ToID:  f.ToID,
		State: connection.State(f.State),
		Type:  connection.Type(f.Type),
	}

	if f.ToID == 0 {
		if f.ToIDString == "" {
			return fmt.Errorf("user_to_id must be set")
		}

		id, err := strconv.ParseUint(f.ToIDString, 10, 64)
		if err != nil {
			return err
		}

		p.con.ToID = id
	}

	return nil
}

type payloadConnections struct {
	cons    connection.List
	origin  uint64
	userMap user.Map
}

func (p *payloadConnections) MarshalJSON() ([]byte, error) {
	f := struct {
		Incoming      []*payloadConnection `json:"incoming"`
		IncomingCount int                  `json:"incoming_connections_count"`
		Outgoing      []*payloadConnection `json:"outgoing"`
		OutgoingCount int                  `json:"outgoing_connections_count"`
		Users         []*payloadUser       `json:"users"`
		UsersCount    int                  `json:"users_count"`
	}{
		Incoming:   []*payloadConnection{},
		Outgoing:   []*payloadConnection{},
		Users:      []*payloadUser{},
		UsersCount: len(p.userMap),
	}

	for _, c := range p.cons {
		if c.FromID == p.origin {
			f.Outgoing = append(f.Outgoing, &payloadConnection{con: c})
		} else {
			f.Incoming = append(f.Incoming, &payloadConnection{con: c})
		}
	}

	for _, u := range p.userMap {
		f.Users = append(f.Users, &payloadUser{user: u})
	}

	f.IncomingCount = len(f.Incoming)
	f.OutgoingCount = len(f.Outgoing)

	return json.Marshal(f)
}

type payloadSocial struct {
	ConnectionIDs []string
	Platform      string
	State         connection.State
	Type          connection.Type
}

func (p *payloadSocial) UnmarshalJSON(raw []byte) error {
	f := struct {
		ConnectionIDs []string `json:"connection_ids"`
		Platform      string   `json:"platform"`
		State         string   `json:"state"`
		Type          string   `json:"type"`
	}{}

	err := json.Unmarshal(raw, &f)
	if err != nil {
		return err
	}

	if f.State != "" {
		s := connection.State(f.State)
		switch s {
		case connection.StatePending, connection.StateConfirmed, connection.StateRejected:
			p.State = s
		default:
			return fmt.Errorf("invalid state %s", f.State)
		}
	} else {
		p.State = connection.StateConfirmed
	}

	t := connection.Type(f.Type)

	switch t {
	case connection.TypeFollow, connection.TypeFriend:
		p.Type = t
	default:
		return fmt.Errorf("invalid type %s", f.Type)
	}

	p.ConnectionIDs = f.ConnectionIDs
	p.Platform = f.Platform

	return nil
}

func connectionCursorAfter(cs connection.List, limit int) string {
	var after string

	if len(cs) > 0 {
		after = toTimeCursor(cs[0].UpdatedAt)
	}

	return after
}

func connectionCursorBefore(cs connection.List, limit int) string {
	var before string

	if len(cs) > 0 {
		before = toTimeCursor(cs[len(cs)-1].UpdatedAt)
	}

	return before
}
