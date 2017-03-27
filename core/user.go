package core

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/tapglue/snaas/platform/generate"
	"github.com/tapglue/snaas/service/app"
	"github.com/tapglue/snaas/service/connection"
	"github.com/tapglue/snaas/service/session"
	"github.com/tapglue/snaas/service/user"
)

// UserCreateFunc stores the provided user and creates a session.
type UserCreateFunc func(
	currentApp *app.App,
	origin Origin,
	u *user.User,
) (*user.User, error)

// UserCreate stores the provided user and creates a session.
func UserCreate(
	sessions session.Service,
	users user.Service,
) UserCreateFunc {
	return func(
		currentApp *app.App,
		origin Origin,
		u *user.User,
	) (*user.User, error) {
		if err := userConstrainPrivate(origin, u.Private); err != nil {
			return nil, err
		}

		if err := u.Validate(); err != nil {
			return nil, wrapError(ErrInvalidEntity, "%s", err)
		}

		epw, err := passwordSecure(u.Password)
		if err != nil {
			return nil, err
		}

		u.Enabled = true
		u.Password = epw

		u, err = users.Put(currentApp.Namespace(), u)
		if err != nil {
			return nil, err
		}

		err = enrichSessionToken(sessions, currentApp, u, origin.DeviceID)
		if err != nil {
			return nil, err
		}

		return u, nil
	}
}

// UserDeleteFunc disables the user.
type UserDeleteFunc func(
	currentApp *app.App,
	origin *user.User,
) error

// UserDelete disables the user.
func UserDelete(
	users user.Service,
) UserDeleteFunc {
	return func(
		currentApp *app.App,
		origin *user.User,
	) error {
		origin.Enabled = false
		origin.Deleted = true

		_, err := users.Put(currentApp.Namespace(), origin)
		return err
	}
}

// UserFetchFunc returns the User for the given id.
type UserFetchFunc func(currentApp *app.App, id uint64) (*user.User, error)

// UserFetch returns the User for the given id.
func UserFetch(users user.Service) UserFetchFunc {
	return func(currentApp *app.App, id uint64) (*user.User, error) {
		us, err := users.Query(currentApp.Namespace(), user.QueryOptions{
			Enabled: &defaultEnabled,
			IDs: []uint64{
				id,
			},
		})
		if err != nil {
			return nil, err
		}

		if len(us) == 0 {
			return nil, ErrNotFound
		}

		return us[0], nil
	}
}

// UserListByEmailsFunc returns all users for the given emails.
type UserListByEmailsFunc func(
	currentApp *app.App,
	originID uint64,
	opts user.QueryOptions,
) (user.List, error)

// UserListByEmails returns all users for the given emails.
func UserListByEmails(
	connections connection.Service,
	users user.Service,
) UserListByEmailsFunc {
	return func(
		currentApp *app.App,
		originID uint64,
		opts user.QueryOptions,
	) (user.List, error) {
		us, err := users.Query(currentApp.Namespace(), user.QueryOptions{
			Before:  opts.Before,
			Enabled: &defaultEnabled,
			Emails:  opts.Emails,
			Limit:   opts.Limit,
		})
		if err != nil {
			return nil, err
		}

		for _, u := range us {
			r, err := queryRelation(connections, currentApp, originID, u.ID)
			if err != nil {
				return nil, err
			}

			u.IsFriend = r.isFriend
			u.IsFollower = r.isFollower
			u.IsFollowing = r.isFollowing
		}

		return us, nil
	}
}

// UserListByPlatformIDsFunc returns all users for the given ids for the social
// platform.
type UserListByPlatformIDsFunc func(
	currentApp *app.App,
	originID uint64,
	opts user.QueryOptions,
) (user.List, error)

// UserListByPlatformIDs returns all users for the given ids for the social
// platform.
func UserListByPlatformIDs(
	connections connection.Service,
	users user.Service,
) UserListByPlatformIDsFunc {
	return func(
		currentApp *app.App,
		originID uint64,
		opts user.QueryOptions,
	) (user.List, error) {
		us, err := users.Query(currentApp.Namespace(), user.QueryOptions{
			Before:    opts.Before,
			Enabled:   &defaultEnabled,
			Limit:     opts.Limit,
			SocialIDs: opts.SocialIDs,
		})
		if err != nil {
			return nil, err
		}

		for _, u := range us {
			r, err := queryRelation(connections, currentApp, originID, u.ID)
			if err != nil {
				return nil, err
			}

			u.IsFriend = r.isFriend
			u.IsFollower = r.isFollower
			u.IsFollowing = r.isFollowing
		}

		return us, nil
	}
}

