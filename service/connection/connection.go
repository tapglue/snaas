package connection

import (
	"time"

	"github.com/tapglue/api/platform/service"
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

// Acker permantly removes the workload from the Source.
type Acker interface {
	Ack(id string) error
}

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
	After   time.Time
	Before  time.Time
	Enabled *bool
	FromIDs []uint64
	Limit   int
	States  []State
	ToIDs   []uint64
	Types   []Type
}

// Service for connection interactions.
type Service interface {
	service.Lifecycle

	Count(namespace string, opts QueryOptions) (int, error)
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
	Acker
	Consumer
	Producer
}

// SourceMiddleware is a chainable behaviour modifier for Source.
type SourceMiddleware func(Source) Source

// Type of a user relation.
type Type string
