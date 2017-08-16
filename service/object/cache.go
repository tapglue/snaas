package object

import (
	"fmt"
	"strings"

	"github.com/tapglue/snaas/platform/cache"
)

const (
	cachePrefixCount = "objects.count"
)

type cacheService struct {
	countsCache cache.CountService
	next        Service
}

// CacheServiceMiddleware adds caching capabilities to the Service by using
// read-through and write-through methods to store results of heavy computation
// with sensible TTLs.
func CacheServiceMiddleware(countsCache cache.CountService) ServiceMiddleware {
	return func(next Service) Service {
		return &cacheService{
			countsCache: countsCache,
			next:        next,
		}
	}
}

func (s *cacheService) Count(ns string, opts QueryOptions) (int, error) {
	key := cacheCountKey(opts)

	count, err := s.countsCache.Get(ns, key)
	if err == nil {
		return count, nil
	}

	if !cache.IsKeyNotFound(err) {
		return -1, err
	}

	count, err = s.next.Count(ns, opts)
	if err != nil {
		return -1, err
	}

	err = s.countsCache.Set(ns, key, count)

	return count, err
}

func (s *cacheService) Put(ns string, input *Object) (output *Object, err error) {
	key := cacheCountKey(QueryOptions{
		Types: []string{
			input.Type,
		},
		ObjectIDs: []uint64{
			input.ObjectID,
		},
	})

	o, err := s.next.Put(ns, input)
	if err != nil {
		return nil, err
	}

	if input.Deleted {
		_, err := s.countsCache.Decr(ns, key)
		if err != nil {
			// We ignore the error of the cache operation.
		}
	} else {
		_, err := s.countsCache.Incr(ns, key)
		if err != nil {
			// We ignore the error of the cache operation.
		}
	}

	return o, nil
}

func (s *cacheService) Query(ns string, opts QueryOptions) (os List, err error) {
	return s.next.Query(ns, opts)
}

func (s *cacheService) Setup(ns string) (err error) {
	return s.next.Setup(ns)
}

func (s *cacheService) Teardown(ns string) (err error) {
	return s.next.Teardown(ns)
}

func cacheCountKey(opts QueryOptions) string {
	ps := []string{
		cachePrefixCount,
	}

	if len(opts.Types) == 1 {
		ps = append(ps, opts.Types[0])
	}

	if len(opts.ObjectIDs) == 1 {
		ps = append(ps, fmt.Sprintf("%d", opts.ObjectIDs[0]))
	}

	return strings.Join(ps, cache.KeySeparator)
}
