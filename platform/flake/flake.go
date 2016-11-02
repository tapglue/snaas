package flake

import (
	"time"

	"github.com/sony/sonyflake"
)

var flakes = map[string]*sonyflake.Sonyflake{}

// NextID returns the next safe to use ID for the given namespace.
func NextID(namespace string) (uint64, error) {
	if _, ok := flakes[namespace]; !ok {
		var s sonyflake.Settings
		s.StartTime = time.Date(2015, 8, 31, 18, 7, 0, 0, time.UTC)

		flakes[namespace] = sonyflake.NewSonyflake(s)
	}

	return flakes[namespace].NextID()
}
