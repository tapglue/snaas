package device

import (
	"fmt"
	"time"

	"golang.org/x/text/language"

	"github.com/tapglue/snaas/platform/service"
	"github.com/tapglue/snaas/platform/sns"
)

const (
	// DefaultLanguage for devices.
	DefaultLanguage = "en"
)

// Platform supported for a Device.
const (
	PlatformIOS        = sns.PlatformAPNS
	PlatformIOSSandbox = sns.PlatformAPNSSandbox
	PlatformAndroid    = sns.PlatformGCM
)

// Device represents a physical device like mobile phone or tablet of a user.
type Device struct {
	Deleted     bool
	DeviceID    string
	Disabled    bool
	EndpointARN string
	ID          uint64
	Language    string
	Platform    sns.Platform
	Token       string
	UserID      uint64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Validate returns an error when a semantic check fails.
func (d *Device) Validate() error {
	if d.DeviceID == "" {
		return wrapError(ErrInvalidDevice, "DeviceID must be set")
	}

	if _, err := language.Parse(d.Language); err != nil {
		return wrapError(ErrInvalidDevice, "Language invalid '%s'", d.Language)
	}

	if d.Platform == 0 {
		return wrapError(ErrInvalidDevice, "Platform must be set")
	}

	if d.Platform > PlatformAndroid {
		return wrapError(ErrInvalidDevice, "Platform '%d' not supported", d.Platform)
	}

	if d.Token == "" {
		return wrapError(ErrInvalidDevice, "Token must be set")
	}

	if d.UserID == 0 {
		return wrapError(ErrInvalidDevice, "UserID must be set")
	}

	return nil
}

// List is a collection of devices.
type List []*Device

// QueryOptions is used to narrow-down user queries.
type QueryOptions struct {
	Deleted      *bool
	DeviceIDs    []string
	Disabled     *bool
	EndpointARNs []string
	IDs          []uint64
	Platforms    []sns.Platform
	UserIDs      []uint64
}

// Service for device interactions.
type Service interface {
	service.Lifecycle

	Put(namespace string, device *Device) (*Device, error)
	Query(namespace string, opts QueryOptions) (List, error)
}

// ServiceMiddleware is a chainable behaviour modifier for Service.
type ServiceMiddleware func(Service) Service

func flakeNamespace(ns string) string {
	return fmt.Sprintf("%s_%s", ns, "devices")
}
