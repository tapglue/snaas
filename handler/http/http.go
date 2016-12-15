package http

import (
	"encoding/json"
	"net/http"

	"github.com/garyburd/redigo/redis"
	"github.com/jmoiron/sqlx"
	"golang.org/x/net/context"

	"github.com/tapglue/snaas/core"
	serr "github.com/tapglue/snaas/error"
)

const pgHealthcheck = `SELECT 1`

// Handler is the gateway specific http.HandlerFunc expecting a context.Context.
type Handler func(context.Context, http.ResponseWriter, *http.Request)

// Middleware can be used to chain Handlers with different responsibilities.
type Middleware func(Handler) Handler

// Chain takes a varidatic number of Middlewares and returns a combined
// Middleware.
func Chain(ms ...Middleware) Middleware {
	return func(handler Handler) Handler {
		for i := len(ms) - 1; i >= 0; i-- {
			handler = ms[i](handler)
		}

		return handler
	}
}

// Wrap takes a Middleware and Handler and returns an http.HandlerFunc.
func Wrap(
	middleware Middleware,
	handler Handler,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		middleware(handler)(context.Background(), w, r)
	}
}

// Health checks for liveliness of backing servicesa and responds with status.
func Health(pg *sqlx.DB, rClient *redis.Pool) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		res := struct {
			Healthy  bool            `json:"healthy"`
			Services map[string]bool `json:"services"`
		}{
			Healthy: true,
			Services: map[string]bool{
				"postgres": true,
				"redis":    true,
			},
		}

		if _, err := pg.Exec(pgHealthcheck); err != nil {
			res.Healthy = false
			res.Services["postgres"] = false

			respondJSON(w, 500, &res)
			return
		}

		conn := rClient.Get()
		if err := conn.Err(); err != nil {
			res.Healthy = false
			res.Services["redis"] = false

			respondJSON(w, 500, &res)
			return
		}
		defer conn.Close()

		respondJSON(w, http.StatusOK, &res)
	}
}

type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func createOrigin(deviceID, tokenType string, userID uint64) core.Origin {
	integration := core.IntegrationApplication

	if tokenType == tokenBackend {
		integration = core.IntegrationBackend
	}

	return core.Origin{
		DeviceID:    deviceID,
		Integration: integration,
		UserID:      userID,
	}
}

func respondError(w http.ResponseWriter, code int, err error) {
	statusCode := http.StatusInternalServerError

	e := unwrapError(err)

	switch e {
	case ErrBadRequest:
		statusCode = http.StatusBadRequest
	case ErrLimitExceeded:
		statusCode = 429
	case ErrUnauthorized:
		statusCode = http.StatusUnauthorized
	case core.ErrInvalidEntity:
		statusCode = http.StatusBadRequest
	case core.ErrNotFound:
		code = http.StatusNotFound
		statusCode = http.StatusNotFound
	case core.ErrUnauthorized:
		statusCode = http.StatusUnauthorized
	case serr.ErrUserExists:
		code = 4001
		statusCode = http.StatusUnauthorized
	}

	respondJSON(w, statusCode, struct {
		Errors []apiError `json:"errors"`
	}{
		Errors: []apiError{
			{Code: code, Message: err.Error()},
		},
	})
}

func respondJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(payload)
}
