package core

import (
	"crypto/md5"
	"fmt"
	"math/rand"
	"time"

	"github.com/tapglue/snaas/platform/generate"
	"github.com/tapglue/snaas/service/app"
	"github.com/tapglue/snaas/service/connection"
	"github.com/tapglue/snaas/service/device"
	"github.com/tapglue/snaas/service/object"
	"github.com/tapglue/snaas/service/rule"
	"github.com/tapglue/snaas/service/user"
)

// App bundles the App service with counts.
type App struct {
	Counts AppCounts

	*app.App
}

// AppCounts bundles related entity counts together.
type AppCounts struct {
	Connections uint
	Comments    uint
	Devices     uint
	Posts       uint
	Rules       uint
	Users       uint
}

// AppCreateFunc creates a new App.
type AppCreateFunc func(name, description string) (*app.App, error)

// AppCreate creates a new App.
func AppCreate(apps app.Service) AppCreateFunc {
	return func(name, description string) (*app.App, error) {
		token, backendToken, err := generateTokens()
		if err != nil {
			return nil, err
		}

		return apps.Put(app.NamespaceDefault, &app.App{
			BackendToken: backendToken,
			Description:  description,
			Enabled:      true,
			InProduction: false,
			Name:         name,
			Token:        token,
		})
	}
}

// AppFetchFunc returns the App for the given id.
type AppFetchFunc func(id uint64) (*app.App, error)

// AppFetch returns the App for the given id.
func AppFetch(apps app.Service) AppFetchFunc {
	return func(id uint64) (*app.App, error) {
		as, err := apps.Query(app.NamespaceDefault, app.QueryOptions{
			Enabled: &defaultEnabled,
			IDs: []uint64{
				id,
			},
		})
		if err != nil {
			return nil, err
		}

		if len(as) == 0 {
			return nil, wrapError(ErrNotFound, "app (%d) not found", id)
		}

		return as[0], nil
	}
}

// AppFetchWithCountsFunc returns the App for the given id.
type AppFetchWithCountsFunc func(id uint64) (*App, error)

// AppFetchWithCounts returns the App for the given id.
func AppFetchWithCounts(
	apps app.Service,
	connections connection.Service,
	devices device.Service,
	objects object.Service,
	rules rule.Service,
	users user.Service,
) AppFetchWithCountsFunc {
	return func(id uint64) (*App, error) {
		as, err := apps.Query(app.NamespaceDefault, app.QueryOptions{
			Enabled: &defaultEnabled,
			IDs: []uint64{
				id,
			},
		})
		if err != nil {
			return nil, err
		}

		if len(as) == 0 {
			return nil, wrapError(ErrNotFound, "app (%d) not found", id)
		}

		a := as[0]

		commentCount, err := objects.Count(a.Namespace(), object.QueryOptions{
			Types: []string{
				TypeComment,
			},
		})
		if err != nil {
			return nil, err
		}

		connectionCount, err := connections.Count(a.Namespace(), connection.QueryOptions{
			Enabled: &defaultEnabled,
		})
		if err != nil {
			return nil, err
		}

		deviceCount, err := devices.Count(a.Namespace(), device.QueryOptions{
			Deleted: &defaultDeleted,
		})
		if err != nil {
			return nil, err
		}

		postCount, err := objects.Count(a.Namespace(), object.QueryOptions{
			Types: []string{
				TypePost,
			},
		})
		if err != nil {
			return nil, err
		}

		rs, err := rules.Query(a.Namespace(), rule.QueryOptions{
			Deleted: &defaultDeleted,
		})
		if err != nil {
			return nil, err
		}

		userCount, err := users.Count(a.Namespace(), user.QueryOptions{
			Deleted: &defaultDeleted,
		})
		if err != nil {
			return nil, err
		}

		return &App{
			App: a,
			Counts: AppCounts{
				Connections: uint(connectionCount),
				Comments:    uint(commentCount),
				Devices:     deviceCount,
				Posts:       uint(postCount),
				Rules:       uint(len(rs)),
				Users:       uint(userCount),
			},
		}, nil
	}
}

// AppListFunc returns all Apps.
type AppListFunc func() (app.List, error)

// AppList returns all apps.
func AppList(apps app.Service) AppListFunc {
	return func() (app.List, error) {
		return apps.Query(app.NamespaceDefault, app.QueryOptions{})
	}
}

func generateTokens() (string, string, error) {
	src := rand.NewSource(time.Now().UnixNano())

	tokenHash := md5.New()
	_, err := tokenHash.Write(generate.RandomBytes(src, 32))
	if err != nil {
		return "", "", err
	}
	token := fmt.Sprintf("%x", tokenHash.Sum(nil))

	backendHash := md5.New()
	_, err = backendHash.Write(generate.RandomBytes(src, 12))
	if err != nil {
		return "", "", err
	}

	return token, fmt.Sprintf(
		"%s%s",
		token,
		fmt.Sprintf("%x", backendHash.Sum(nil))[:12],
	), nil
}
