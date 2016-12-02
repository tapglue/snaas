package main

import (
	"github.com/tapglue/snaas/core"
	pErr "github.com/tapglue/snaas/error"
	"github.com/tapglue/snaas/platform/sns"
	"github.com/tapglue/snaas/service/app"
)

type channelFunc func(*app.App, *message) error

func channelPush(
	deviceListUser core.DeviceListUserFunc,
	deviceSync core.DeviceSyncEndpointFunc,
	fetchActive core.PlatformFetchActiveFunc,
	push sns.PushFunc,
) channelFunc {
	return func(currentApp *app.App, msg *message) error {
		ds, err := deviceListUser(currentApp, msg.recipient)
		if err != nil {
			return err
		}
		if len(ds) == 0 {
			return nil
		}

		for _, d := range ds {
			p, err := fetchActive(currentApp, d.Platform)
			if err != nil {
				if pErr.IsNotFound(err) {
					continue
				}
				return err
			}

			d, err = deviceSync(currentApp, p.ARN, d)
			if err != nil {
				return err
			}

			err = push(d.Platform, d.EndpointARN, p.Scheme, msg.urn, msg.message)
			if err != nil {
				if sns.IsDeliveryFailure(err) {
					return nil
				}

				return err
			}
		}

		return nil
	}
}