// UserLoginFunc finds the user by email or username and returns them with a
// valid session token.
type UserLoginFunc func(
	currentApp *app.App,
	origin Origin,
	email, username, password string,
) (*user.User, error)

// UserLogin finds the user by email or username and returns them with a valid
// session token.
func UserLogin(
	connections connection.Service,
	sessions session.Service,
	users user.Service,
) UserLoginFunc {
	return func(
		currentApp *app.App,
		origin Origin,
		email, username, password string,
	) (*user.User, error) {
		us, err := users.Query(currentApp.Namespace(), user.QueryOptions{
			Enabled: &defaultEnabled,
			Emails: []string{
				email,
			},
		})
		if err != nil {
			return nil, err
		}

		if len(us) == 1 {
			return login(connections, sessions, users, currentApp, us[0], password, origin.DeviceID)
		}

		us, err = users.Query(currentApp.Namespace(), user.QueryOptions{
			Enabled: &defaultEnabled,
			Usernames: []string{
				username,
			},
		})
		if err != nil {
			return nil, err
		}

		if len(us) == 1 {
			return login(connections, sessions, users, currentApp, us[0], password, origin.DeviceID)
		}

		return nil, ErrNotFound
	}
}

// UserLogoutFunc destroys the session stored under token.
type UserLogoutFunc func(
	currentApp *app.App,
	origin uint64,
	token string,
) error

// UserLogout destroys the session stored under token.
func UserLogout(
	sessions session.Service,
) UserLogoutFunc {
	return func(
		currentApp *app.App,
		origin uint64,
		token string,
	) error {
		ss, err := sessions.Query(currentApp.Namespace(), session.QueryOptions{
			Enabled: &defaultEnabled,
			IDs: []string{
				token,
			},
			UserIDs: []uint64{
				origin,
			},
		})
		if err != nil {
			return err
		}

		if len(ss) == 0 {
			return nil
		}

		s := ss[0]
		s.Enabled = false

		_, err = sessions.Put(currentApp.Namespace(), s)
		return err
	}
}

// UserRetrieveFunc returns the user for the given id.
type UserRetrieveFunc func(
	currentApp *app.App,
	origin Origin,
	userID uint64,
) (*user.User, error)

// UserRetrieve returns the user for the given id.
func UserRetrieve(
	connections connection.Service,
	sessions session.Service,
	users user.Service,
) UserRetrieveFunc {
	return func(
		currentApp *app.App,
		origin Origin,
		userID uint64,
	) (*user.User, error) {
		us, err := users.Query(currentApp.Namespace(), user.QueryOptions{
			Enabled: &defaultEnabled,
			IDs: []uint64{
				userID,
			},
		})
		if err != nil {
			return nil, err
		}

		if len(us) != 1 {
			return nil, ErrNotFound
		}

		u := us[0]

		err = enrichRelation(connections, currentApp, origin.UserID, u)
		if err != nil {
			return nil, err
		}

		err = enrichConnectionCounts(connections, users, currentApp, u)
		if err != nil {
			return nil, err
		}

		if origin.UserID == userID {
			err = enrichSessionToken(sessions, currentApp, u, origin.DeviceID)
			if err != nil {
				return nil, err
			}
		}

		return u, nil
	}
}

// UserSearchFunc returns all users for the given query.
type UserSearchFunc func(
	currentApp *app.App,
	origin uint64,
	query string,
	opts user.QueryOptions,
) (user.List, error)

