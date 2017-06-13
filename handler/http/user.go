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
	"github.com/tapglue/snaas/service/user"
)

// UserCreate stores the provided user and returns it with a valid session.
func UserCreate(
	createFn core.UserCreateFunc,
	createWithInviteFn core.UserCreateWithInviteFunc,
) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			currentApp      = appFromContext(ctx)
			deviceID        = deviceIDFromContext(ctx)
			invite, conType = extractInviteConnections(r)
			p               = payloadUser{}
			tokenType       = tokenTypeFromContext(ctx)
			origin          = createOrigin(deviceID, tokenType, 0)
		)

		err := json.NewDecoder(r.Body).Decode(&p)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		var u *user.User

		if invite {
			u, err = createWithInviteFn(currentApp, origin, p.user, conType)
		} else {
			u, err = createFn(currentApp, origin, p.user)
		}
		if err != nil {
			respondError(w, 0, err)
			return
		}

		respondJSON(w, http.StatusCreated, &payloadUser{user: u})
	}
}

// UserDelete disbales the current user.
func UserDelete(fn core.UserDeleteFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			currentApp  = appFromContext(ctx)
			currentUser = userFromContext(ctx)
		)

		err := fn(currentApp, currentUser)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		respondJSON(w, http.StatusNoContent, nil)
	}
}

// UserLogin finds the user by email or username and creates a Session.
func UserLogin(fn core.UserLoginFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			app       = appFromContext(ctx)
			deviceID  = deviceIDFromContext(ctx)
			p         = payloadLogin{}
			tokenType = tokenTypeFromContext(ctx)

			origin = createOrigin(deviceID, tokenType, 0)
		)

		err := json.NewDecoder(r.Body).Decode(&p)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		if p.email == "" && p.username == "" {
			respondError(w, 0, wrapError(ErrBadRequest, "email or user_name must be set"))
			return
		}

		u, err := fn(app, origin, p.email, p.username, p.password)
		if err != nil {
			if core.IsNotFound(err) {
				respondError(w, 1001, wrapError(ErrUnauthorized, "application user not found"))
				return
			}

			respondError(w, 0, err)
			return
		}

		respondJSON(w, http.StatusCreated, &payloadUser{user: u})
	}
}

// UserLogout finds the session of the user and destroys it.
func UserLogout(fn core.UserLogoutFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			currentApp  = appFromContext(ctx)
			currentUser = userFromContext(ctx)
			token       = tokenFromContext(ctx)
			tokenType   = tokenTypeFromContext(ctx)
		)

		if tokenType == tokenBackend {
			respondJSON(w, http.StatusNoContent, nil)
			return
		}

		err := fn(currentApp, currentUser.ID, token)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		respondJSON(w, http.StatusNoContent, nil)
	}
}

// UserRetrieve returns the user for the requested id.
func UserRetrieve(fn core.UserRetrieveFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			currentApp  = appFromContext(ctx)
			currentUser = userFromContext(ctx)
			deviceID    = deviceIDFromContext(ctx)
			tokenType   = tokenTypeFromContext(ctx)

			origin = createOrigin(deviceID, tokenType, currentUser.ID)
		)

		userID, err := extractUserID(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		u, err := fn(currentApp, origin, userID)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		respondJSON(w, http.StatusOK, &payloadUser{user: u})
	}
}

// UserRetrieveMe returns the current user.
func UserRetrieveMe(fn core.UserRetrieveFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			currentApp  = appFromContext(ctx)
			currentUser = userFromContext(ctx)
			deviceID    = deviceIDFromContext(ctx)
			tokenType   = tokenTypeFromContext(ctx)

			origin = createOrigin(deviceID, tokenType, currentUser.ID)
		)

		u, err := fn(currentApp, origin, currentUser.ID)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		respondJSON(w, http.StatusOK, &payloadUser{user: u})
	}
}

func UserFetchConsole(fn core.UserFetchConsoleFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		appID, err := extractAppID(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		userID, err := extractUserID(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		u, err := fn(appID, userID)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		respondJSON(w, http.StatusOK, &payloadUser{user: u})
	}
}

// UserSearchConsole returns users matching the query.
func UserSearchConsole(fn core.UserSearchConsoleFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		appID, err := extractAppID(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		var (
			query = r.URL.Query().Get(keyUserQuery)
		)

		limit, err := extractLimit(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		offset, err := extractOffsetCursorBefore(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		us, err := fn(appID, query, limit, offset)
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
				limit,
				userSearchCursorAfter(us, limit, offset),
				userSearchCursorBefore(us, limit, offset),
				keyUserQuery, query,
			),
			users: us,
		})
	}
}

