package sns

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/sns"
)

// Message Attributes.
const (
	AttributeEnabled            = "Enabled"
	AttributeToken              = "Token"
	AttributeEndpointCreated    = "EventEndpointCreated"
	AttributeEndpointDeleted    = "EventEndpointDeleted"
	AttributeEndpointUpdated    = "EventEndpointUpdated"
	AttributeDeliveryFailure    = "EventDeliveryFailure"
	AttributePlatformCredential = "PlatformCredential"
	AttributePlatformPrincipal  = "PlatformPrincipal"
)

// Message Types.
const (
	TypeDeliveryFailure = "DeliveryFailure"
	TypeNotification    = "Notification"
)

// Platform supported by SNS for push.
const (
	PlatformAPNSSandbox Platform = iota + 1
	PlatformAPNS
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

// PlatformIdentifiers helps to map Platfrom to human-readable strings.
var PlatformIdentifiers = map[Platform]string{
	PlatformAPNS:        "APNS",
	PlatformAPNSSandbox: "APNS_SANDBOX",
	PlatformGCM:         "GCM",
}

// API bundles common SNS interactions in a reasonably sized interface.
type API interface {
	CreatePlatformApplication(*sns.CreatePlatformApplicationInput) (*sns.CreatePlatformApplicationOutput, error)
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

// AppCreateAPNSFunc creates a new platform application for APNS production.
type AppCreateAPNSFunc func(name, cert, key string) (string, error)

// AppCreateAPNS creates a new platform application for APNS production.
func AppCreateAPNS(api API, changeTopic string) AppCreateAPNSFunc {
	return func(name, cert, key string) (string, error) {
		return createPlatformApp(
			api,
			PlatformAPNS,
			name,
			changeTopic,
			map[string]*string{
				AttributePlatformCredential: aws.String(key),
				AttributePlatformPrincipal:  aws.String(cert),
			},
		)
	}
}

// AppCreateAPNSSandboxFunc creates a new platform application for APNS sandbox.
type AppCreateAPNSSandboxFunc func(name, cert, key string) (string, error)

// AppCreateAPNSSandbox creates a new platform application for APNS sandbox.
func AppCreateAPNSSandbox(api API, changeTopic string) AppCreateAPNSSandboxFunc {
	return func(name, cert, key string) (string, error) {
		return createPlatformApp(
			api,
			PlatformAPNSSandbox,
			name,
			changeTopic,
			map[string]*string{
				AttributePlatformCredential: aws.String(key),
				AttributePlatformPrincipal:  aws.String(cert),
			},
		)
	}
}

// AppCreateGCMFunc creates a new platform application for GCM.
type AppCreateGCMFunc func(name, key string) (string, error)

// AppCreateGCM creates a new platform application for GCM.
func AppCreateGCM(api API, changeTopic string) AppCreateGCMFunc {
	return func(name, key string) (string, error) {
		return createPlatformApp(
			api,
			PlatformGCM,
			name,
			changeTopic,
			map[string]*string{
				AttributePlatformCredential: aws.String(key),
			},
		)
	}
}

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

// PushFunc pushes a new notification to the device for the given endpoint ARN.
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

func createPlatformApp(
	api API,
	platform Platform,
	name, topic string,
	attr map[string]*string,
) (string, error) {
	attr[AttributeEndpointCreated] = aws.String(topic)
	attr[AttributeEndpointDeleted] = aws.String(topic)
	attr[AttributeEndpointUpdated] = aws.String(topic)
	attr[AttributeDeliveryFailure] = aws.String(topic)

	res, err := api.CreatePlatformApplication(&sns.CreatePlatformApplicationInput{
		Attributes: attr,
		Name:       aws.String(name),
		Platform:   aws.String(PlatformIdentifiers[platform]),
	})
	if err != nil {
		return "", err
	}

	return *res.PlatformApplicationArn, nil
}
