package http

import (
	"encoding/json"
	"net/http"

	"golang.org/x/net/context"

	"github.com/tapglue/snaas/core"
)

// InviteCreate stores the key and value for a users invite.
func InviteCreate(fn core.InviteCreateFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			currentApp = appFromContext(ctx)
			origin     = originFromContext(ctx)
			p          = payloadInvite{}
		)

		err := json.NewDecoder(r.Body).Decode(&p)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		err = fn(currentApp, origin, p.Key, p.Value)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		respondJSON(w, http.StatusNoContent, nil)
	}
}

type payloadInvite struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
