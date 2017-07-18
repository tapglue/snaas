package http

import (
	"encoding/json"
	"net/http"

	"golang.org/x/net/context"

	"github.com/tapglue/snaas/core"
)

// CounterGetAll returns the sum of all counter for a coutner name.
func CounterGetAll(fn core.CounterGetAllFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			currentApp = appFromContext(ctx)
		)

		name, err := extractCounterName(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		v, err := fn(currentApp, name)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		respondJSON(w, http.StatusOK, &payloadCounter{Value: v})
	}
}

// CounterSet sets the counter for the current user and the given counter name
// to the new value.
func CounterSet(fn core.CounterSetFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			currentApp  = appFromContext(ctx)
			currentUser = userFromContext(ctx)
			p           = payloadCounter{}
		)

		name, err := extractCounterName(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		err = json.NewDecoder(r.Body).Decode(&p)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		err = fn(currentApp, currentUser.ID, name, p.Value)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		respondJSON(w, http.StatusNoContent, nil)
	}
}

type payloadCounter struct {
	Value uint64 `json:"value"`
}
