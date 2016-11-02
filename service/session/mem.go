package session

import "time"

type memService struct {
	sessions map[string]Map
}

// MemService returns a memory based Service implementation.
func MemService() Service {
	return &memService{
		sessions: map[string]Map{},
	}
}

func (s *memService) Put(ns string, session *Session) (*Session, error) {
	if err := s.Setup(ns); err != nil {
		return nil, err
	}

	if err := session.Validate(); err != nil {
		return nil, err
	}

	bucket := s.sessions[ns]

	if session.ID == "" {
		session.ID = generateID()
		session.CreatedAt = time.Now().UTC()
	} else {
		keep := false

		for _, s := range bucket {
			if s.ID == session.ID {
				keep = true
				session.CreatedAt = s.CreatedAt
			}
		}

		if !keep {
			return nil, ErrNotFound
		}
	}

	bucket[session.ID] = copy(session)

	return copy(session), nil
}

func (s *memService) Query(ns string, opts QueryOptions) (List, error) {
	if err := s.Setup(ns); err != nil {
		return nil, err
	}

	return filterMap(s.sessions[ns], opts), nil
}

func (s *memService) Setup(ns string) error {
	if _, ok := s.sessions[ns]; !ok {
		s.sessions[ns] = Map{}
	}

	return nil
}

func (s *memService) Teardown(ns string) error {
	if _, ok := s.sessions[ns]; ok {
		delete(s.sessions, ns)
	}

	return nil
}

func copy(s *Session) *Session {
	old := *s
	return &old
}

func filterMap(sm Map, opts QueryOptions) List {
	ss := List{}

	for id, s := range sm {
		if !inTypes(s.DeviceID, opts.DeviceIDs) {
			continue
		}

		if opts.Enabled != nil && s.Enabled != *opts.Enabled {
			continue
		}

		if !inTypes(id, opts.IDs) {
			continue
		}

		if !inIDs(s.UserID, opts.UserIDs) {
			continue
		}

		ss = append(ss, s)
	}

	return ss
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
