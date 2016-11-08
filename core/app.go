package core

import (
	"github.com/tapglue/snaas/service/app"
)

// AppFetchFunc returns the App for the given id.
type AppFetchFunc func(id uint64) (*app.App, error)

// AppFetchFunc returns the App for the given id.
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