// UserSearch returns all users for the given search query.
func UserSearch(fn core.UserSearchFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			currentApp  = appFromContext(ctx)
			currentUser = userFromContext(ctx)
			query       = r.URL.Query().Get(keyUserQuery)
		)

		if len(query) < 3 {
			respondError(w, 0, wrapError(ErrBadRequest, "query must be at least 3 characters"))
			return
		}

		opts, err := extractUserOpts(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		opts.Limit, err = extractLimit(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		opts.Offset, err = extractOffsetCursorBefore(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		us, err := fn(currentApp, currentUser.ID, query, opts)
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
				userSearchCursorAfter(us, opts.Limit, opts.Offset),
				userSearchCursorBefore(us, opts.Limit, opts.Offset),
				keyUserQuery, query,
			),
			users: us,
		})
	}
}

// UserSearchEmails returns all Users for the emails of the payload.
func UserSearchEmails(fn core.UserListByEmailsFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			app         = appFromContext(ctx)
			currentUser = userFromContext(ctx)
			p           = payloadSearchEmails{}
		)

		err := json.NewDecoder(r.Body).Decode(&p)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		if len(p.Emails) == 0 {
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

		opts.Emails = p.Emails

		us, err := fn(app, currentUser.ID, opts)
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

// UserSearchPlatform returns all users for the given ids and platform.
func UserSearchPlatform(fn core.UserListByPlatformIDsFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			platform    = mux.Vars(r)["platform"]
			app         = appFromContext(ctx)
			currentUser = userFromContext(ctx)
			p           = payloadSearchPlatform{}
		)

		err := json.NewDecoder(r.Body).Decode(&p)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		if len(p.IDs) == 0 {
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
			platform: p.IDs,
		}

		us, err := fn(app, currentUser.ID, opts)
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

func UserUpdateConsole(fn core.UserUpdateConsoleFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		appID, err := extractAppID(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		userID, err := extractUserID(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		p := payloadUser{}

		err = json.NewDecoder(r.Body).Decode(&p)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		u, err := fn(appID, userID, p.user.Username)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		respondJSON(w, http.StatusOK, &payloadUser{user: u})
	}
}

// UserUpdate stores the new attributes given.
func UserUpdate(fn core.UserUpdateFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			currentApp  = appFromContext(ctx)
			currentUser = userFromContext(ctx)
			deviceID    = deviceIDFromContext(ctx)
			p           = payloadUser{}
			tokenType   = tokenTypeFromContext(ctx)
		)

		err := json.NewDecoder(r.Body).Decode(&p)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		u, err := fn(
			currentApp,
			createOrigin(deviceID, tokenType, currentUser.ID),
			currentUser,
			p.user,
		)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		respondJSON(w, http.StatusOK, &payloadUser{user: u})
	}
}

type payloadLogin struct {
	email    string
	password string
	username string
	wildcard string
}

func (p *payloadLogin) UnmarshalJSON(raw []byte) error {
	f := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Username string `json:"user_name"`
		Wildcard string `json:"username"`
	}{}

	err := json.Unmarshal(raw, &f)
	if err != nil {
		return err
	}

	if f.Password == "" {
		return fmt.Errorf("password must be set")
	}

	if f.Wildcard != "" {
		f.Email, f.Username = f.Wildcard, f.Wildcard
	}

	if f.Email == "" && f.Username == "" {
		return fmt.Errorf("email or user_name must be provided")
	}

	p.email = f.Email
	p.password = f.Password
	p.username = f.Username

	return nil
}

type payloadSearchEmails struct {
	Emails []string `json:"emails"`
}

type payloadSearchPlatform struct {
	IDs []string `json:"ids"`
}

type payloadUser struct {
	user *user.User
}

