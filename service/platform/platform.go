package platform

import (
	"fmt"
	"time"

	pErr "github.com/tapglue/snaas/error"
	"github.com/tapglue/snaas/platform/service"
	"github.com/tapglue/snaas/platform/sns"
)

// Ecosystem supported for a Platform.
const (
	IOS        = sns.PlatformAPNS
	IOSSandbox = sns.PlatformAPNSSandbox
	Android    = sns.PlatformGCM
)

// Platform represents an ecosystem like Android or iOS for user device management.
type Platform struct {
	Active    bool
	AppID     uint64
	ARN       string
	Deleted   bool
	Ecosystem sns.Platform
	ID        uint64
	Name      string
	Scheme    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Validate check for semantic correctness.
func (p *Platform) Validate() error {
	if p.ARN == "" {
		return pErr.Wrap(pErr.ErrInvalidPlatform, "ARN must be set")
	}

	if p.Ecosystem == 0 {
		return pErr.Wrap(pErr.ErrInvalidPlatform, "Ecosystem must be set")
	}

	if p.Ecosystem > Android {
		return pErr.Wrap(pErr.ErrInvalidPlatform, "Ecosystem '%d' not supported", p.Ecosystem)
	}

	if p.Name == "" {
		return pErr.Wrap(pErr.ErrInvalidPlatform, "Name must be set")
	}

	if p.Scheme == "" {
		return pErr.Wrap(pErr.ErrInvalidPlatform, "Scheme must be set")
	}

	return nil
}

// List is a collection of Platforms.
type List []*Platform

// QueryOptions to narrow-down platform queries.
type QueryOptions struct {
	Active     *bool
	ARNs       []string
	AppIDs     []uint64
	Deleted    *bool
	Ecosystems []sns.Platform
	IDs        []uint64
}

// Service for platform interactions.
type Service interface {
	service.Lifecycle

	Put(namespace string, platform *Platform) (*Platform, error)
	Query(namespace string, opts QueryOptions) (List, error)
}

// ServiceMiddleware is a chainable behaviour modifier for Service.
type ServiceMiddleware func(Service) Service

func flakeNamespace(ns string) string {
	return fmt.Sprintf("%s_%s", ns, "platforms")
}
