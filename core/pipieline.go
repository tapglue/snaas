package core

import (
	"bytes"
	"encoding/json"
	"text/template"
	"time"

	serr "github.com/tapglue/snaas/error"
	"github.com/tapglue/snaas/service/app"
	"github.com/tapglue/snaas/service/connection"
	"github.com/tapglue/snaas/service/event"
	"github.com/tapglue/snaas/service/object"
	"github.com/tapglue/snaas/service/reaction"
	"github.com/tapglue/snaas/service/rule"
	"github.com/tapglue/snaas/service/user"
)

const (
	queryCondOwnerFriends = "ownerFriends"
	queryCondObjectOwner  = "objectOwner"
	queryCondOwner        = "owner"
	queryCondParentOwner  = "parentOwner"
	queryCondUserFrom     = "userFrom"
	queryCondUserTo       = "userTo"
)

// Message is the envelope which holds the templated message produced by a
// Pipeline together with the recipient and the URN to deliver with it.
type Message struct {
	Messages  map[string]string
	Recipient uint64
	URN       string
}

// Messages is a Message collection
type Messages []*Message

// PipelineConnectionFunc constructs a Pipeline that by applying the provided
// rules outputs messages.
type PipelineConnectionFunc func(
	*app.App,
	*connection.StateChange,
	...*rule.Rule,
) (Messages, error)

// PipelineConnection constructs a Pipeline that by applying the provided
// rules outputs Messages.
func PipelineConnection(users user.Service) PipelineConnectionFunc {
	return func(
		currentApp *app.App,
		change *connection.StateChange,
		rules ...*rule.Rule,
	) (Messages, error) {
		var (
			ms = Messages{}
			c  = change.New

			context  *contextConnection
			from, to *user.User
		)

		if change.New == nil {
			return Messages{}, nil
		}

		from, err := UserFetch(users)(currentApp, c.FromID)
		if err != nil {
			return nil, err
		}

		to, err = UserFetch(users)(currentApp, c.ToID)
		if err != nil {
			return nil, err
		}

		context = &contextConnection{
			Conenction: c,
			From:       from,
			To:         to,
		}

		for _, currentRule := range rules {
			if !currentRule.Criteria.Match(change) {
				continue
			}

			for _, recipient := range currentRule.Recipients {
				cs, err := recipientsConnection()(currentApp, context, recipient.Query)
				if err != nil {
					return nil, err
				}

				for _, c := range cs {
					msg, err := compileMessage(context, recipient, c)
					if err != nil {
						return nil, err
					}

					ms = append(ms, msg)
				}
			}
		}

		return ms, nil
	}
}

func compileMessage(
	context interface{},
	recipient rule.Recipient,
	target *user.User,
) (*Message, error) {
	urn, err := compileTemplate(context, recipient.URN)
	if err != nil {
		return nil, err
	}

	msgs := map[string]string{}

	for lang, tmpl := range recipient.Templates {
		msg, err := compileTemplate(context, tmpl)
		if err != nil {
			return nil, err
		}

		msgs[lang] = msg
	}

	return &Message{Messages: msgs, Recipient: target.ID, URN: urn}, nil
}

// PipelineEventFunc constructs a Pipeline that by applying the provided rules
// outputs Messages.
type PipelineEventFunc func(
	*app.App,
	*event.StateChange,
	...*rule.Rule,
) (Messages, error)

// PipelineEvent constructs a Pipeline that by applying the provided rules
// outputs Messages.
func PipelineEvent(
	objects object.Service,
	users user.Service,
) PipelineEventFunc {
	return func(
		currentApp *app.App,
		change *event.StateChange,
		rules ...*rule.Rule,
	) (Messages, error) {
		var (
			ms = Messages{}
			e  = change.New

			context     *contextEvent
			owner       *user.User
			parent      *object.Object
			parentOwner *user.User
		)

		owner, err := UserFetch(users)(currentApp, e.UserID)
		if err != nil {
			return nil, err
		}

		if e.ObjectID != 0 {
			parent, err = objectFetch(objects)(currentApp, e.ObjectID)
			if err != nil {
				return nil, err
			}

			parentOwner, err = UserFetch(users)(currentApp, parent.OwnerID)
			if err != nil {
				return nil, err
			}
		}

		context = &contextEvent{
			Event:       e,
			Owner:       owner,
			Parent:      parent,
			ParentOwner: parentOwner,
		}

		for _, currentRule := range rules {
			if !currentRule.Criteria.Match(change) {
				continue
			}

			for _, recipient := range currentRule.Recipients {
				rs, err := recipientsEvent()(currentApp, context, recipient.Query)
				if err != nil {
					return nil, err
				}

				for _, r := range rs {
					msg, err := compileMessage(context, recipient, r)
					if err != nil {
						return nil, err
					}

					ms = append(ms, msg)
				}
			}
		}

		return ms, nil
	}
}

