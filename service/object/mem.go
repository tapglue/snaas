package object

import (
	"math"
	"time"

	"github.com/tapglue/api/platform/flake"
)

type memService struct {
	objects map[string]map[uint64]*Object
}

// MemService returns a memory backed implementation of Service.
func MemService() Service {
	return &memService{
		objects: map[string]map[uint64]*Object{},
	}
}

func (s *memService) Count(ns string, opts QueryOptions) (int, error) {
	if err := s.Setup(ns); err != nil {
		return 0, err
	}

	bucket, ok := s.objects[ns]
	if !ok {
		return 0, ErrNamespaceNotFound
	}

	return len(filterMap(bucket, opts)), nil
}

func (s *memService) Put(ns string, object *Object) (*Object, error) {
	if err := object.Validate(); err != nil {
		return nil, err
	}

	if err := s.Setup(ns); err != nil {
		return nil, err
	}

	bucket, ok := s.objects[ns]
	if !ok {
		return nil, ErrNamespaceNotFound
	}

	if object.ObjectID != 0 {
		keep := false
		for _, o := range bucket {
			if o.ID == object.ObjectID {
				keep = true
			}
		}

		if !keep {
			return nil, ErrMissingReference
		}
	}

	now := time.Now().UTC()

	if object.ID == 0 {
		id, err := flake.NextID(flakeNamespace(ns))
		if err != nil {
			return nil, err
		}

		if object.CreatedAt.IsZero() {
			object.CreatedAt = now
		} else {
			object.CreatedAt = object.CreatedAt.UTC()
		}

		object.ID = id
	} else {
		keep := false

		for _, o := range bucket {
			if o.ID == object.ID {
				keep = true
				object.CreatedAt = o.CreatedAt
			}
		}

		if !keep {
			return nil, ErrNotFound
		}
	}

	object.UpdatedAt = now
	bucket[object.ID] = copy(object)

	return copy(object), nil
}

func (s *memService) Query(ns string, opts QueryOptions) (List, error) {
	if err := s.Setup(ns); err != nil {
		return nil, err
	}

	bucket, ok := s.objects[ns]
	if !ok {
		return nil, ErrNamespaceNotFound
	}

	return filterMap(bucket, opts), nil
}

func (s *memService) Setup(ns string) error {
	if _, ok := s.objects[ns]; !ok {
		s.objects[ns] = map[uint64]*Object{}
	}

	return nil
}

func (s *memService) Teardown(ns string) error {
	if _, ok := s.objects[ns]; ok {
		delete(s.objects, ns)
	}

	return nil
}

func copy(o *Object) *Object {
	old := *o
	return &old
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

func filterMap(om Map, opts QueryOptions) List {
	os := []*Object{}

	for id, object := range om {
		if !opts.Before.IsZero() && object.CreatedAt.UTC().After(opts.Before.UTC()) {
			continue
		}

		if object.Deleted != opts.Deleted {
			continue
		}

		if opts.Owned != nil {
			if object.Owned != *opts.Owned {
				continue
			}
		}

		if !inTypes(object.ExternalID, opts.ExternalIDs) {
			continue
		}

		if opts.ID != nil && id != *opts.ID {
			continue
		}

		if !inIDs(object.OwnerID, opts.OwnerIDs) {
			continue
		}

		if !inIDs(object.ObjectID, opts.ObjectIDs) {
			continue
		}

		if len(opts.Tags) > len(object.Tags) {
			continue
		}

		for _, t := range opts.Tags {
			if !inTypes(t, object.Tags) {
				continue
			}
		}

		if !inTypes(object.Type, opts.Types) {
			continue
		}

		if !inVisibilities(object.Visibility, opts.Visibilities) {
			continue
		}

		os = append(os, object)
	}

	if len(os) == 0 {
		return os
	}

	if opts.Limit > 0 {
		l := math.Min(float64(len(os)), float64(opts.Limit))

		return os[:int(l)]
	}

	return os
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
