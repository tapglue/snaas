package reaction

import (
	"time"

	serr "github.com/tapglue/snaas/error"
	"github.com/tapglue/snaas/platform/flake"
)

type memService struct {
	reactions map[string]Map
}

// MemService returns a memory based Service implementation.
func MemService() Service {
	return &memService{
		reactions: map[string]Map{},
	}
}

func (s *memService) Count(ns string, opts QueryOptions) (uint, error) {
	if err := s.Setup(ns); err != nil {
		return 0, err
	}

	return uint(len(filterList(s.reactions[ns].ToList(), opts))), nil
}

func (s *memService) CountMulti(ns string, opts QueryOptions) (CountsMap, error) {
	if err := s.Setup(ns); err != nil {
		return nil, err
	}

	countsMap := CountsMap{}

	for _, oid := range opts.ObjectIDs {
		counts := Counts{}

		for _, r := range s.reactions[ns] {
			if r.Deleted {
				continue
			}

			if r.ObjectID == oid {
				switch r.Type {
				case TypeAngry:
					counts.Angry++
				case TypeHaha:
					counts.Haha++
				case TypeLike:
					counts.Like++
				case TypeLove:
					counts.Love++
				case TypeSad:
					counts.Sad++
				case TypeWow:
					counts.Wow++
				}
			}
		}

		countsMap[oid] = counts
	}

	return countsMap, nil
}

func (s *memService) Put(ns string, input *Reaction) (*Reaction, error) {
	if err := s.Setup(ns); err != nil {
		return nil, err
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	var (
		bucket = s.reactions[ns]
		now    = time.Now().UTC()
	)

	if input.ID == 0 {
		id, err := flake.NextID(flakeNamespace(ns))
		if err != nil {
			return nil, err
		}

		if input.CreatedAt.IsZero() {
			input.CreatedAt = now
		}

		input.ID = id
	} else {
		keep := false

		for _, input := range bucket {
			if input.ID == input.ID {
				keep = true
				input.CreatedAt = input.CreatedAt
			}
		}

		if !keep {
			return nil, serr.Wrap(serr.ErrReactionNotFound, "%d", input.ID)
		}
	}

	input.UpdatedAt = now
	bucket[input.ID] = copyReaction(input)

	return copyReaction(input), nil
}

func (s *memService) Query(ns string, opts QueryOptions) (List, error) {
	if err := s.Setup(ns); err != nil {
		return nil, err
	}

	rs := filterList(s.reactions[ns].ToList(), opts)

	if opts.Limit > 0 && len(rs) > opts.Limit {
		rs = rs[:opts.Limit]
	}

	return rs, nil
}

func (s *memService) Setup(ns string) error {
	if _, ok := s.reactions[ns]; !ok {
		s.reactions[ns] = Map{}
	}

	return nil
}

func (s *memService) Teardown(ns string) error {
	if _, ok := s.reactions[ns]; ok {
		delete(s.reactions, ns)
	}

	return nil
}

func copyReaction(r *Reaction) *Reaction {
	old := *r
	return &old
}

func filterList(rs List, opts QueryOptions) List {
	fs := List{}

	for _, r := range rs {
		if !opts.Before.IsZero() {
			if r.UpdatedAt.After(opts.Before) || r.UpdatedAt == opts.Before {
				continue
			}
		}

		if opts.Deleted != nil && r.Deleted != *opts.Deleted {
			continue
		}

		if !inIDs(r.ID, opts.IDs) {
			continue
		}

		if !inIDs(r.ObjectID, opts.ObjectIDs) {
			continue
		}

		if !inIDs(r.OwnerID, opts.OwnerIDs) {
			continue
		}

		if !inTypes(r.Type, opts.Types) {
			continue
		}

		fs = append(fs, r)
	}

	return fs
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

func inTypes(t Type, ts []Type) bool {
	if len(ts) == 0 {
		return true
	}

	keep := false

	for _, i := range ts {
		if t == i {
			keep = true
			break
		}
	}

	return keep
}
