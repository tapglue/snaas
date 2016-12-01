package core

import (
	"sort"

	"github.com/tapglue/snaas/service/app"
	"github.com/tapglue/snaas/service/connection"
	"github.com/tapglue/snaas/service/user"
)

// ConnectionFeed is the composite to transport information relevant for
// connections.
type ConnectionFeed struct {
	Connections connection.List
	Users       user.List
	UserMap     user.Map
}

// ConnectionByStateFunc returns all connections for the given origin and state.
type ConnectionByStateFunc func(
	currentApp *app.App,
	originID uint64,
	state connection.State,
	opts connection.QueryOptions,
) (*ConnectionFeed, error)

// ConnectionByState returns all connections for the given origin and state.
func ConnectionByState(
	connections connection.Service,
	users user.Service,
) ConnectionByStateFunc {
	return func(
		currentApp *app.App,
		origin uint64,
		state connection.State,
		opts connection.QueryOptions,
	) (*ConnectionFeed, error) {
		switch state {
		case connection.StatePending, connection.StateConfirmed, connection.StateRejected:
			// valid
		default:
			return nil, wrapError(ErrInvalidEntity, "unsupported state %s", string(state))
		}

		ics, err := connections.Query(currentApp.Namespace(), connection.QueryOptions{
			Before:  opts.Before,
			Enabled: &defaultEnabled,
			FromIDs: []uint64{origin},
			Limit:   opts.Limit,
			States:  []connection.State{state},
		})
		if err != nil {
			return nil, err
		}

		ocs, err := connections.Query(currentApp.Namespace(), connection.QueryOptions{
			Before:  opts.Before,
			Enabled: &defaultEnabled,
			Limit:   opts.Limit,
			States:  []connection.State{state},
			ToIDs:   []uint64{origin},
		})
		if err != nil {
			return nil, err
		}

		cons := append(ics, ocs...)

		sort.Sort(cons)

		if len(cons) == 0 {
			return &ConnectionFeed{
				Connections: connection.List{},
				UserMap:     user.Map{},
			}, nil
		}

		if len(cons) > opts.Limit {
			cons = cons[:opts.Limit]
		}

		ids := []uint64{}

		for _, c := range cons {
			if c.FromID == origin {
				ids = append(ids, c.ToID)
			} else {
				ids = append(ids, c.FromID)
			}
		}

		um, err := user.MapFromIDs(
			users,
			currentApp.Namespace(),
			ids...,
		)
		if err != nil {
			return nil, err
		}

		for _, u := range um {
			err = enrichRelation(connections, currentApp, origin, u)
			if err != nil {
				return nil, err
			}
		}

		return &ConnectionFeed{
			Connections: cons,
			UserMap:     um,
		}, nil
	}
}

// ConnectionCreateSocialFunc connects the origin with the users matching the
// platform ids.
type ConnectionCreateSocialFunc func(
	currentApp *app.App,
	originID uint64,
	connectionType connection.Type,
	connectionState connection.State,
	opts user.QueryOptions,
) (user.List, error)

// ConnectionCreateSocial connects the origin with the users matching the
// platform ids.
func ConnectionCreateSocial(
	connections connection.Service,
	users user.Service,
) ConnectionCreateSocialFunc {
	return func(
		currentApp *app.App,
		originID uint64,
		connectionType connection.Type,
		connectionState connection.State,
		opts user.QueryOptions,
	) (user.List, error) {
		opts.Enabled = &defaultEnabled

		us, err := users.Query(currentApp.Namespace(), opts)
		if err != nil {
			return nil, err
		}

		for _, u := range us {
			_, err := connections.Put(currentApp.Namespace(), &connection.Connection{
				Enabled: true,
				FromID:  originID,
				ToID:    u.ID,
				State:   connectionState,
				Type:    connectionType,
			})
			if err != nil {
				return nil, err
			}

			r, err := queryRelation(connections, currentApp, originID, u.ID)
			if err != nil {
				return nil, err
			}

			u.IsFollower = r.isFollower
			u.IsFollowing = r.isFollowing
			u.IsFriend = r.isFriend
		}

		return us, nil
	}
}

