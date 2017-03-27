package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"golang.org/x/net/context"

	"github.com/tapglue/snaas/core"
	"github.com/tapglue/snaas/service/app"
)

// AppCreate creates a new App.
func AppCreate(fn core.AppCreateFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		p := payloadApp{}

		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		a, err := fn(p.Name, p.Description)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		respondJSON(w, http.StatusOK, &payloadApp{app: a})
	}
}

// AppList returns all apps.
func AppList(fn core.AppListFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		as, err := fn()
		if err != nil {
			respondError(w, 0, err)
			return
		}

		respondJSON(w, http.StatusOK, &payloadApps{apps: as})
	}
}

// AppRetrieve returns the app for the requested id.
func AppRetrieve(fn core.AppFetchWithCountsFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		id, err := extractAppID(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		a, err := fn(id)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		respondJSON(w, http.StatusOK, &payloadApp{app: a.App, counts: a.Counts})
	}
}

type payloadApp struct {
	app         *app.App
	counts      core.AppCounts
	Description string `json:"description"`
	Name        string `json:"name"`
}

func (p *payloadApp) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		BackendToken string            `json:"backend_token"`
		Counts       *payloadAppCounts `json:"counts"`
		Description  string            `json:"description"`
		Enabled      bool              `json:"enabled"`
		ID           string            `json:"id"`
		Name         string            `json:"name"`
		Token        string            `json:"token"`
	}{
		BackendToken: p.app.BackendToken,
		Counts:       &payloadAppCounts{counts: p.counts},
		Description:  p.app.Description,
		Enabled:      p.app.Enabled,
		ID:           strconv.FormatUint(p.app.ID, 10),
		Name:         p.app.Name,
		Token:        p.app.Token,
	})
}

type payloadApps struct {
	apps app.List
}

func (p *payloadApps) MarshalJSON() ([]byte, error) {
	as := []*payloadApp{}

	for _, a := range p.apps {
		as = append(as, &payloadApp{app: a})
	}

	return json.Marshal(struct {
		Apps []*payloadApp `json:"apps"`
	}{
		Apps: as,
	})
}

type payloadAppCounts struct {
	counts core.AppCounts
}

func (p *payloadAppCounts) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Comments    uint `json:"comments"`
		Connections uint `json:"connections"`
		Devices     uint `json:"devices"`
		Posts       uint `json:"posts"`
		Rules       uint `json:"rules"`
		Users       uint `json:"users"`
	}{
		Comments:    p.counts.Comments,
		Connections: p.counts.Connections,
		Devices:     p.counts.Devices,
		Posts:       p.counts.Posts,
		Rules:       p.counts.Rules,
		Users:       p.counts.Users,
	})
}
