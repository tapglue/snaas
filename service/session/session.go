package session

import (
	"encoding/base64"
	"math/rand"
	"time"

	"github.com/tapglue/snaas/platform/generate"
	"github.com/tapglue/snaas/platform/service"
)

// DeviceIDUnknown is the default for untracked devices.
const DeviceIDUnknown = "unknown"

// List is a collection of sessions.
type List []*Session

// Map is a session collection with their id as index.
type Map map[string]*Session

// QueryOptions is used to narrow-down session queries.
type QueryOptions struct {
	DeviceIDs []string
	Enabled   *bool
	IDs       []string
	UserIDs   []uint64
}

// Service for session interactions
type Service interface {
	service.Lifecycle

	Put(namespace string, session *Session) (*Session, error)
	Query(namespace string, opts QueryOptions) (List, error)
}

// ServiceMiddleware is a chainable behaviour modifier for Service.
type ServiceMiddleware func(Service) Service

// Session attaches a session id to a user id.
type Session struct {
	CreatedAt time.Time
	DeviceID  string
	Enabled   bool
	ID        string
	UserID    uint64
}

// Validate performs semantic checks on the Session.
func (s *Session) Validate() error {
	if s.DeviceID == "" {
		return wrapError(ErrInvalidSession, "DeviceID must be set")
	}

	if s.UserID == 0 {
		return wrapError(ErrInvalidSession, "UserID must be set")
	}

	return nil
}

func generateID() string {
	src := rand.NewSource(time.Now().UnixNano())

	return base64.StdEncoding.EncodeToString(generate.RandomBytes(src, 20))
}
