package app

import (
	"fmt"
	"time"

	"github.com/tapglue/api/platform/service"
)

const (
	// NamespaceDefault is the default namespace to isolate top-level data sets.
	NamespaceDefault = "tg"

	fmtNamespace = "app_%d_%d"

	limitProduction = 20000
	limitStaging    = 100
)

// App represents an Org owned data container.
type App struct {
	BackendToken string    `json:"backend_token"`
	Description  string    `json:"description"`
	Enabled      bool      `json:"enabled"`
	ID           uint64    `json:"-"`
	InProduction bool      `json:"in_production"`
	Name         string    `json:"name"`
	OrgID        uint64    `json:"-"`
	PublicID     string    `json:"id"`
	PublicOrgID  string    `json:"account_id"`
	Token        string    `json:"token"`
	URL          string    `json:"url"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Limit returns the desired rate limit for an Application varied by production
// state.
func (a *App) Limit() int64 {
	if a.InProduction {
		return limitProduction
	}

	return limitStaging
}

// Namespace is the identifier used to slice and dice data related to a
// customers app.
func (a *App) Namespace() string {
	return fmt.Sprintf(fmtNamespace, a.OrgID, a.ID)
}

func (a *App) Validate() error {
	return nil
}

// List is an App collection.
type List []*App

// QueryOptions are used to narrow down app queries.
type QueryOptions struct {
	Before        time.Time
	BackendTokens []string
	Enabled       *bool
	IDs           []uint64
	InProduction  *bool
	Limit         int
	OrgIDs        []uint64
	PublicIDs     []string
	Tokens        []string
}

// Service for app interactions.
type Service interface {
	service.Lifecycle

	Put(namespace string, app *App) (*App, error)
	Query(namespace string, opts QueryOptions) (List, error)
}

// ServiceMiddleware is a chainable behaviour modifier for Service.
type ServiceMiddleware func(Service) Service
