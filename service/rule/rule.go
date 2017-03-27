package rule

import (
	"fmt"
	"time"

	"github.com/tapglue/snaas/platform/service"
	"github.com/tapglue/snaas/platform/sns"
	"github.com/tapglue/snaas/service/connection"
	"github.com/tapglue/snaas/service/event"
	"github.com/tapglue/snaas/service/object"
	"github.com/tapglue/snaas/service/reaction"
)

// Type to distinct between different stored criteria.
const (
	TypeConnection Type = iota
	TypeEvent
	TypeObject
	TypeReaction
)

type CriteriaConnection struct {
	New *connection.QueryOptions `json:"new"`
	Old *connection.QueryOptions `json:"old"`
}

func (c *CriteriaConnection) Match(i interface{}) bool {
	s, ok := i.(*connection.StateChange)
	if !ok {
		return false
	}

	if s.New == nil && s.Old == nil {
		return false
	}

	if s.Old == nil {
		return s.New.MatchOpts(c.New)
	}

	return s.New.MatchOpts(c.New) && s.Old.MatchOpts(c.Old)
}

type CriteriaReaction struct {
	New *reaction.QueryOptions `json:"new"`
	Old *reaction.QueryOptions `json:"old"`
}

func (c *CriteriaReaction) Match(i interface{}) bool {
	s, ok := i.(*reaction.StateChange)
	if !ok {
		return false
	}

	if s.New == nil && s.Old == nil {
		return false
	}

	if s.Old == nil {
		return s.New.MatchOpts(c.New)
	}

	return s.New.MatchOpts(c.New) && s.Old.MatchOpts(c.Old)
}

type CriteriaEvent struct {
	New *event.QueryOptions `json:"new"`
	Old *event.QueryOptions `json:"old"`
}

func (c *CriteriaEvent) Match(i interface{}) bool {
	s, ok := i.(*event.StateChange)
	if !ok {
		return false
	}

	if s.New == nil && s.Old == nil {
		return false
	}

	if s.Old == nil {
		return s.New.MatchOpts(c.New)
	}

	return s.New.MatchOpts(c.New) && s.Old.MatchOpts(c.Old)
}

type CriteriaObject struct {
	New *object.QueryOptions `json:"new"`
	Old *object.QueryOptions `json:"old"`
}

func (c *CriteriaObject) Match(i interface{}) bool {
	s, ok := i.(*object.StateChange)
	if !ok {
		return false
	}

	if s.New == nil && s.Old == nil {
		return false
	}

	if s.Old == nil {
		return s.New.MatchOpts(c.New)
	}

	return s.New.MatchOpts(c.New) && s.Old.MatchOpts(c.Old)
}

// List is a Rule collection.
type List []*Rule

// Matcher determines if a given state-change should trigger the Rule.
type Matcher interface {
	Match(c interface{}) bool
}

// Query is a mapping for templated Recipient lookups.
type Query map[string]string

// QueryOptions to narrow-down Rule queries.
type QueryOptions struct {
	Active  *bool
	Deleted *bool
	IDs     []uint64
	Types   []Type
}

// Recipient is an abstract description of how to lookup users and template the
// messaging as well as meta-information.
type Recipient struct {
	Query     Query     `json:"query"`
	Templates Templates `json:"templates"`
	URN       string    `json:"urn"`
}

// Recipients is a Recipient collection.
type Recipients []Recipient

// Rule is a data container to parametrise Pipelines.
type Rule struct {
	Active     bool
	Criteria   Matcher
	Deleted    bool
	Ecosystem  sns.Platform
	ID         uint64
	Name       string
	Recipients Recipients
	Type       Type
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Validate checks for semantic correctness.
func (r *Rule) Validate() error {
	return nil
}

// Service for rule interactions.
type Service interface {
	service.Lifecycle

	Put(namespace string, r *Rule) (*Rule, error)
	Query(namespace string, opts QueryOptions) (List, error)
}

// ServiceMiddleware is a chainable behaviour modifier for Service.
type ServiceMiddleware func(Service) Service

// Templates map languages to template strings.
type Templates map[string]string

// Type indicates for which entity the criterias are encoded in the rule.
type Type uint8

func flakeNamespace(ns string) string {
	return fmt.Sprintf("%s_%s", ns, "rules")
}
