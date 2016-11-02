package cache

import (
	"fmt"
	"strings"

	"github.com/garyburd/redigo/redis"
)

const (
	cacheTTLDefault = 300

	redisCommandEX  = "EX"
	redisCommandGET = "GET"
	redisCommandSET = "SET"

	errCode = -1
)

type redisCountService struct {
	pool *redis.Pool
}

func RedisCountService(pool *redis.Pool) CountService {
	return &redisCountService{
		pool: pool,
	}
}

func (s *redisCountService) Get(ns, key string) (int, error) {
	var (
		con          = s.pool.Get()
		count uint64 = 0
	)
	defer con.Close()

	res, err := con.Do(redisCommandGET, prefixKey(ns, key))
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

func (s *redisCountService) Set(ns, key string, count int) error {
	con := s.pool.Get()
	defer con.Close()

	_, err := con.Do(
		redisCommandSET,
		prefixKey(ns, key),
		uint64(count),
		redisCommandEX,
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