// ConnectionDeleteFunc disables the given connection.
type ConnectionDeleteFunc func(
	currentApp *app.App,
	con *connection.Connection,
) error

// ConnectionDelete disables the given connection.
func ConnectionDelete(
	connections connection.Service,
) ConnectionDeleteFunc {
	return func(
		currentApp *app.App,
		con *connection.Connection,
	) error {
		var (
			fromIDs = []uint64{con.FromID}
			toIDs   = []uint64{con.ToID}
		)

		if con.Type == connection.TypeFriend {
			fromIDs = []uint64{con.FromID, con.ToID}
			toIDs = []uint64{con.FromID, con.ToID}
		}

		cs, err := connections.Query(currentApp.Namespace(), connection.QueryOptions{
			Enabled: &defaultEnabled,
			FromIDs: fromIDs,
			Limit:   1,
			ToIDs:   toIDs,
			Types:   []connection.Type{con.Type},
		})
		if err != nil {
			return err
		}

		if len(cs) == 0 {
			return nil
		}

		con = cs[0]

		con.Enabled = false

		_, err = connections.Put(currentApp.Namespace(), con)

		return err
	}
}

// ConnectionFollowerIDsFunc returns the list of ids of users who follow origin.
type ConnectionFollowerIDsFunc func(
	currentApp *app.App,
	origin uint64,
) ([]uint64, error)

// ConnectionFollowerIDs returns the list of ids of users who follow origin.
func ConnectionFollowerIDs(
	connections connection.Service,
) ConnectionFollowerIDsFunc {
	return func(currentApp *app.App, origin uint64) ([]uint64, error) {
		fs, err := connections.Query(currentApp.Namespace(), connection.QueryOptions{
			Enabled: &defaultEnabled,
			States: []connection.State{
				connection.StateConfirmed,
			},
			ToIDs: []uint64{
				origin,
			},
			Types: []connection.Type{
				connection.TypeFollow,
			},
		})
		if err != nil {
			return nil, err
		}

		return fs.FromIDs(), nil
	}
}

// ConnectionFollowersFunc returns the list of users who follow the origin.
type ConnectionFollowersFunc func(
	currentApp *app.App,
	origin uint64,
	userID uint64,
	opts connection.QueryOptions,
) (*ConnectionFeed, error)

// ConnectionFollowers returns the list of users who follow the origin.
func ConnectionFollowers(
	connections connection.Service,
	users user.Service,
) ConnectionFollowersFunc {
	return func(
		currentApp *app.App,
		origin uint64,
		userID uint64,
		opts connection.QueryOptions,
	) (*ConnectionFeed, error) {
		cs, err := connections.Query(currentApp.Namespace(), connection.QueryOptions{
			Before:  opts.Before,
			Enabled: &defaultEnabled,
			Limit:   opts.Limit,
			ToIDs:   []uint64{userID},
			States:  []connection.State{connection.StateConfirmed},
			Types:   []connection.Type{connection.TypeFollow},
		})
		if err != nil {
			return nil, err
		}

		us, err := user.ListFromIDs(users, currentApp.Namespace(), cs.FromIDs()...)
		if err != nil {
			return nil, err
		}

		for _, u := range us {
			err := enrichConnectionCounts(connections, users, currentApp, u)
			if err != nil {
				return nil, err
			}

			err = enrichRelation(connections, currentApp, origin, u)
			if err != nil {
				return nil, err
			}
		}

		return &ConnectionFeed{
			Connections: cs,
			Users:       us,
		}, nil
	}
}

// ConnectionFollowingsFunc returns the list of users the origin is following.
type ConnectionFollowingsFunc func(
	currentApp *app.App,
	origin uint64,
	userID uint64,
	opts connection.QueryOptions,
) (*ConnectionFeed, error)

