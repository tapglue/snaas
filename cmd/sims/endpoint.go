package main

import (
	"github.com/tapglue/snaas/core"
	"github.com/tapglue/snaas/platform/sns"
	"github.com/tapglue/snaas/service/app"
)

const serviceSNS = "SNS"

type endpointChange struct {
	ack            ackFunc
	EndpointArn    string `json:"EndpointArn"`
	EventType      string `json:"EventType"`
	FailureMessage string `json:"FailureMessage"`
	FailureType    string `json:"FailureType"`
	Resource       string `json:"Resource"`
	Service        string `json:"Service"`
}

func endpointUpdate(
	disableDevice core.DeviceDisableFunc,
	currentApp *app.App,
	c endpointChange,
) (err error) {
	defer func() {
		if err == nil {
			c.ack()
		}
	}()

	if c.Service != serviceSNS {
		return nil
	}

	if c.EventType != sns.TypeDeliveryFailure {
		return nil
	}

	return disableDevice(currentApp, c.EndpointArn)
}
