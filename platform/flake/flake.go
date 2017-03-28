package flake

import (
	"fmt"
	"time"

	"github.com/sony/sonyflake"
)

const fmtNamespace = "%s_%s"

var flakes = map[string]*sonyflake.Sonyflake{}

// Namespace returns the prefixed entity path.
func Namespace(prefix, entity string) string {
	return fmt.Sprintf(fmtNamespace, prefix, entity)
}

// NextID returns the next safe to use ID for the given namespace.
func NextID(namespace string) (uint64, error) {
	if _, ok := flakes[namespace]; !ok {
		var s sonyflake.Settings
		s.StartTime = time.Date(2015, 8, 31, 18, 7, 0, 0, time.UTC)

		flakes[namespace] = sonyflake.NewSonyflake(s)
	}

	return flakes[namespace].NextID()
}
