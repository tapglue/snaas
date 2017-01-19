package reaction

import (
	"fmt"
	"sort"
	"time"

	serr "github.com/tapglue/snaas/error"
	"github.com/tapglue/snaas/platform/service"
	"github.com/tapglue/snaas/platform/source"
)

// Supported Reaction types.
const (
	TypeLike Type = iota + 1
	TypeLove
	TypeHaha
	TypeWow
	TypeSad
	TypeAngry
)

type Consumer interface {
	Consume() (*StateChange, error)
}

// List is a collection of Reaction.
type List []*Reaction

func (rs List) Len() int {
	return len(rs)
}

func (rs List) Less(i, j int) bool {
	return rs[i].UpdatedAt.After(rs[j].UpdatedAt)
}

func (rs List) Swap(i, j int) {
	rs[i], rs[j] = rs[j], rs[i]
}

// OwnerIDs returns the list of owner ids for the Reaction collection.
func (rs List) OwnerIDs() []uint64 {
	is := []uint64{}

	for _, r := range rs {
		is = append(is, r.OwnerID)
	}

	return is
}

// Map is a Reaction collection with their id as index.
type Map map[uint64]*Reaction

func (m Map) ToList() List {
	rs := List{}

	for _, r := range m {
		rs = append(rs, r)
	}

	sort.Sort(rs)

	return rs
}

// Producer creates a state change notification.
type Producer interface {
	Propagate(namespace string, old, new *Reaction) (string, error)
}

// QueryOptions to narrow-down queries.
type QueryOptions struct {
	Before    time.Time `json:"-"`
	Deleted   *bool     `json:"deleted,omitempty"`
	IDs       []uint64  `json:"-"`
	Limit     int       `json:"-"`
	ObjectIDs []uint64  `json:"object_ids"`
	OwnerIDs  []uint64  `json:"owner_ids"`
	Types     []Type    `json:"types"`
}

// Reaction is the building block to express interactions on Objects/Posts.
type Reaction struct {
	Deleted   bool
	ID        uint64
	ObjectID  uint64
	OwnerID   uint64
	Type      Type
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Validate checks for semantic correctness.
func (r *Reaction) Validate() error {
	if r.ObjectID == 0 {
		return serr.Wrap(serr.ErrInvalidReaction, "missing object id")
	}

	if r.OwnerID == 0 {
		return serr.Wrap(serr.ErrInvalidReaction, "missing owner id")
	}

	if r.Type < TypeLike || r.Type > TypeAngry {
		return serr.Wrap(serr.ErrInvalidReaction, "unspported type '%d'", r.Type)
	}

	return nil
}

// Service for Reaction interactions.
type Service interface {
	service.Lifecycle

	Count(namespace string, opts QueryOptions) (uint, error)
	Put(namespace string, reaction *Reaction) (*Reaction, error)
	Query(namespace string, opts QueryOptions) (List, error)
}

// ServiceMiddleware is a chainable behaviour modifier for Service.
type ServiceMiddleware func(Service) Service

// Source encapsulates state change notification operations.
type Source interface {
	source.Acker
	Consumer
	Producer
}

// SourceMiddleware is a chainable behaviour modifier for Source.
type SourceMiddleware func(Source) Source

// StateChange transports all information necessary to observe state changes.
type StateChange struct {
	AckID     string
	ID        string
	Namespace string
	New       *Reaction
	Old       *Reaction
	SentAt    time.Time
}

// Type is used to distinct Reactions by type.
type Type uint

func flakeNamespace(ns string) string {
	return fmt.Sprintf("%s_%s", ns, "reactions")
}
