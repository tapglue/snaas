package limiter

import "time"

// Limitee is the limit that we want to apply.
type Limitee struct {
	Hash       string
	Limit      int64
	WindowSize time.Duration
}

// Limiter is the actual the one providing the actual limitation implementation.
type Limiter interface {
	// Request accepts a limitee parameter and for that it checks if it's still
	// within the limits or not. If not, it will return -1. If yes, it will
	// decrement the remaining number of hits by 1.
	Request(*Limitee) (int64, time.Time, error)
}

type limitee struct {
	hash       string
	limit      int64
	windowSize int64
}
