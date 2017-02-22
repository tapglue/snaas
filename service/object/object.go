package object

import (
	"fmt"
	"time"

	"github.com/asaskevich/govalidator"
	"golang.org/x/text/language"

	"github.com/tapglue/snaas/platform/service"
	"github.com/tapglue/snaas/platform/source"
)

// Attachment variants available for Objects.
const (
	AttachmentTypeText = "text"
	AttachmentTypeURL  = "url"
)

// DefaultLanguage is used when no lang is provided for object content.
const DefaultLanguage = "en"

// State variants available for Objects.
const (
	StatePending State = iota
	StateConfirmed
	StateDeclined
)

// Visibility variants available for Objects.
const (
	VisibilityPrivate Visibility = (iota + 1) * 10
	VisibilityConnection
	VisibilityPublic
	VisibilityGlobal
)

// Attachment is typed media which belongs to an Object.
type Attachment struct {
	Contents Contents `json:"contents"`
	Name     string   `json:"name"`
	Type     string   `json:"type"`
}

// Validate returns an error if a Attachment constraint is not full-filled.
func (a Attachment) Validate() error {
	if a.Name == "" {
		return wrapError(ErrInvalidAttachment, "name must be set")
	}

	if a.Type == "" ||
		(a.Type != AttachmentTypeText && a.Type != AttachmentTypeURL) {
		return wrapError(ErrInvalidAttachment, "unsupported type '%s'", a.Type)
	}

	if a.Contents == nil || len(a.Contents) == 0 {
		return wrapError(ErrInvalidAttachment, "contents can't be empty")
	}

	for tag, content := range a.Contents {
		_, err := language.Parse(tag)
		if err != nil {
			return wrapError(
				ErrInvalidAttachment,
				"invalid language tag '%s'",
				tag,
			)
		}

		if content == "" {
			return wrapError(ErrInvalidAttachment, "content missing for '%s'", tag)
		}

		if a.Type == AttachmentTypeURL && !govalidator.IsURL(content) {
			return wrapError(ErrInvalidAttachment, "invalid url for '%s'", tag)
		}
	}

	return nil
}

// TextAttachment returns an Attachment of type Text.
func TextAttachment(name string, contents Contents) Attachment {
	return Attachment{
		Contents: contents,
		Name:     name,
		Type:     AttachmentTypeText,
	}
}

// URLAttachment returns an Attachment of type URL.
func URLAttachment(name string, contents Contents) Attachment {
	return Attachment{
		Contents: contents,
		Name:     name,
		Type:     AttachmentTypeURL,
	}
}

// Consumer observes state changes.
type Consumer interface {
	Consume() (*StateChange, error)
}

// Contents is the mapping of content to locale.
type Contents map[string]string

// Validate performs semantic checks on the localisation fields.
func (c Contents) Validate() error {
	return nil
}

// List is an Object collection.
type List []*Object

func (l List) Len() int {
	return len(l)
}

func (l List) Less(i, j int) bool {
	return l[i].CreatedAt.After(l[j].CreatedAt)
}

func (l List) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

// OwnerIDs returns all user ids of the associated object owners.
func (l List) OwnerIDs() []uint64 {
	ids := []uint64{}

	for _, o := range l {
		ids = append(ids, o.OwnerID)
	}

	return ids
}

// Map is an Object collection indexed by id.
type Map map[uint64]*Object

// Object is a generic building block to express different domains like Posts,
// Albums with their dependend objects.
type Object struct {
	Attachments  []Attachment  `json:"attachments"`
	CreatedAt    time.Time     `json:"created_at"`
	Deleted      bool          `json:"deleted"`
	ExternalID   string        `json:"external_id"`
	ID           uint64        `json:"id"`
	Latitude     float64       `json:"latitude"`
	Location     string        `json:"location"`
	Longitude    float64       `json:"longitude"`
	ObjectID     uint64        `json:"object_id"`
	Owned        bool          `json:"owned"`
	OwnerID      uint64        `json:"owner_id"`
	Private      *Private      `json:"private,omitempty"`
	Restrictions *Restrictions `json:"restrictions,omitempty"`
	Tags         []string      `json:"tags"`
	Type         string        `json:"type"`
	UpdatedAt    time.Time     `json:"updated_at"`
	Visibility   Visibility    `json:"visibility"`
}

