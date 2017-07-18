package counter

import (
	"github.com/tapglue/snaas/platform/service"
)

// Service for counter interactions.
type Service interface {
	service.Lifecycle

	Count(namespace, name string, userID uint64) (uint64, error)
	CountAll(namespace, name string) (uint64, error)
	Set(namespace, name string, userID, value uint64) error
}

// ServiceMiddleware is a chainable behaviour modifier for Service.
type ServiceMiddleware func(Service) Service
