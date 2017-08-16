package redis

import (
	"time"

	"github.com/garyburd/redigo/redis"
)

// Commands.
const (
	CommandAuth   = "AUTH"
	CommandDecr   = "DECR"
	CommandEx     = "EX"
	CommandExec   = "Exec"
	CommandExpire = "EXPIRE"
	CommandGet    = "GET"
	CommandIncr   = "INCR"
	CommandMulti  = "MULTI"
	CommandPing   = "PING"
	CommandSet    = "SET"
)

// Defaults.
const (
	defaultIdleTimeout = 240 * time.Second
	defaultMaxIdle     = 10
	defaultNetwork     = "tcp"
)

type borrowFunc func(redis.Conn, time.Time) error
type dialFunc func() (redis.Conn, error)

func Pool(addr, password string) *redis.Pool {
	return &redis.Pool{
		Dial:         dial(addr, password),
		IdleTimeout:  defaultIdleTimeout,
		MaxIdle:      defaultMaxIdle,
		TestOnBorrow: borrow,
	}
}

func borrow(c redis.Conn, t time.Time) error {
	if time.Since(t) < time.Minute {
		return nil
	}

	_, err := c.Do(CommandPing)
	return err
}

func dial(addr, password string) dialFunc {
	return func() (redis.Conn, error) {
		c, err := redis.Dial(defaultNetwork, addr)
		if err != nil {
			return nil, err
		}

		if password != "" {
			if _, err := c.Do(CommandAuth, password); err != nil {
				c.Close()

				return nil, err
			}
		}

		return c, err
	}
}
