package connection

import (
	"time"

	"github.com/tapglue/snaas/platform/service"
	"github.com/tapglue/snaas/platform/source"
)

// Supported states for connections.
const (
	StateConfirmed State = "confirmed"
	StatePending   State = "pending"
	StateRejected  State = "rejected"
)

// Supported types for connections.
const (
	TypeFollow Type = "follow"
	TypeFriend Type = "friend"
)

// Connection represents a relation between two users.
type Connection struct {
	Enabled   bool      `json:"enabled"`
	FromID    uint64    `json:"user_from_id"`
	State     State     `json:"state"`
	ToID      uint64    `json:"user_to_id"`
	Type      Type      `json:"type"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Consumer observes state changes.
type Consumer interface {
	Consume() (*StateChange, error)
}

// MatchOpts indicates if the Connection matches the given QueryOptions.
func (c *Connection) MatchOpts(opts *QueryOptions) bool {
	if opts == nil {
		return true
	}

	if opts.Enabled != nil && c.Enabled != *opts.Enabled {
		return false
	}

	if len(opts.States) > 0 {
		discard := true

		for _, s := range opts.States {
			if c.State == s {
				discard = false
			}
		}

		if discard {
			return false
		}
	}

	if len(opts.Types) > 0 {
		discard := true

		for _, t := range opts.Types {
			if c.Type == t {
				discard = false
			}
		}

		if discard {
			return false
		}
	}

	return true
}

// Validate performs checks on the Connection values for completeness and
// correctness.
func (c Connection) Validate() error {
	if c.FromID == 0 {
		return wrapError(ErrInvalidConnection, "from id not set")
	}

	if c.ToID == 0 {
		return wrapError(ErrInvalidConnection, "to id not set")
	}

	switch c.State {
	case StateConfirmed, StatePending, StateRejected:
		// valid
	default:
		return wrapError(ErrInvalidConnection, "invalid state")
	}

	switch c.Type {
	case TypeFollow, TypeFriend:
		// valid
	default:
		return wrapError(ErrInvalidConnection, "invalid type")
	}

	return nil
}

// List is a collection of Connections.
type List []*Connection

// FromIDs returns the extracted FromID of all connections as list.
func (l List) FromIDs() []uint64 {
	ids := []uint64{}

	for _, c := range l {
		ids = append(ids, c.FromID)
	}

	return ids
}

func (l List) Len() int {
	return len(l)
}

func (l List) Less(i, j int) bool {
	return l[i].UpdatedAt.After(l[j].UpdatedAt)
}

func (l List) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

// OtherIDs returns the user ids not being the origin.
func (l List) OtherIDs(origin uint64) []uint64 {
	is := []uint64{}

	for _, c := range l {
		if c.FromID == origin {
			is = append(is, c.ToID)
		} else {
			is = append(is, c.FromID)
		}
	}

	return is
}

// ToIDs returns the extracted ToID of all connections as list.
func (l List) ToIDs() []uint64 {
	ids := []uint64{}

	for _, c := range l {
		ids = append(ids, c.ToID)
	}

	return ids
}

// Producer creates state change notifications.
type Producer interface {
	Propagate(namespace string, old, new *Connection) (string, error)
}

// QueryOptions are used to narrow down Connection queries.
type QueryOptions struct {
	After   time.Time `json:"-"`
	Before  time.Time `json:"-"`
	Enabled *bool     `json:"enabled,omitempty"`
	FromIDs []uint64  `json:"from_ids,omitempty"`
	Limit   int       `json:"-"`
	States  []State   `json:"states,omitempty"`
	ToIDs   []uint64  `json:"to_ids,omitempty"`
	Types   []Type    `json:"types,omitempty"`
}

// Service for connection interactions.
type Service interface {
	service.Lifecycle

	Count(namespace string, opts QueryOptions) (int, error)
	Friends(namespace string, origin uint64) (List, error)
	Put(namespace string, connection *Connection) (*Connection, error)
	Query(namespace string, opts QueryOptions) (List, error)
}

// ServiceMiddleware is a chainable behaviour modifier for Service.
type ServiceMiddleware func(Service) Service

// State of a connection request.
type State string

// StateChange transports all information necessary to observe state changes.
type StateChange struct {
	AckID     string
	ID        string
	Namespace string
	New       *Connection
	Old       *Connection
	SentAt    time.Time
}

// Source encapsulates state change notification operations.
type Source interface {
	source.Acker
	Consumer
	Producer
}

// SourceMiddleware is a chainable behaviour modifier for Source.
type SourceMiddleware func(Source) Source

// Type of a user relation.
type Type string