// MatchOpts indicates if the Object matches the given QueryOptions.
func (o *Object) MatchOpts(opts *QueryOptions) bool {
	if opts == nil {
		return true
	}

	if o.Deleted != opts.Deleted {
		return false
	}

	if opts.Owned != nil && o.Owned != *opts.Owned {
		return false
	}

	if len(opts.Tags) > 0 && len(o.Tags) == 0 {
		return false
	}

	if len(opts.Tags) > 0 {
		for _, t := range opts.Tags {
			discard := true

			for _, tag := range o.Tags {
				if tag == t {
					discard = false
				}
			}

			if discard {
				return false
			}
		}
	}

	if len(opts.Types) > 0 {
		discard := true

		for _, t := range opts.Types {
			if o.Type == t {
				discard = false
			}
		}

		if discard {
			return false
		}
	}

	return true
}

// Validate returns an error if a constraint on the Object is not full-filled.
func (o *Object) Validate() error {
	if len(o.Attachments) > 5 {
		return wrapError(ErrInvalidObject, "too many attachments")
	}

	for _, a := range o.Attachments {
		if err := a.Validate(); err != nil {
			return err
		}
	}

	if o.OwnerID == 0 {
		return wrapError(ErrInvalidObject, "missing owner")
	}

	states := []State{StatePending, StateConfirmed, StateDeclined}

	if o.Private != nil && !inStates(o.Private.State, states) {
		return wrapError(
			ErrInvalidObject,
			"unsupported state (%d)",
			o.Private.State,
		)
	}

	if len(o.Tags) > 25 {
		return wrapError(ErrInvalidObject, "too many tags")
	}

	if o.Type == "" {
		return wrapError(ErrInvalidObject, "missing type")

	}

	vs := []Visibility{
		VisibilityPrivate,
		VisibilityConnection,
		VisibilityPublic,
		VisibilityGlobal,
	}

	if !inVisibilities(o.Visibility, vs) {
		return wrapError(ErrInvalidObject, "unsupported visibility")
	}

	return nil
}

// Private is the bucket for protected fields on an Object.
type Private struct {
	State   State `json:"state"`
	Visible bool  `json:"visible"`
}

// Producer creates a state change notification.
type Producer interface {
	Propagate(namespace string, old, new *Object) (string, error)
}

// QueryOptions are passed to narrow down query for objects.
type QueryOptions struct {
	After        time.Time    `json:"-"`
	Before       time.Time    `json:"-"`
	Deleted      bool         `json:"deleted,omitempty"`
	ExternalIDs  []string     `json:"-"`
	ID           *uint64      `json:"id,omitempty"`
	Limit        int          `json:"-"`
	ObjectIDs    []uint64     `json:"object_ids,omitempty"`
	OwnerIDs     []uint64     `json:"owner_ids,omitempty"`
	Owned        *bool        `json:"owned,omitempty"`
	Tags         []string     `json:"tags,omitempty"`
	Types        []string     `json:"types,omitempty"`
	Visibilities []Visibility `json:"visibilities,omitempty"`
}

// Restrictions is the composite to regulate common interactions on Posts.
type Restrictions struct {
	Comment bool `json:"comment"`
	Like    bool `json:"like"`
	Report  bool `json:"report"`
}

// Service for object interactions.
type Service interface {
	service.Lifecycle

	Count(namespace string, opts QueryOptions) (int, error)
	Put(namespace string, object *Object) (*Object, error)
	Query(namespace string, opts QueryOptions) (List, error)
}

// ServiceMiddleware is a chainable behaviour modifier for Service.
type ServiceMiddleware func(Service) Service

// StateChange transports all information necessary to observe state change.
type StateChange struct {
	AckID     string
	ID        string
	Namespace string
	New       *Object
	Old       *Object
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

// State reflects the progress of an object through a review process.
type State uint8

// Visibility determines the visibility of Objects when consumed.
type Visibility uint8

func flakeNamespace(ns string) string {
	return fmt.Sprintf("%s_%s", ns, "objects")
}

func inStates(c State, ss []State) bool {
	if len(ss) == 0 {
		return true
	}

	for _, s := range ss {
		if c == s {
			return true
		}
	}

	return false
}

func inVisibilities(c Visibility, vs []Visibility) bool {
	if len(vs) == 0 {
		return true
	}

	for _, v := range vs {
		if c == v {
			return true
		}
	}

	return false
}