// UserSearch returns all users for the given query.
func UserSearch(
	connections connection.Service,
	users user.Service,
) UserSearchFunc {
	return func(
		currentApp *app.App,
		origin uint64,
		query string,
		opts user.QueryOptions,
	) (user.List, error) {
		t := []string{query}

		us, err := users.Search(currentApp.Namespace(), user.QueryOptions{
			Before:     opts.Before,
			Enabled:    &defaultEnabled,
			Emails:     t,
			Firstnames: t,
			Lastnames:  t,
			Limit:      opts.Limit,
			Usernames:  t,
		})
		if err != nil {
			return nil, err
		}

		for _, u := range us {
			err = enrichConnectionCounts(connections, users, currentApp, u)
			if err != nil {
				return nil, err
			}

			err = enrichRelation(connections, currentApp, origin, u)
			if err != nil {
				return nil, err
			}
		}

		return us, nil
	}
}

// UserUpdateFunc stores the new attributes for the user.
type UserUpdateFunc func(
	currentApp *app.App,
	origin Origin,
	old *user.User,
	new *user.User,
) (*user.User, error)

// UserUpdate stores the new attributes for the user.
func UserUpdate(
	connections connection.Service,
	sessions session.Service,
	users user.Service,
) UserUpdateFunc {
	return func(
		currentApp *app.App,
		origin Origin,
		old *user.User,
		new *user.User,
	) (*user.User, error) {
		if err := userConstrainPrivate(origin, new.Private); err != nil {
			return nil, err
		}

		new.Enabled = true
		new.ID = old.ID

		if new.Password != "" {
			epw, err := passwordSecure(new.Password)
			if err != nil {
				return nil, err
			}

			new.Password = epw
		} else {
			new.Password = old.Password
		}

		if new.Private == nil {
			new.Private = old.Private
		}

		u, err := users.Put(currentApp.Namespace(), new)
		if err != nil {
			return nil, err
		}

		err = enrichConnectionCounts(connections, users, currentApp, u)
		if err != nil {
			return nil, err
		}

		err = enrichSessionToken(sessions, currentApp, u, origin.DeviceID)
		if err != nil {
			return nil, err
		}

		return u, nil
	}
}

// UsersFetchFunc retrieves the users for the given ids.
type UsersFetchFunc func(currentApp *app.App, ids ...uint64) (user.List, error)

func UsersFetch(users user.Service) UsersFetchFunc {
	return func(currentApp *app.App, ids ...uint64) (user.List, error) {
		if len(ids) == 0 {
			return user.List{}, nil
		}

		return users.Query(currentApp.Namespace(), user.QueryOptions{
			Enabled: &defaultEnabled,
			IDs:     ids,
		})
	}
}