func (p *payloadUser) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		About          string                `json:"about"`
		CustomID       string                `json:"custom_id,omitempty"`
		Email          string                `json:"email"`
		Enabled        bool                  `json:"enabled"`
		Firstname      string                `json:"first_name"`
		FollowerCount  int                   `json:"follower_count"`
		FollowingCount int                   `json:"followed_count"`
		FriendCount    int                   `json:"friend_count"`
		ID             uint64                `json:"id"`
		IDString       string                `json:"id_string"`
		Images         map[string]user.Image `json:"images,omitempty"`
		IsFollower     bool                  `json:"is_follower"`
		IsFollowing    bool                  `json:"is_followed"`
		IsFriend       bool                  `json:"is_friend"`
		Lastname       string                `json:"last_name"`
		Metadata       user.Metadata         `json:"metadata,omitempty"`
		Private        *user.Private         `json:"private,omitempty"`
		SessionToken   string                `json:"session_token,omitempty"`
		SocialIDs      map[string]string     `json:"social_ids,omitempty"`
		URL            string                `json:"url,omitempty"`
		Username       string                `json:"user_name"`
		CreatedAt      time.Time             `json:"created_at"`
		UpdatedAt      time.Time             `json:"updated_at"`
	}{
		About:          p.user.About,
		CustomID:       p.user.CustomID,
		Email:          p.user.Email,
		Enabled:        p.user.Enabled,
		Firstname:      p.user.Firstname,
		FollowerCount:  p.user.FollowerCount,
		FollowingCount: p.user.FollowingCount,
		FriendCount:    p.user.FriendCount,
		ID:             p.user.ID,
		IDString:       strconv.FormatUint(p.user.ID, 10),
		Images:         p.user.Images,
		IsFollower:     p.user.IsFollower,
		IsFollowing:    p.user.IsFollowing,
		IsFriend:       p.user.IsFriend,
		Lastname:       p.user.Lastname,
		Metadata:       p.user.Metadata,
		Private:        p.user.Private,
		SessionToken:   p.user.SessionToken,
		SocialIDs:      p.user.SocialIDs,
		URL:            p.user.URL,
		Username:       p.user.Username,
		CreatedAt:      p.user.CreatedAt,
		UpdatedAt:      p.user.UpdatedAt,
	})
}

func (p *payloadUser) UnmarshalJSON(raw []byte) error {
	f := struct {
		About     string                `json:"about"`
		CustomID  string                `json:"custom_id,omitempty"`
		Email     string                `json:"email"`
		Firstname string                `json:"first_name"`
		Images    map[string]user.Image `json:"images,omitempty"`
		Lastname  string                `json:"last_name"`
		Metadata  user.Metadata         `json:"metadata,omitempty"`
		Password  string                `json:"password,omitempty"`
		Private   *user.Private         `json:"private,omitempty"`
		SocialIDs map[string]string     `json:"social_ids"`
		URL       string                `json:"url,omitempty"`
		Username  string                `json:"user_name"`
	}{}

	err := json.Unmarshal(raw, &f)
	if err != nil {
		return err
	}

	if len(f.Metadata) > 5 {
		return fmt.Errorf("metadata fields limit of 5 exceeded")
	}

	p.user = &user.User{
		About:     f.About,
		CustomID:  f.CustomID,
		Email:     f.Email,
		Firstname: f.Firstname,
		Images:    f.Images,
		Lastname:  f.Lastname,
		Metadata:  f.Metadata,
		Password:  f.Password,
		Private:   f.Private,
		SocialIDs: f.SocialIDs,
		URL:       f.URL,
		Username:  f.Username,
	}

	return nil
}

type payloadUsers struct {
	pagination *payloadPagination
	users      user.List
}

func (p *payloadUsers) MarshalJSON() ([]byte, error) {
	ps := []*payloadUser{}

	for _, u := range p.users {
		ps = append(ps, &payloadUser{
			user: u,
		})
	}

	return json.Marshal(struct {
		Pagination *payloadPagination `json:"paging"`
		Users      []*payloadUser     `json:"users"`
		UsersCount int                `json:"users_count"`
	}{
		Pagination: p.pagination,
		Users:      ps,
		UsersCount: len(ps),
	})
}

type payloadUserMap struct {
	userMap user.Map
}

func (p *payloadUserMap) MarshalJSON() ([]byte, error) {
	m := map[string]*payloadUser{}

	for id, u := range p.userMap {
		m[strconv.FormatUint(id, 10)] = &payloadUser{user: u}
	}

	return json.Marshal(m)
}

func userCursorAfter(us user.List, limit int) string {
	var after string

	if len(us) > 0 {
		after = toIDCursor(us[0].ID)
	}

	return after
}

func userCursorBefore(us user.List, limit int) string {
	var before string

	if len(us) > 0 {
		before = toIDCursor(us[len(us)-1].ID)
	}

	return before
}

func userSearchCursorAfter(us user.List, limit int, offset uint) string {
	if offset == 0 || offset <= uint(limit) {
		return toOffsetCursor(0)
	}

	return toOffsetCursor(offset - uint(limit))
}

func userSearchCursorBefore(us user.List, limit int, offset uint) string {
	return toOffsetCursor(uint(limit) + offset)
}
