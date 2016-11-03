package user

import (
	"strings"
	"time"

	"github.com/tapglue/snaas/platform/flake"
)

type memService struct {
	users map[string]Map
}

// MemService returns a memory based Service implementation.
func MemService() Service {
	return &memService{
		users: map[string]Map{},
	}
}

func (s *memService) Count(ns string, opts QueryOptions) (int, error) {
	if err := s.Setup(ns); err != nil {
		return 0, err
	}

	return len(filterMap(s.users[ns], opts)), nil
}

func (s *memService) Put(ns string, input *User) (*User, error) {
	if err := s.Setup(ns); err != nil {
		return nil, err
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	var (
		bucket = s.users[ns]
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

		for _, u := range bucket {
			if u.ID == input.ID {
				keep = true
				input.CreatedAt = u.CreatedAt
			}
		}

		if !keep {
			return nil, ErrNotFound
		}
	}

	input.UpdatedAt = now
	bucket[input.ID] = copy(input)

	return copy(input), nil
}

func (s *memService) PutLastRead(ns string, userID uint64, ts time.Time) error {
	if err := s.Setup(ns); err != nil {
		return err
	}

	u, ok := s.users[ns][userID]
	if ok {
		u.LastRead = ts.UTC()
		s.users[ns][userID] = u
	}

	return nil
}
func (s *memService) Query(ns string, opts QueryOptions) (List, error) {
	if err := s.Setup(ns); err != nil {
		return nil, err
	}

	us := filterMap(s.users[ns], opts)

	if opts.Limit > 0 && len(us) > opts.Limit {
		us = us[:opts.Limit]
	}

	return us, nil
}

func (s *memService) Search(ns string, opts QueryOptions) (List, error) {
	if err := s.Setup(ns); err != nil {
		return nil, err
	}

	sOpts := opts

	opts.Emails = nil
	opts.Firstnames = nil
	opts.Lastnames = nil
	opts.Usernames = nil

	us := filterMap(s.users[ns], opts)
	us = searchUsers(us, sOpts)

	return us, nil
}

func (s *memService) Setup(ns string) error {
	if _, ok := s.users[ns]; !ok {
		s.users[ns] = Map{}
	}

	return nil
}

func (s *memService) Teardown(ns string) error {
	if _, ok := s.users[ns]; ok {
		delete(s.users, ns)
	}

	return nil
}

func contains(s string, ts ...string) bool {
	if len(ts) == 0 {
		return true
	}

	keep := false

	for _, t := range ts {
		if keep = strings.Contains(s, t); keep {
			break
		}
	}

	return keep
}

func copy(u *User) *User {
	old := *u
	return &old
}

func filterMap(um Map, opts QueryOptions) List {
	us := List{}

	for id, u := range um {
		if !inTypes(u.CustomID, opts.CustomIDs) {
			continue
		}

		if opts.Deleted != nil && u.Deleted != *opts.Deleted {
			continue
		}

		if !inTypes(u.Email, opts.Emails) {
			continue
		}

		if opts.Enabled != nil && u.Enabled != *opts.Enabled {
			continue
		}

		if !inIDs(id, opts.IDs) {
			continue
		}

		if opts.SocialIDs != nil {
			keep := false

			for platform, ids := range opts.SocialIDs {
				if _, ok := u.SocialIDs[platform]; !ok {
					continue
				}

				if !inTypes(u.SocialIDs[platform], ids) {
					continue
				}

				keep = true
			}

			if !keep {
				continue
			}
		}

		if !inTypes(u.Username, opts.Usernames) {
			continue
		}

		us = append(us, u)
	}

	return us
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

func searchUsers(is List, opts QueryOptions) List {
	us := List{}

	for _, u := range is {
		if !contains(u.Email, opts.Emails...) ||
			!contains(u.Firstname, opts.Firstnames...) ||
			!contains(u.Lastname, opts.Lastnames...) ||
			!contains(u.Username, opts.Usernames...) {
			continue
		}

		us = append(us, u)
	}

	return us
}