// ConnectionFollowings returns the list of users the origin is following.
func ConnectionFollowings(
	connections connection.Service,
	users user.Service,
) ConnectionFollowingsFunc {
	return func(
		currentApp *app.App,
		origin uint64,
		userID uint64,
		opts connection.QueryOptions,
	) (*ConnectionFeed, error) {
		cs, err := connections.Query(currentApp.Namespace(), connection.QueryOptions{
			Before:  opts.Before,
			Enabled: &defaultEnabled,
			FromIDs: []uint64{userID},
			Limit:   opts.Limit,
			States:  []connection.State{connection.StateConfirmed},
			Types:   []connection.Type{connection.TypeFollow},
		})
		if err != nil {
			return nil, err
		}

		us, err := user.ListFromIDs(users, currentApp.Namespace(), cs.ToIDs()...)
		if err != nil {
			return nil, err
		}

		for _, u := range us {
			err := enrichConnectionCounts(connections, users, currentApp, u)
			if err != nil {
				return nil, err
			}

			err = enrichRelation(connections, currentApp, origin, u)
			if err != nil {
				return nil, err
			}
		}

		return &ConnectionFeed{
			Connections: cs,
			Users:       us,
		}, nil
	}
}

// ConnectionFriendIDsFunc returns the list of ids of users who are friends with
// origin.
type ConnectionFriendIDsFunc func(
	currentApp *app.App,
	origin uint64,
) ([]uint64, error)

// ConnectionFriendIDs returns the list of ids of users who are friends with
// origin.
func ConnectionFriendIDs(connections connection.Service) ConnectionFriendIDsFunc {
	return func(currentApp *app.App, origin uint64) ([]uint64, error) {
		fs, err := connections.Query(currentApp.Namespace(), connection.QueryOptions{
			Enabled: &defaultEnabled,
			FromIDs: []uint64{
				origin,
			},
			States: []connection.State{
				connection.StateConfirmed,
			},
			Types: []connection.Type{
				connection.TypeFriend,
			},
		})
		if err != nil {
			return nil, err
		}

		ts, err := connections.Query(currentApp.Namespace(), connection.QueryOptions{
			Enabled: &defaultEnabled,
			ToIDs: []uint64{
				origin,
			},
			States: []connection.State{
				connection.StateConfirmed,
			},
			Types: []connection.Type{
				connection.TypeFriend,
			},
		})
		if err != nil {
			return nil, err
		}

		return append(fs.ToIDs(), ts.FromIDs()...), nil
	}
}

// ConnectionFriendsFunc returns the list of users the origin is friends with.
type ConnectionFriendsFunc func(
	currentApp *app.App,
	origin uint64,
	userID uint64,
	opts connection.QueryOptions,
) (*ConnectionFeed, error)

// ConnectionFriends returns the list of users the origin is friends with.
func ConnectionFriends(
	connections connection.Service,
	users user.Service,
) ConnectionFriendsFunc {
	return func(
		currentApp *app.App,
		origin uint64,
		userID uint64,
		opts connection.QueryOptions,
	) (*ConnectionFeed, error) {
		fs, err := connections.Query(currentApp.Namespace(), connection.QueryOptions{
			Before:  opts.Before,
			Enabled: &defaultEnabled,
			FromIDs: []uint64{userID},
			Limit:   opts.Limit,
			States:  []connection.State{connection.StateConfirmed},
			Types:   []connection.Type{connection.TypeFriend},
		})
		if err != nil {
			return nil, err
		}

		ts, err := connections.Query(currentApp.Namespace(), connection.QueryOptions{
			Before:  opts.Before,
			Enabled: &defaultEnabled,
			Limit:   opts.Limit,
			ToIDs:   []uint64{userID},
			States:  []connection.State{connection.StateConfirmed},
			Types:   []connection.Type{connection.TypeFriend},
		})
		if err != nil {
			return nil, err
		}

		cs := append(fs, ts...)

		sort.Sort(cs)

		if len(cs) > opts.Limit {
			cs = cs[:opts.Limit-1]
		}

		ids := []uint64{}

		for _, con := range cs {
			if con.FromID == userID {
				ids = append(ids, con.ToID)
			} else {
				ids = append(ids, con.FromID)
			}
		}

		us, err := user.ListFromIDs(
			users,
			currentApp.Namespace(),
			ids...,
		)
		if err != nil {
			return nil, err
		}

		for _, u := range us {
			err := enrichConnectionCounts(connections, users, currentApp, u)
			if err != nil {
				return nil, err
			}

			err = enrichRelation(connections, currentApp, origin, u)
			if err != nil {
				return nil, err
			}
		}

		return &ConnectionFeed{
			Connections: cs,
			Users:       us,
		}, nil
	}
}