func enrichConnectionCounts(
	connections connection.Service,
	users user.Service,
	currentApp *app.App,
	u *user.User,
) error {
	cs, err := connections.Query(currentApp.Namespace(), connection.QueryOptions{
		Enabled: &defaultEnabled,
		States: []connection.State{
			connection.StateConfirmed,
		},
		ToIDs: []uint64{
			u.ID,
		},
		Types: []connection.Type{
			connection.TypeFollow,
		},
	})
	if err != nil {
		return err
	}

	if len(cs) > 0 {
		u.FollowerCount, err = users.Count(currentApp.Namespace(), user.QueryOptions{
			Enabled: &defaultEnabled,
			IDs:     cs.FromIDs(),
		})
		if err != nil {
			return err
		}
	}

	cs, err = connections.Query(currentApp.Namespace(), connection.QueryOptions{
		Enabled: &defaultEnabled,
		FromIDs: []uint64{
			u.ID,
		},
		States: []connection.State{
			connection.StateConfirmed,
		},
		Types: []connection.Type{
			connection.TypeFollow,
		},
	})
	if err != nil {
		return err
	}

	if len(cs) > 0 {
		u.FollowingCount, err = users.Count(currentApp.Namespace(), user.QueryOptions{
			Enabled: &defaultEnabled,
			IDs:     cs.ToIDs(),
		})
		if err != nil {
			return err
		}
	}

	fs, err := connections.Query(currentApp.Namespace(), connection.QueryOptions{
		Enabled: &defaultEnabled,
		FromIDs: []uint64{
			u.ID,
		},
		States: []connection.State{
			connection.StateConfirmed,
		},
		Types: []connection.Type{
			connection.TypeFriend,
		},
	})
	if err != nil {
		return err
	}

	ts, err := connections.Query(currentApp.Namespace(), connection.QueryOptions{
		Enabled: &defaultEnabled,
		States: []connection.State{
			connection.StateConfirmed,
		},
		ToIDs: []uint64{
			u.ID,
		},
		Types: []connection.Type{
			connection.TypeFriend,
		},
	})
	if err != nil {
		return err
	}

	ids := append(fs.ToIDs(), ts.FromIDs()...)

	if len(ids) > 0 {
		u.FriendCount, err = users.Count(currentApp.Namespace(), user.QueryOptions{
			Enabled: &defaultEnabled,
			IDs:     ids,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func enrichRelation(
	s connection.Service,
	currentApp *app.App,
	origin uint64,
	u *user.User,
) error {
	if origin == u.ID {
		return nil
	}

	r, err := queryRelation(s, currentApp, origin, u.ID)
	if err != nil {
		return err
	}

	u.IsFriend = r.isFriend
	u.IsFollower = r.isFollower
	u.IsFollowing = r.isFollowing

	return nil
}

func enrichSessionToken(
	sessions session.Service,
	currentApp *app.App,
	u *user.User,
	deviceID string,
) error {
	ss, err := sessions.Query(currentApp.Namespace(), session.QueryOptions{
		DeviceIDs: []string{
			deviceID,
		},
		Enabled: &defaultEnabled,
		UserIDs: []uint64{
			u.ID,
		},
	})
	if err != nil {
		return err
	}

	var s *session.Session

	if len(ss) > 0 {
		s = ss[0]
	} else {
		s, err = sessions.Put(currentApp.Namespace(), &session.Session{
			DeviceID: deviceID,
			Enabled:  true,
			UserID:   u.ID,
		})
		if err != nil {
			return err
		}
	}

	u.SessionToken = s.ID

	return nil
}

func login(
	connections connection.Service,
	sessions session.Service,
	users user.Service,
	currentApp *app.App,
	u *user.User,
	password string,
	deviceID string,
) (*user.User, error) {
	valid, err := passwordCompare(password, u.Password)
	if err != nil {
		return nil, ErrNotFound
	}

	if !valid {
		return nil, wrapError(ErrUnauthorized, "wrong credentials")
	}

	err = enrichSessionToken(sessions, currentApp, u, deviceID)
	if err != nil {
		return nil, err
	}

	err = enrichConnectionCounts(connections, users, currentApp, u)
	if err != nil {
		return nil, err
	}

	return u, nil
}

func passwordCompare(dec, enc string) (bool, error) {
	d, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return false, err
	}

	ps := strings.SplitN(string(d), ":", 3)

	epw, err := base64.StdEncoding.DecodeString(ps[2])
	if err != nil {
		return false, err
	}

	salt, err := base64.StdEncoding.DecodeString(ps[0])
	if err != nil {
		return false, err
	}

	ts, err := base64.StdEncoding.DecodeString(ps[1])
	if err != nil {
		return false, err
	}

	esalt := []byte{}
	esalt = append(esalt, []byte(salt)...)
	esalt = append(esalt, []byte(":")...)
	esalt = append(esalt, []byte(ts)...)

	ipw, err := generate.EncryptPassword([]byte(dec), esalt)
	if err != nil {
		return false, err
	}

	return string(epw) == string(ipw), nil
}

func passwordSecure(pw string) (string, error) {
	// create Salt
	salt, err := generate.Salt()
	if err != nil {
		return "", err
	}

	// create scrypt salt
	var (
		esalt = []byte{}
		ts    = []byte(time.Now().Format(time.RFC3339))
	)

	esalt = append(esalt, salt...)
	esalt = append(esalt, []byte(":")...)
	esalt = append(esalt, ts...)

	// encrypt
	epw, err := generate.EncryptPassword([]byte(pw), esalt)
	if err != nil {
		return "", err
	}

	// encode
	enc := fmt.Sprintf(
		"%s:%s:%s",
		base64.StdEncoding.EncodeToString(salt),
		base64.StdEncoding.EncodeToString(ts),
		base64.StdEncoding.EncodeToString(epw),
	)

	return base64.StdEncoding.EncodeToString([]byte(enc)), nil
}

func userConstrainPrivate(origin Origin, private *user.Private) error {
	if !origin.IsBackend() && private != nil {
		return wrapError(
			ErrUnauthorized,
			"private can only be set by backend integration",
		)
	}

	return nil
}