// PipelineObjectFunc constructs a Pipeline that by appplying the provided
// rules outputs Messages.
type PipelineObjectFunc func(
	*app.App,
	*object.StateChange,
	...*rule.Rule,
) (Messages, error)

// PipelineObject constructs a Pipeline that by appplying the provided rules
// outputs Messages.
func PipelineObject(
	connections connection.Service,
	objects object.Service,
	users user.Service,
) PipelineObjectFunc {
	return func(
		currentApp *app.App,
		change *object.StateChange,
		rules ...*rule.Rule,
	) (Messages, error) {
		var (
			ms = Messages{}
			o  = change.New

			context     *contextObject
			parent      *object.Object
			parentOwner *user.User
		)

		if change.New == nil {
			return Messages{}, nil
		}

		owner, err := UserFetch(users)(currentApp, change.New.OwnerID)
		if err != nil {
			return nil, err
		}

		if o.ObjectID != 0 {
			parent, err = objectFetch(objects)(currentApp, o.ObjectID)
			if err != nil {
				return nil, err
			}

			parentOwner, err = UserFetch(users)(currentApp, parent.OwnerID)
			if err != nil {
				return nil, err
			}
		}

		am := map[string]object.Contents{}

		for _, a := range change.New.Attachments {
			am[a.Name] = a.Contents
		}

		context = &contextObject{
			Attachments: am,
			Object:      change.New,
			Owner:       owner,
			Parent:      parent,
			ParentOwner: parentOwner,
		}

		for _, currentRule := range rules {
			if !currentRule.Criteria.Match(change) {
				continue
			}

			for _, recipient := range currentRule.Recipients {
				rs, err := recipientsObject(
					connections,
					objects,
					users,
				)(currentApp, context, recipient.Query)
				if err != nil {
					return nil, err
				}

				for _, r := range rs {
					msg, err := compileMessage(context, recipient, r)
					if err != nil {
						return nil, err
					}

					ms = append(ms, msg)
				}
			}
		}

		return ms, nil
	}
}

// PipelineReactionFunc constructs a Pipeline that by applying the provided
// rules outputs Messages.
type PipelineReactionFunc func(
	*app.App,
	*reaction.StateChange,
	...*rule.Rule,
) (Messages, error)

// PipelineReaction constructs a Pipeline that by applying the provided rules
// outputs Messages.
func PipelineReaction(
	objects object.Service,
	users user.Service,
) PipelineReactionFunc {
	return func(
		currentApp *app.App,
		change *reaction.StateChange,
		rules ...*rule.Rule,
	) (Messages, error) {
		var (
			ms = Messages{}
			r  = change.New

			context     *contextReaction
			owner       *user.User
			parent      *object.Object
			parentOwner *user.User
		)

		owner, err := UserFetch(users)(currentApp, r.OwnerID)
		if err != nil {
			return nil, err
		}

		if r.ObjectID != 0 {
			parent, err = objectFetch(objects)(currentApp, r.ObjectID)
			if err != nil {
				return nil, err
			}

			parentOwner, err = UserFetch(users)(currentApp, parent.OwnerID)
			if err != nil {
				return nil, err
			}
		}

		context = &contextReaction{
			Owner:       owner,
			Parent:      parent,
			ParentOwner: parentOwner,
			Reaction:    r,
		}

		for _, currentRule := range rules {
			if !currentRule.Criteria.Match(change) {
				continue
			}

			for _, recipient := range currentRule.Recipients {
				rs, err := recipientsReaction()(currentApp, context, recipient.Query)
				if err != nil {
					return nil, err
				}

				for _, r := range rs {
					msg, err := compileMessage(context, recipient, r)
					if err != nil {
						return nil, err
					}

					ms = append(ms, msg)
				}
			}
		}

		return ms, nil
	}
}

type contextConnection struct {
	Conenction *connection.Connection
	From       *user.User
	To         *user.User
}

type contextEvent struct {
	Event       *event.Event
	Owner       *user.User
	Parent      *object.Object
	ParentOwner *user.User
}

type contextObject struct {
	Attachments map[string]object.Contents
	Object      *object.Object
	Owner       *user.User
	Parent      *object.Object
	ParentOwner *user.User
}

type contextReaction struct {
	Owner       *user.User
	Parent      *object.Object
	ParentOwner *user.User
	Reaction    *reaction.Reaction
}

