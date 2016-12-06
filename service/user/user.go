package user

import (
	"fmt"
	"sort"
	"time"

	"github.com/asaskevich/govalidator"

	"github.com/tapglue/snaas/platform/service"
)

// TargetType is the identifier used for events targeting a User.
const TargetType = "tg_user"

var defaultEnabled = true

// Image represents a user image asset.
type Image struct {
	URL    string `json:"url"`
	Type   string `json:"type"`
	Height int    `json:"height"`
	Width  int    `json:"width"`
}

// List is a collection of users.
type List []*User

func (l List) Len() int {
	return len(l)
}

func (l List) Less(i, j int) bool {
	return l[i].CreatedAt.After(l[j].CreatedAt)
}

func (l List) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

// ToMap transforms the list to a Map.
func (l List) ToMap() Map {
	um := Map{}

	for _, u := range l {
		um[u.ID] = u
	}

	return um
}

// Map is a user collection with their id as index.
type Map map[uint64]*User

// Merge combines two Maps.
func (m Map) Merge(x Map) Map {
	for id, user := range x {
		m[id] = user
	}

	return m
}

// ToList returns the Map as an ordered List.
func (m Map) ToList() List {
	us := List{}

	for _, u := range m {
		us = append(us, u)
	}

	sort.Sort(us)

	return us
}

// Metadata is a bucket to provide additional user information.
type Metadata map[string]interface{}

// Private is the bucket for protected fields on a User.
type Private struct {
	Type     string `json:"type,omitempty"`
	Verified bool   `json:"verified"`
}

// QueryOptions is used to narrow-down user queries.
type QueryOptions struct {
	Before     uint64
	CustomIDs  []string
	Deleted    *bool
	Emails     []string
	Firstnames []string
	Enabled    *bool
	IDs        []uint64
	Lastnames  []string
	Limit      int
	SocialIDs  map[string][]string
	Usernames  []string
}

// Service for user interactions.
type Service interface {
	service.Lifecycle

	Count(namespace string, opts QueryOptions) (int, error)
	Put(namespace string, user *User) (*User, error)
	PutLastRead(namespace string, userID uint64, lastRead time.Time) error
	Query(namespace string, opts QueryOptions) (List, error)
	Search(namespace string, opts QueryOptions) (List, error)
}

// ServiceMiddleware is a chainable behaviour modifier for Service.
type ServiceMiddleware func(Service) Service

// User is the representation of a customer of an app.
type User struct {
	About          string            `json:"about"`
	CustomID       string            `json:"custom_id,omitempty"`
	Deleted        bool              `json:"deleted"`
	Enabled        bool              `json:"enabled"`
	Email          string            `json:"email,omitempty"`
	Firstname      string            `json:"first_name"`
	FollowerCount  int               `json:"-"`
	FollowingCount int               `json:"-"`
	FriendCount    int               `json:"-"`
	ID             uint64            `json:"id"`
	Images         map[string]Image  `json:"images,omitempty"`
	IsFollower     bool              `json:"-"`
	IsFollowing    bool              `json:"-"`
	IsFriend       bool              `json:"-"`
	Lastname       string            `json:"last_name"`
	LastRead       time.Time         `json:"-"`
	Metadata       Metadata          `json:"metadata"`
	Password       string            `json:"password"`
	Private        *Private          `json:"private,omitempty"`
	SessionToken   string            `json:"-"`
	SocialIDs      map[string]string `json:"social_ids,omitempty"`
	URL            string            `json:"url,omitempty"`
	Username       string            `json:"user_name,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// Validate performs semantic checks on the passed User values for correctness.
func (u *User) Validate() error {
	if u.Email == "" && u.Username == "" {
		return wrapError(ErrInvalidUser, "email or username must be set")
	}

	if ok := govalidator.IsEmail(u.Email); u.Email != "" && !ok {
		return wrapError(ErrInvalidUser, "invalid email address '%s'", u.Email)
	}

	if u.Firstname != "" {
		if len(u.Firstname) < 1 {
			return wrapError(ErrInvalidUser, "firstname too short")
		}
		if len(u.Firstname) > 40 {
			return wrapError(ErrInvalidUser, "firstname too long")
		}
	}

	if u.Lastname != "" {
		if len(u.Lastname) < 1 {
			return wrapError(ErrInvalidUser, "lastname too short")
		}
		if len(u.Lastname) > 40 {
			return wrapError(ErrInvalidUser, "lastname too long")
		}
	}

	if ok := govalidator.IsURL(u.URL); u.URL != "" && !ok {
		return wrapError(ErrInvalidUser, "invalid url")
	}

	if u.Password == "" {
		return wrapError(ErrInvalidUser, "password must be set")
	}

	if u.Username != "" {
		if len(u.Username) < 2 {
			return wrapError(ErrInvalidUser, "username too short")
		}
		if len(u.Username) > 40 {
			return wrapError(ErrInvalidUser, "username too long")
		}
	}

	return nil
}

// ListFromIDs gathers a user collection from the Service for the given ids.
func ListFromIDs(s Service, ns string, ids ...uint64) (List, error) {
	var (
		is   = []uint64{}
		seen = map[uint64]struct{}{}
	)

	if len(ids) == 0 {
		return List{}, nil
	}

	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}

		is = append(is, id)
	}

	return s.Query(ns, QueryOptions{
		Enabled: &defaultEnabled,
		IDs:     is,
	})
}

// MapFromIDs return a populated user map for the given list of ids.
func MapFromIDs(s Service, ns string, ids ...uint64) (Map, error) {
	us, err := ListFromIDs(s, ns, ids...)
	if err != nil {
		return nil, err
	}

	um := Map{}

	for _, u := range us {
		um[u.ID] = u
	}

	return um, nil
}

func flakeNamespace(ns string) string {
	return fmt.Sprintf("%s_%s", ns, "users")
}