// ConnectionUpdateFunc transitions the passed Connection to its new state.
type ConnectionUpdateFunc func(
	currentApp *app.App,
	new *connection.Connection,
) (*connection.Connection, error)

// ConnectionUpdate transitions the passed Connection to its new state.
func ConnectionUpdate(
	connections connection.Service,
	users user.Service,
) ConnectionUpdateFunc {
	return func(
		currentApp *app.App,
		new *connection.Connection,
	) (*connection.Connection, error) {
		us, err := users.Query(currentApp.Namespace(), user.QueryOptions{
			Enabled: &defaultEnabled,
			IDs: []uint64{
				new.ToID,
			},
		})
		if err != nil {
			return nil, err
		}

		if len(us) != 1 {
			return nil, ErrNotFound
		}

		var (
			fromIDs = []uint64{new.FromID}
			toIDs   = []uint64{new.ToID}
		)

		if new.Type == connection.TypeFriend {
			fromIDs = []uint64{new.FromID, new.ToID}
			toIDs = []uint64{new.FromID, new.ToID}
		}

		cs, err := connections.Query(currentApp.Namespace(), connection.QueryOptions{
			Enabled: &defaultEnabled,
			FromIDs: fromIDs,
			Limit:   1,
			ToIDs:   toIDs,
			Types:   []connection.Type{new.Type},
		})
		if err != nil {
			return nil, err
		}

		if len(cs) > 0 && cs[0].State == new.State {
			return cs[0], nil
		}

		var old *connection.Connection

		if len(cs) > 0 {
			old = cs[0]

			new.FromID = old.FromID
			new.ToID = old.ToID
		}

		new.Enabled = true

		if err := validateConTransition(old, new); err != nil {
			return nil, err
		}

		return connections.Put(currentApp.Namespace(), new)
	}
}

func validateConTransition(old, new *connection.Connection) error {
	if old == nil {
		return nil
	}

	if old.FromID != new.FromID {
		return wrapError(
			ErrInvalidEntity,
			"from id miss-match %d != %d",
			old.FromID,
			new.FromID,
		)
	}

	if old.ToID != new.ToID {
		return wrapError(
			ErrInvalidEntity,
			"to id miss-match %d != %d",
			old.ToID,
			new.ToID,
		)
	}

	if old.Type != new.Type {
		return wrapError(
			ErrInvalidEntity,
			"type miss-match %s != %s",
			string(old.Type),
			string(new.Type),
		)
	}

	if old.State == new.State {
		return nil
	}

	switch old.State {
	case connection.StatePending:
		switch new.State {
		case connection.StateConfirmed, connection.StateRejected:
			return nil
		}
	case connection.StateConfirmed:
		switch new.State {
		case connection.StateRejected:
			return nil
		}
	}

	return wrapError(
		ErrInvalidEntity,
		"invalid state transition from %s to %s",
		string(old.State),
		string(new.State),
	)
}

type relation struct {
	isFriend    bool
	isFollower  bool
	isFollowing bool
	isSelf      bool
}

func queryRelation(
	connections connection.Service,
	currentApp *app.App,
	origin, userID uint64,
) (*relation, error) {
	if origin == userID {
		return &relation{isSelf: true}, nil
	}

	cs, err := connections.Query(currentApp.Namespace(), connection.QueryOptions{
		Enabled: &defaultEnabled,
		FromIDs: []uint64{
			origin,
			userID,
		},
		States: []connection.State{
			connection.StateConfirmed,
		},
		ToIDs: []uint64{
			origin,
			userID,
		},
	})
	if err != nil {
		return nil, err
	}

	r := &relation{}

	for _, c := range cs {
		if c.Type == connection.TypeFriend {
			r.isFriend = true
		}

		if c.Type == connection.TypeFollow && c.FromID == origin {
			r.isFollowing = true
		}

		if c.Type == connection.TypeFollow && c.ToID == origin {
			r.isFollower = true
		}
	}

	return r, nil
}
