package reaction

import (
	"fmt"
	"strings"

	"github.com/tapglue/snaas/platform/cache"
)

const (
	cachePrefixCount = "reactions.count"
)

type cacheService struct {
	countsCache cache.CountService
	next        Service
}

// CacheServiceMiddleware adds caching capabilities to the Service by using
// read-through and write-through methods to store resutls of heavy computation
// with sensible TTLs.
func CacheServiceMiddleware(countsCache cache.CountService) ServiceMiddleware {
	return func(next Service) Service {
		return &cacheService{
			countsCache: countsCache,
			next:        next,
		}
	}
}

func (s *cacheService) Count(ns string, opts QueryOptions) (uint, error) {
	key := cacheCountKey(opts)

	count, err := s.countsCache.Get(ns, key)
	if err == nil {
		return uint(count), nil
	}

	if !cache.IsKeyNotFound(err) {
		return 0, err
	}

	aCount, err := s.next.Count(ns, opts)
	if err != nil {
		return 0, err
	}

	err = s.countsCache.Set(ns, key, int(aCount))

	return aCount, err
}

func (s *cacheService) Put(ns string, input *Reaction) (*Reaction, error) {
	key := cacheCountKey(QueryOptions{
		ObjectIDs: []uint64{
			input.ObjectID,
		},
		Types: []Type{
			input.Type,
		},
	})

	new := false

	if input.ID == 0 {
		new = true
	}

	r, err := s.next.Put(ns, input)
	if err != nil {
		return nil, err
	}

	if input.Deleted {
		_, err := s.countsCache.Decr(ns, key)
		if err != nil {
			// We ignore the error of the cache operation.
		}
	} else if new {
		_, err := s.countsCache.Incr(ns, key)
		if err != nil {
			// We ignore the error of the cache operation.
		}
	}

	return r, nil
}

func (s *cacheService) Query(ns string, opts QueryOptions) (List, error) {
	return s.next.Query(ns, opts)
}

func (s *cacheService) Setup(ns string) error {
	return s.next.Setup(ns)
}

func (s *cacheService) Teardown(ns string) error {
	return s.next.Teardown(ns)
}

func cacheCountKey(opts QueryOptions) string {
	ps := []string{
		cachePrefixCount,
	}

	if len(opts.Types) == 1 {
		ps = append(ps, TypeToIdenitifier[opts.Types[0]])
	}

	if len(opts.ObjectIDs) == 1 {
		ps = append(ps, fmt.Sprintf("%d", opts.ObjectIDs[0]))
	}

	return strings.Join(ps, cache.KeySeparator)
}
