package main

import (
	"github.com/tapglue/snaas/core"
	"github.com/tapglue/snaas/platform/sns"
	"github.com/tapglue/snaas/service/app"
	"github.com/tapglue/snaas/service/device"
)

type channelFunc func(*app.App, *message) error

func channelPush(
	deviceListUser core.DeviceListUserFunc,
	deviceSync core.DeviceSyncEndpointFunc,
	push sns.PushFunc,
	pApps platformApps,
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
			pa, err := platformAppForPlatform(pApps, currentApp, d.Platform)
			if err != nil {
				if isPlatformNotFound(err) {
					continue
				}
				return err
			}

			d, err = deviceSync(currentApp, pa.ARN, d)
			if err != nil {
				return err
			}

			var p sns.Platform

			switch d.Platform {
			case device.PlatformAndroid:
				p = sns.PlatformGCM
			case device.PlatformIOS:
				p = sns.PlatformAPNS
			case device.PlatformIOSSandbox:
				p = sns.PlatformAPNSSandbox
			}

			err = push(p, d.EndpointARN, pa.Scheme, msg.urn, msg.message)
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
