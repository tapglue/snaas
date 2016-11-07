package sns

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/sns"
)

// Message Attributes.
const (
	AttributeEnabled = "Enabled"
	AttributeToken   = "Token"
)

// Message Types.
const (
	TypeDeliveryFailure = "DeliveryFailure"
	TypeNotification    = "Notification"
)

// Platform supported by SNS for push.
const (
	PlatformADM Platform = iota + 1
	PlatformAPNS
	PlatformAPNSSandbox
	PlatformGCM
)

// Publish structures.
const (
	structureJSON = "json"
)

// Push formats.
const (
	fmtURN         = `%s://%s`
	msgAPNS        = `{"APNS": "{\"aps\": {\"alert\": \"%s\"}, \"urn\":\"%s\"}" }`
	msgAPNSSandbox = `{"APNS_SANDBOX": "{\"aps\": {\"alert\": \"%s\"}, \"urn\":\"%s\"}" }`
	msgGCM         = `{"GCM": "{\"notification\": {\"title\": \"%s\", \"data\": {\"urn\": \"%s\"}} }"}`
)

type API interface {
	CreatePlatformEndpoint(*sns.CreatePlatformEndpointInput) (*sns.CreatePlatformEndpointOutput, error)
	GetEndpointAttributes(*sns.GetEndpointAttributesInput) (*sns.GetEndpointAttributesOutput, error)
	SetEndpointAttributes(*sns.SetEndpointAttributesInput) (*sns.SetEndpointAttributesOutput, error)
	Publish(*sns.PublishInput) (*sns.PublishOutput, error)
}

// Endpoint is the AWS SNS representation of a Device.
type Endpoint struct {
	ARN   string
	Token string
}

// Platform of a device.
type Platform uint8

// EndpointCreateFunc registers a new device endpoint for the given platform
// and token.
type EndpointCreateFunc func(platformARN, token string) (*Endpoint, error)

// EndpointCreate registers a new device endpoint for the given platform and
// token.
func EndpointCreate(api API) EndpointCreateFunc {
	return func(platformARN, token string) (*Endpoint, error) {
		r, err := api.CreatePlatformEndpoint(&sns.CreatePlatformEndpointInput{
			PlatformApplicationArn: aws.String(platformARN),
			Token: aws.String(token),
		})
		if err != nil {
			return nil, err
		}

		return &Endpoint{
			ARN:   *r.EndpointArn,
			Token: token,
		}, nil
	}
}

// EndpointRetrieveFunc returns the Endpoint for the given ARN.
type EndpointRetrieveFunc func(arn string) (*Endpoint, error)

// EndpointRetrieve returns the Endpoint for the given ARN.
func EndpointRetrieve(api API) EndpointRetrieveFunc {
	return func(arn string) (*Endpoint, error) {
		r, err := api.GetEndpointAttributes(
			&sns.GetEndpointAttributesInput{
				EndpointArn: aws.String(arn),
			},
		)
		if err != nil {
			if awsErr, ok := err.(awserr.RequestFailure); ok &&
				awsErr.StatusCode() == 404 {
				return nil, ErrEndpointNotFound
			}

			return nil, err
		}

		if *r.Attributes[AttributeEnabled] == "false" {
			return nil, ErrEndpointDisabled
		}

		return &Endpoint{
			ARN:   arn,
			Token: *r.Attributes[AttributeToken],
		}, nil
	}
}

// EndpointUpdateFunc takes a new token and stores it with the Endpoint.
type EndpointUpdateFunc func(arn, token string) (*Endpoint, error)

// EndpointUpdate takes a new token and stores it with the Endpoint.
func EndpointUpdate(api API) EndpointUpdateFunc {
	return func(arn, token string) (*Endpoint, error) {
		_, err := api.SetEndpointAttributes(&sns.SetEndpointAttributesInput{
			Attributes: map[string]*string{
				AttributeToken: aws.String(token),
			},
			EndpointArn: aws.String(arn),
		})
		if err != nil {
			return nil, err
		}

		return &Endpoint{
			ARN:   arn,
			Token: token,
		}, nil
	}
}

// PushAFunc pushes a new notification to the device for the given endpoint ARN.
type PushFunc func(
	platform Platform,
	endpointARN, scheme, urn, message string,
) error

// Push pushes a new notification to the device for the given endpoint ARN.
func Push(api API) PushFunc {
	return func(p Platform, arn, scheme, urn, message string) error {
		fmtMsg := ""

		switch p {
		case PlatformAPNS:
			fmtMsg = msgAPNS
		case PlatformAPNSSandbox:
			fmtMsg = msgAPNSSandbox
		case PlatformGCM:
			fmtMsg = msgGCM
		}

		var (
			u = fmt.Sprintf(fmtURN, scheme, urn)
			m = fmt.Sprintf(fmtMsg, message, u)
		)

		_, err := api.Publish(&sns.PublishInput{
			Message:          aws.String(m),
			MessageStructure: aws.String(structureJSON),
			TargetArn:        aws.String(arn),
		})
		if err != nil {
			if awsErr, ok := err.(awserr.RequestFailure); ok {
				if awsErr.StatusCode() == 400 {
					return ErrDeliveryFailure
				}
			}
		}

		return nil
	}
}
