package cache

import (
	"fmt"
	"strings"

	"github.com/garyburd/redigo/redis"

	predis "github.com/tapglue/snaas/platform/redis"
)

const (
	cacheTTLDefault = 86400
	errCode         = -1
)

type redisCountService struct {
	pool *redis.Pool
}

func RedisCountService(pool *redis.Pool) CountService {
	return &redisCountService{
		pool: pool,
	}
}

func (s *redisCountService) Decr(ns, key string) (int, error) {
	con := s.pool.Get()
	defer con.Close()

	con.Send(predis.CommandMulti)
	con.Send(predis.CommandDecr, prefixKey(ns, key))
	con.Send(predis.CommandExpire, prefixKey(ns, key), cacheTTLDefault)

	res, err := redis.Values(con.Do(predis.CommandExec))
	if err != nil {
		return 0, fmt.Errorf("cache decr failed: %s", err)
	}

	var count int

	if _, err := redis.Scan(res, &count); err != nil {
		return 0, err
	}

	return count, nil
}

func (s *redisCountService) Get(ns, key string) (int, error) {
	var (
		con = s.pool.Get()

		count uint64
	)
	defer con.Close()

	res, err := con.Do(predis.CommandGet, prefixKey(ns, key))
	if err != nil {
		return errCode, fmt.Errorf("cache get failed: %s", err)
	}

	if res == nil {
		return errCode, wrapError(ErrKeyNotFound, "%s.%s", ns, key)
	}

	_, err = redis.Scan([]interface{}{res}, &count)
	if err != nil {
		return errCode, fmt.Errorf("cache scan failed: %s", err)
	}

	return int(count), nil
}

func (s *redisCountService) Incr(ns, key string) (int, error) {
	con := s.pool.Get()
	defer con.Close()

	con.Send(predis.CommandMulti)
	con.Send(predis.CommandIncr, prefixKey(ns, key))
	con.Send(predis.CommandExpire, prefixKey(ns, key), cacheTTLDefault)

	res, err := redis.Values(con.Do(predis.CommandExec))
	if err != nil {
		return 0, fmt.Errorf("cache incr failed: %s", err)
	}

	var count int

	if _, err := redis.Scan(res, &count); err != nil {
		return 0, err
	}

	return count, nil
}

func (s *redisCountService) Set(ns, key string, count int) error {
	con := s.pool.Get()
	defer con.Close()

	_, err := con.Do(
		predis.CommandSet,
		prefixKey(ns, key),
		uint64(count),
		predis.CommandEx,
		cacheTTLDefault,
	)
	if err != nil {
		return fmt.Errorf("cache set failed: %s", err)
	}

	return nil
}

func prefixKey(ns, key string) string {
	ps := []string{
		countPrefix,
		ns,
		key,
	}

	return strings.Join(ps, KeySeparator)
}
