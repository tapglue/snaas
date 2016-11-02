package connection

import (
	"fmt"
	"math"
	"time"
)

type memService struct {
	cons map[string]map[string]*Connection
}

// MemService returns a memory backed implementation of Service.
func MemService() Service {
	return &memService{
		cons: map[string]map[string]*Connection{},
	}
}

func (s *memService) Count(ns string, opts QueryOptions) (int, error) {
	if err := s.Setup(ns); err != nil {
		return -1, err
	}

	return len(filterMap(s.cons[ns], opts)), nil
}

func (s *memService) Put(ns string, con *Connection) (*Connection, error) {
	if err := s.Setup(ns); err != nil {
		return nil, err
	}

	if err := con.Validate(); err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	if con.CreatedAt.IsZero() {
		con.CreatedAt = now
	}

	con.CreatedAt = con.CreatedAt.UTC()

	stored, ok := s.cons[ns][stringKey(con)]
	if ok {
		con.CreatedAt = stored.CreatedAt
	}

	con.UpdatedAt = now

	s.cons[ns][stringKey(con)] = con

	return con, nil
}

func (s *memService) Query(ns string, opts QueryOptions) (List, error) {
	if err := s.Setup(ns); err != nil {
		return nil, err
	}

	return filterMap(s.cons[ns], opts), nil
}

func (s *memService) Setup(ns string) error {
	_, ok := s.cons[ns]
	if ok {
		return nil
	}

	s.cons[ns] = map[string]*Connection{}

	return nil
}

func (s *memService) Teardown(ns string) error {
	return fmt.Errorf("not implemented")
}

func filterMap(cm map[string]*Connection, opts QueryOptions) List {
	cs := List{}

	for _, con := range cm {
		if !opts.Before.IsZero() && con.CreatedAt.UTC().After(opts.Before.UTC()) {
			continue
		}

		if opts.Enabled != nil && con.Enabled != *opts.Enabled {
			continue
		}

		if !inIDs(con.FromID, opts.FromIDs) {
			continue
		}

		if !inStates(con.State, opts.States) {
			continue
		}

		if !inIDs(con.ToID, opts.ToIDs) {
			continue
		}

		if !inTypes(con.Type, opts.Types) {
			continue
		}

		cs = append(cs, con)
	}

	if len(cs) == 0 {
		return cs
	}

	if opts.Limit > 0 {
		l := math.Min(float64(len(cs)), float64(opts.Limit))

		return cs[:int(l)]
	}

	return cs
}

func inIDs(id uint64, ids []uint64) bool {
	if len(ids) == 0 {
		return true
	}

	keep := false

	for _, i := range ids {
		if i == id {
			keep = true
			break
		}
	}

	return keep
}

func inStates(s State, ss []State) bool {
	if len(ss) == 0 {
		return true
	}

	keep := false

	for _, state := range ss {
		if s == state {
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

	for _, ty := range ts {
		if t == ty {
			keep = true
			break
		}
	}

	return keep
}

func stringKey(con *Connection) string {
	return fmt.Sprintf("%d-%d-%s", con.FromID, con.ToID, con.Type)
}
