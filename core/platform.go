package core

import (
	pErr "github.com/tapglue/snaas/error"
	"github.com/tapglue/snaas/platform/pg"
	"github.com/tapglue/snaas/platform/sns"
	"github.com/tapglue/snaas/service/app"
	"github.com/tapglue/snaas/service/platform"
)

var (
	defaultActive = true
)

// PlatformCreateFunc stores the provided platform.
type PlatformCreateFunc func(
	currentApp *app.App,
	p *platform.Platform,
	cert, key string,
) (*platform.Platform, error)

// PlatformCreate stores the provided platform.
func PlatformCreate(
	platforms platform.Service,
	createAPNS sns.AppCreateAPNSFunc,
	createAPNSSandbox sns.AppCreateAPNSSandboxFunc,
	createAndroid sns.AppCreateGCMFunc,
) PlatformCreateFunc {
	return func(
		currentApp *app.App,
		p *platform.Platform,
		cert, key string,
	) (*platform.Platform, error) {
		arn := ""

		switch p.Ecosystem {
		case platform.Android:
			var err error
			arn, err = createAndroid(p.Name, key)
			if err != nil {
				return nil, err
			}
		case platform.IOS:
			var err error
			arn, err = createAPNS(p.Name, cert, key)
			if err != nil {
				return nil, err
			}
		case platform.IOSSandbox:
			var err error
			arn, err = createAPNSSandbox(p.Name, cert, key)
			if err != nil {
				return nil, err
			}
		}

		p.AppID = currentApp.ID
		p.ARN = arn

		return platforms.Put(pg.MetaNamespace, p)
	}
}

// PlatformFetchActiveFunc returns the active platform for the current app and the
// given ecosystem.
type PlatformFetchActiveFunc func(*app.App, sns.Platform) (*platform.Platform, error)

// PlatformFetchActive returns the active platform for the current app and the
// given ecosystem.
func PlatformFetchActive(platforms platform.Service) PlatformFetchActiveFunc {
	return func(
		currentApp *app.App,
		ecosystem sns.Platform,
	) (*platform.Platform, error) {
		ps, err := platforms.Query(pg.MetaNamespace, platform.QueryOptions{
			Active: &defaultActive,
			AppIDs: []uint64{
				currentApp.ID,
			},
			Deleted: &defaultDeleted,
			Ecosystems: []sns.Platform{
				ecosystem,
			},
		})
		if err != nil {
			return nil, err
		}

		if len(ps) != 1 {
			return nil, pErr.Wrap(
				pErr.ErrNotFound,
				"no active platform found for %s",
				sns.PlatformIdentifiers[ecosystem],
			)
		}

		return ps[0], nil
	}
}

// PlatformFetchByARNFunc returns the Platform for the given ARN.
type PlatformFetchByARNFunc func(arn string) (*platform.Platform, error)

// PlatformFetchByARN returns the Platform for the given ARN.
func PlatformFetchByARN(platforms platform.Service) PlatformFetchByARNFunc {
	return func(arn string) (*platform.Platform, error) {
		ps, err := platforms.Query(pg.MetaNamespace, platform.QueryOptions{
			ARNs: []string{
				arn,
			},
			Deleted: &defaultDeleted,
		})
		if err != nil {
			return nil, err
		}

		if len(ps) != 1 {
			return nil, pErr.Wrap(
				pErr.ErrNotFound,
				"no platform found for '%s'",
				arn,
			)
		}

		return ps[0], nil
	}
}
