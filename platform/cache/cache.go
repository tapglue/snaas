package cache

// KeySeparator is used to build complete keys out of parts.
const KeySeparator = "."

const countPrefix = "cache.count"

// CountService caches counts separated by namespace.
type CountService interface {
	Decr(namespace, key string) (int, error)
	Get(namespace, key string) (int, error)
	Incr(namespace, key string) (int, error)
	Set(namespace, key string, count int) error
}

// CountServiceMiddleware is a chainable behaviour modifier for CountService.
type CountServiceMiddleware func(CountService) CountService
