package main

import (
	"github.com/tapglue/snaas/core"
	serr "github.com/tapglue/snaas/error"
	"github.com/tapglue/snaas/platform/sns"
	"github.com/tapglue/snaas/service/app"
)

type channelFunc func(*app.App, *core.Message) error

func channelPush(
	deviceListUser core.DeviceListUserFunc,
	deviceSync core.DeviceSyncEndpointFunc,
	fetchActive core.PlatformFetchActiveFunc,
	push sns.PushFunc,
) channelFunc {
	return func(currentApp *app.App, msg *core.Message) error {
		ds, err := deviceListUser(currentApp, msg.Recipient)
		if err != nil {
			return err
		}
		if len(ds) == 0 {
			return nil
		}

		for _, d := range ds {
			p, err := fetchActive(currentApp, d.Platform)
			if err != nil {
				if serr.IsNotFound(err) {
					continue
				}

				return err
			}

			d, err = deviceSync(currentApp, p.ARN, d)
			if err != nil {
				if serr.IsDeviceDisabled(err) {
					continue
				}

				return err
			}

			err = push(d.Platform, d.EndpointARN, p.Scheme, msg.URN, msg.Message)
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
