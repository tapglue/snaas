package cache

import (
	"math/rand"
	"testing"

	"github.com/garyburd/redigo/redis"

	predis "github.com/tapglue/snaas/platform/redis"
)

func TestRedisCountServiceDecr(t *testing.T) {
	var (
		key       = "decr"
		namespace = "counter"
		pool      = newPool()
		s         = RedisCountService(pool)

		count = rand.Int()
	)

	con := pool.Get()
	defer con.Close()

	_, err := con.Do(
		predis.CommandSet,
		prefixKey(namespace, key),
		uint64(count),
	)
	if err != nil {
		t.Fatal(err)
	}

	res, err := s.Decr(namespace, key)
	if err != nil {
		t.Fatal(err)
	}

	if have, want := res, count-1; have != want {
		t.Errorf("have %v, want %v", have, want)
	}
}

func TestRedisCountServiceGet(t *testing.T) {
	var (
		key       = "get"
		namespace = "counter"
		pool      = newPool()
		s         = RedisCountService(pool)

		count = rand.Int()
	)

	con := pool.Get()
	defer con.Close()

	_, err := con.Do(
		predis.CommandSet,
		prefixKey(namespace, key),
		uint64(count),
	)
	if err != nil {
		t.Fatal(err)
	}

	res, err := s.Get(namespace, key)
	if err != nil {
		t.Fatal(err)
	}

	if have, want := res, count; have != want {
		t.Errorf("have %v, want %v", have, want)
	}
}

func TestRedisCountServiceIncr(t *testing.T) {
	var (
		key       = "incr"
		namespace = "counter"
		pool      = newPool()
		s         = RedisCountService(pool)

		count = rand.Int()
	)

	con := pool.Get()
	defer con.Close()

	_, err := con.Do(
		predis.CommandSet,
		prefixKey(namespace, key),
		uint64(count),
	)
	if err != nil {
		t.Fatal(err)
	}

	res, err := s.Incr(namespace, key)
	if err != nil {
		t.Fatal(err)
	}

	if have, want := res, count+1; have != want {
		t.Errorf("have %v, want %v", have, want)
	}
}

func TestRedisCountServiceSet(t *testing.T) {
	var (
		key       = "set"
		namespace = "counter"
		pool      = newPool()
		s         = RedisCountService(pool)

		count = rand.Int()
	)

	err := s.Set(namespace, key, count)
	if err != nil {
		t.Fatal(err)
	}

	con := pool.Get()
	defer con.Close()

	res, err := redis.Int(con.Do(
		predis.CommandGet,
		prefixKey(namespace, key),
	))
	if err != nil {
		t.Fatal(err)
	}

	if have, want := res, count; have != want {
		t.Errorf("have %v, want %v", have, want)
	}
}

func newPool() *redis.Pool {
	return redis.NewPool(func() (redis.Conn, error) {
		return redis.Dial("tcp", "127.0.0.1:6379")
	}, 10)
}
