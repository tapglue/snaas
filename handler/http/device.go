package http

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"golang.org/x/net/context"

	"github.com/tapglue/snaas/core"
	"github.com/tapglue/snaas/service/device"
)

// DeviceDelete removes a user's device.
func DeviceDelete(fn core.DeviceDeleteFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			currentApp = appFromContext(ctx)
			deviceID   = mux.Vars(r)["deviceID"]
			origin     = originFromContext(ctx)
		)

		err := fn(currentApp, origin, deviceID)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		respondJSON(w, http.StatusNoContent, nil)
	}
}

// DeviceUpdate stores the platform and token for a user's device.
func DeviceUpdate(fn core.DeviceUpdateFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			currentApp = appFromContext(ctx)
			deviceID   = mux.Vars(r)["deviceID"]
			origin     = originFromContext(ctx)
			p          = payloadDevice{}
		)

		err := json.NewDecoder(r.Body).Decode(&p)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		err = fn(currentApp, origin, deviceID, p.platform, p.token, p.language)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		respondJSON(w, http.StatusNoContent, nil)
	}
}

type payloadDevice struct {
	language string
	platform device.Platform
	token    string
}

func (p *payloadDevice) UnmarshalJSON(raw []byte) error {
	f := struct {
		Language string          `json:"language"`
		Platform device.Platform `json:"platform"`
		Token    string          `json:"token"`
	}{}

	err := json.Unmarshal(raw, &f)
	if err != nil {
		return err
	}

	p.language = f.Language
	p.platform = f.Platform
	p.token = f.Token

	return nil
}
