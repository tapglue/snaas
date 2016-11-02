package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/net/context"

	"github.com/tapglue/api/core"
	"github.com/tapglue/api/service/user"
)

// UserCreate stores the provided user and returns it with a valid session.
func UserCreate(fn core.UserCreateFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			currentApp = appFromContext(ctx)
			deviceID   = deviceIDFromContext(ctx)
			p          = payloadUser{}
			tokenType  = tokenTypeFromContext(ctx)

			origin = createOrigin(deviceID, tokenType, 0)
		)

		err := json.NewDecoder(r.Body).Decode(&p)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		u, err := fn(currentApp, origin, p.user)
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

// UserSearch returns all users for the given search query.
func UserSearch(fn core.UserSearchFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			currentApp  = appFromContext(ctx)
			currentUser = userFromContext(ctx)
			query       = r.URL.Query().Get("q")
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
				userCursorAfter(us, opts.Limit),
				userCursorBefore(us, opts.Limit),
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
			users: us,
		})
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