func compileTemplate(context interface{}, t string) (string, error) {
	tmpl, err := template.New("message").Parse(t)
	if err != nil {
		return "", err
	}

	buf := bytes.NewBuffer([]byte{})

	err = tmpl.Execute(buf, context)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func filterIDs(ids []uint64, fs ...uint64) []uint64 {
	var (
		is   = []uint64{}
		seen = map[uint64]struct{}{}
	)

	for _, id := range fs {
		seen[id] = struct{}{}
	}

	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}

		is = append(is, id)
	}

	return is
}

type objectFetchFunc func(*app.App, uint64) (*object.Object, error)

func objectFetch(objects object.Service) objectFetchFunc {
	return func(currentApp *app.App, id uint64) (*object.Object, error) {
		os, err := objects.Query(currentApp.Namespace(), object.QueryOptions{
			ID: &id,
		})
		if err != nil {
			return nil, err
		}

		if len(os) != 1 {
			return nil, serr.Wrap(serr.ErrNotFound, "object missign for '%d'", id)
		}

		return os[0], nil
	}
}

func ownerIDsFetch(
	objects object.Service,
	ns string,
	opts object.QueryOptions,
) ([]uint64, error) {
	opts.Before = time.Now()

	os, err := objects.Query(ns, opts)
	if err != nil {
		return nil, err
	}

	return os.OwnerIDs(), nil
}

func queryOptsFromTemplate(context *contextObject, t string) (object.QueryOptions, error) {
	opts := object.QueryOptions{}

	tmpl, err := template.New("onwerIDs").Parse(t)
	if err != nil {
		return opts, err
	}

	buf := bytes.NewBuffer([]byte{})

	err = tmpl.Execute(buf, context)
	if err != nil {
		return opts, err
	}

	err = json.Unmarshal(buf.Bytes(), &opts)
	if err != nil {
		return opts, err
	}

	return opts, nil
}

type recipientsConnectionFunc func(
	*app.App,
	*contextConnection,
	rule.Query,
) (user.List, error)

func recipientsConnection() recipientsConnectionFunc {
	return func(
		currentApp *app.App,
		context *contextConnection,
		q rule.Query,
	) (user.List, error) {
		us := user.List{}

		for condType := range q {
			switch condType {
			case queryCondUserFrom:
				us = append(us, context.From)
			case queryCondUserTo:
				us = append(us, context.To)
			}
		}

		return us, nil
	}
}

type recipientsEventFunc func(
	*app.App,
	*contextEvent,
	rule.Query,
) (user.List, error)

func recipientsEvent() recipientsEventFunc {
	return func(
		currentApp *app.App,
		context *contextEvent,
		q rule.Query,
	) (user.List, error) {
		us := user.List{}

		for condType := range q {
			switch condType {
			case queryCondParentOwner:
				if context.Owner.ID != context.ParentOwner.ID {
					us = append(us, context.ParentOwner)
				}
			}
		}

		return us, nil
	}
}

type recipientsObjectFunc func(*app.App, *contextObject, rule.Query) (user.List, error)

func recipientsObject(
	connections connection.Service,
	objects object.Service,
	users user.Service,
) recipientsObjectFunc {
	return func(
		currentApp *app.App,
		context *contextObject,
		q rule.Query,
	) (user.List, error) {
		ids := []uint64{}

		for condType, condTemplate := range q {
			switch condType {
			case queryCondObjectOwner:
				opts, err := queryOptsFromTemplate(context, condTemplate)
				if err != nil {
					return nil, err
				}

				ownerIDs, err := ownerIDsFetch(objects, currentApp.Namespace(), opts)
				if err != nil {
					return nil, err
				}

				oIDs := filterIDs(ownerIDs, context.Owner.ID, context.ParentOwner.ID)

				ids = append(ids, oIDs...)
			case queryCondOwnerFriends:
				friendIDs, err := ConnectionFriendIDs(connections)(currentApp, context.Owner.ID)
				if err != nil {
					return nil, err
				}

				ids = append(ids, friendIDs...)
			case queryCondOwner:
				ids = append(ids, context.Owner.ID)
			case queryCondParentOwner:
				if context.Owner.ID != context.ParentOwner.ID {
					ids = append(ids, context.ParentOwner.ID)
				}
			}
		}

		us, err := user.ListFromIDs(users, currentApp.Namespace(), ids...)
		if err != nil {
			return nil, err
		}

		return us, nil
	}
}

type recipientsReactionFunc func(
	*app.App,
	*contextReaction,
	rule.Query,
) (user.List, error)

func recipientsReaction() recipientsReactionFunc {
	return func(
		currentApp *app.App,
		context *contextReaction,
		q rule.Query,
	) (user.List, error) {
		us := user.List{}

		for condType := range q {
			switch condType {
			case queryCondParentOwner:
				if context.Owner.ID != context.ParentOwner.ID {
					us = append(us, context.ParentOwner)
				}
			}
		}

		return us, nil
	}
}
