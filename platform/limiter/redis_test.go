package limiter

import (
	"testing"
	"time"

	"github.com/garyburd/redigo/redis"
)

func TestLimiter(t *testing.T) {
	var (
		pool = redis.NewPool(func() (redis.Conn, error) {
			return redis.Dial("tcp", "127.0.0.1:6379")
		}, 10)
		limitee = &Limitee{
			Hash:       "token",
			Limit:      10,
			WindowSize: 1 * time.Second,
		}
		l = Redis(pool, "limitertest")
	)

	conn := pool.Get()
	defer conn.Close()

	// XXX This simulates the faulty EC behaviour we observed which leaves a key
	// without a TTL and the value set to -1.
	_, err := conn.Do(
		"SET",
		"limitertest:token",
		-1,
	)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		_, _, err := l.Request(limitee)
		if err != nil {
			t.Fatalf("request failed: %s", err)
		}
	}

	limit, _, err := l.Request(limitee)
	if err != nil {
		t.Fatalf("request failed: %s", err)
	}

	if have, want := limit, int64(-1); have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	time.Sleep(1 * time.Second)

	limit, _, err = l.Request(limitee)
	if err != nil {
		t.Fatalf("request failed: %s", err)
	}

	if have, want := limit, int64(9); have != want {
		t.Errorf("have %v, want %v", have, want)
	}
}
