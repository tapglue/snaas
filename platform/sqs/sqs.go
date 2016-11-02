package sqs

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
)

// Common Attributes.
const (
	AttributeSentAt = "SentAt"
	AttributeAll    = "All"

	FormatSentAt = "2006-01-02 15:04:05.999999999 -0700 MST"

	TypeString = "String"
)

// Common Timeouts.
var (
	TimeoutVisibility int64 = 60
	TimeoutWait       int64 = 10
)

// API bundles common SQS operations.
type API interface {
	DeleteMessage(*sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error)
	GetQueueUrl(*sqs.GetQueueUrlInput) (*sqs.GetQueueUrlOutput, error)
	ReceiveMessage(*sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error)
	SendMessage(*sqs.SendMessageInput) (*sqs.SendMessageOutput, error)
}

// ReceiveMessage given a queue url fetches the latest message with the common
// attributes and timeouts.
func ReceiveMessage(api API, queueURL string) (*sqs.ReceiveMessageOutput, error) {
	return api.ReceiveMessage(&sqs.ReceiveMessageInput{
		MessageAttributeNames: []*string{
			aws.String(AttributeAll),
		},
		QueueUrl:          aws.String(queueURL),
		VisibilityTimeout: aws.Int64(TimeoutVisibility),
		WaitTimeSeconds:   aws.Int64(TimeoutWait),
	})
}

// MessageInput given a queue url and a body creates a common message for to be
// sent over SQS.
func MessageInput(body []byte, queueURL string) *sqs.SendMessageInput {
	now := time.Now().Format(FormatSentAt)

	return &sqs.SendMessageInput{
		MessageAttributes: map[string]*sqs.MessageAttributeValue{
			AttributeSentAt: &sqs.MessageAttributeValue{
				DataType:    aws.String(TypeString),
				StringValue: aws.String(now),
			},
		},
		MessageBody: aws.String(string(body)),
		QueueUrl:    aws.String(queueURL),
	}
}
