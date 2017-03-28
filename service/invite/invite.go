package invite

import (
	"time"

	"github.com/tapglue/snaas/platform/service"
)

const entity = "invite"

// Invite is a loose promise to create a conection if the person assoicated with
// the social id key-value signs up.
type Invite struct {
	Deleted   bool
	ID        uint64
	Key       string
	UserID    uint64
	Value     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// List is a collection of Invite.
type List []*Invite

// QueryOptions to narrow-down Invite queries.
type QueryOptions struct {
	Before  time.Time
	Deleted *bool
	IDs     []uint64
	Keys    []string
	Limit   uint
	UserIDs []uint64
	Values  []string
}

// Service for Invite interactions.
type Service interface {
	service.Lifecycle

	Put(namespace string, i *Invite) (*Invite, error)
	Query(namespace string, opts QueryOptions) (List, error)
}

// ServiceMiddleware is a chainable behaviour modifier for Service.
type ServiceMiddleware func(Service) Service
