package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/net/context"

	"github.com/tapglue/snaas/core"
	"github.com/tapglue/snaas/platform/sns"
	"github.com/tapglue/snaas/service/platform"
)

// PlatformCreate stores the provided platform.
func PlatformCreate(fn core.PlatformCreateFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			currentApp = appFromContext(ctx)
			payload    = payloadPlatform{}
		)

		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		p, err := fn(currentApp, payload.platform, payload.cert, payload.key)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		respondJSON(w, http.StatusCreated, &payloadPlatform{platform: p})
	}
}

type payloadPlatform struct {
	cert, key string
	platform  *platform.Platform
}

func (p *payloadPlatform) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Active    bool      `json:"active"`
		ARN       string    `json:"arn"`
		Deleted   bool      `json:"deleted"`
		Ecosystem int       `json:"ecosystem"`
		ID        string    `json:"id"`
		Name      string    `json:"name"`
		Scheme    string    `json:"scheme"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}{
		Active:    p.platform.Active,
		ARN:       p.platform.ARN,
		Deleted:   p.platform.Deleted,
		Ecosystem: int(p.platform.Ecosystem),
		ID:        strconv.FormatUint(p.platform.ID, 10),
		Name:      p.platform.Name,
		Scheme:    p.platform.Scheme,
		CreatedAt: p.platform.CreatedAt,
		UpdatedAt: p.platform.UpdatedAt,
	})
}

func (p *payloadPlatform) UnmarshalJSON(raw []byte) error {
	f := struct {
		Cert      string       `json:"cert"`
		Ecosystem sns.Platform `json:"ecosystem"`
		Key       string       `json:"key"`
		Name      string       `json:"name"`
		Scheme    string       `json:"scheme"`
	}{}

	if err := json.Unmarshal(raw, &f); err != nil {
		return err
	}

	p.cert = f.Cert
	p.key = f.Key
	p.platform = &platform.Platform{
		Ecosystem: f.Ecosystem,
		Name:      f.Name,
		Scheme:    f.Scheme,
	}

	return nil
}
