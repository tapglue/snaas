package core

import (
	"github.com/tapglue/snaas/service/app"
	"github.com/tapglue/snaas/service/counter"
)

// CounterGetAllFunc returns the sum of all counter for a coutner name.
type CounterGetAllFunc func(currentApp *app.App, name string) (uint64, error)

// CounterGetAll returns the sum of all counter for a coutner name.
func CounterGetAll(counters counter.Service) CounterGetAllFunc {
	return func(currentApp *app.App, name string) (uint64, error) {
		return counters.CountAll(currentApp.Namespace(), name)
	}
}

// CounterSetFunc sets the counter for the current user and the given counter
// name to the new value.
type CounterSetFunc func(
	currentApp *app.App,
	origin uint64,
	name string,
	value uint64,
) error

// CounterSet sets the counter for the current user and the given counter name
// to the new value.
func CounterSet(counters counter.Service) CounterSetFunc {
	return func(
		currentApp *app.App,
		origin uint64,
		name string,
		value uint64,
	) error {
		return counters.Set(currentApp.Namespace(), name, origin, value)
	}
}
