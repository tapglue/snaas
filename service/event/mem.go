package event

import (
	"fmt"
	"math"
	"time"

	"github.com/tapglue/snaas/platform/flake"
)

type memService struct {
	events map[string]map[uint64]*Event
}

// MemService returns a memory backed implementation of Service.
func MemService() Service {
	return &memService{
		events: map[string]map[uint64]*Event{},
	}
}

func (s *memService) Count(ns string, opts QueryOptions) (count int, err error) {
	if err := s.Setup(ns); err != nil {
		return 0, err
	}

	return len(filterList(s.events[ns], opts)), nil
}

func (s *memService) Put(ns string, event *Event) (*Event, error) {
	if err := s.Setup(ns); err != nil {
		return nil, err
	}

	var (
		bucket = s.events[ns]
		now    = time.Now().UTC()
	)

	if event.ID == 0 {
		id, err := flake.NextID(flakeNamespace(ns))
		if err != nil {
			return nil, err
		}

		if event.CreatedAt.IsZero() {
			event.CreatedAt = now
		}

		event.CreatedAt = event.CreatedAt.UTC()
		event.ID = id
	} else {
		keep := false

		for _, e := range bucket {
			if e.ID == event.ID {
				keep = true
				event.CreatedAt = e.CreatedAt
			}
		}

		if !keep {
			return nil, fmt.Errorf("event not found")
		}
	}

	event.UpdatedAt = now
	bucket[event.ID] = copy(event)

	return copy(event), nil
}

func (s *memService) Query(ns string, opts QueryOptions) (List, error) {
	if err := s.Setup(ns); err != nil {
		return nil, err
	}

	return filterList(s.events[ns], opts), nil
}

func (s *memService) Setup(ns string) error {
	if _, ok := s.events[ns]; !ok {
		s.events[ns] = map[uint64]*Event{}
	}

	return nil
}

func (s *memService) Teardown(ns string) error {
	if _, ok := s.events[ns]; ok {
		delete(s.events, ns)
	}

	return nil
}

func copy(e *Event) *Event {
	old := *e
	return &old
}

func filterList(em Map, opts QueryOptions) List {
	es := List{}

	for id, event := range em {
		if !opts.Before.IsZero() && event.CreatedAt.UTC().After(opts.Before.UTC()) {
			continue
		}

		if opts.Enabled != nil && event.Enabled != *opts.Enabled {
			continue
		}

		if event.Object == nil && len(opts.ExternalObjectIDs) > 0 {
			continue
		}

		if event.Object == nil && len(opts.ExternalObjectTypes) > 0 {
			continue
		}

		if event.Object != nil && !inTypes(event.Object.ID, opts.ExternalObjectIDs) {
			continue
		}

		if event.Object != nil && !inTypes(event.Object.Type, opts.ExternalObjectTypes) {
			continue
		}

		if !inIDs(id, opts.IDs) {
			continue
		}

		if !inIDs(event.ObjectID, opts.ObjectIDs) {
			continue
		}

		if opts.Owned != nil && event.Owned != *opts.Owned {
			continue
		}

		if event.Target == nil && len(opts.TargetIDs) > 0 {
			continue
		}

		if event.Target != nil && !inTypes(event.Target.ID, opts.TargetIDs) {
			continue
		}

		if event.Target == nil && len(opts.TargetTypes) > 0 {
			continue
		}

		if event.Target != nil && !inTypes(event.Target.Type, opts.TargetTypes) {
			continue
		}

		if !inTypes(event.Type, opts.Types) {
			continue
		}

		if !inIDs(event.UserID, opts.UserIDs) {
			continue
		}

		if !inVisibilities(event.Visibility, opts.Visibilities) {
			continue
		}

		es = append(es, event)
	}

	if len(es) == 0 {
		return es
	}

	if opts.Limit > 0 {
		l := math.Min(float64(len(es)), float64(opts.Limit))

		return es[:int(l)]
	}

	return es
}

func inIDs(id uint64, ids []uint64) bool {
	if len(ids) == 0 {
		return true
	}

	keep := false

	for _, i := range ids {
		if id == i {
			keep = true
			break
		}
	}

	return keep
}

func inTypes(ty string, ts []string) bool {
	if len(ts) == 0 {
		return true
	}

	keep := false

	for _, t := range ts {
		if ty == t {
			keep = true
			break
		}
	}

	return keep
}

func inVisibilities(visibility Visibility, vs []Visibility) bool {
	if len(vs) == 0 {
		return true
	}

	keep := false

	for _, v := range vs {
		if visibility == v {
			keep = true
			break
		}
	}

	return keep
}
