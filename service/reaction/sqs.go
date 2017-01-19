package reaction

import (
	"encoding/json"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	serr "github.com/tapglue/snaas/error"
	platformSQS "github.com/tapglue/snaas/platform/sqs"
)

const (
	queueName = "reaction-state-change"
)

type sqsSource struct {
	api      platformSQS.API
	queueURL string
}

// SQSSource returns an SQS backed Source implementation.
func SQSSource(api platformSQS.API) (Source, error) {
	res, err := api.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	})
	if err != nil {
		return nil, err
	}

	return &sqsSource{
		api:      api,
		queueURL: *res.QueueUrl,
	}, nil
}

func (s *sqsSource) Ack(id string) error {
	_, err := s.api.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      aws.String(s.queueURL),
		ReceiptHandle: aws.String(id),
	})

	return err
}

func (s *sqsSource) Consume() (*StateChange, error) {
	o, err := platformSQS.ReceiveMessage(s.api, s.queueURL)
	if err != nil {
		return nil, err
	}

	if len(o.Messages) == 0 {
		return nil, serr.ErrEmptySource
	}

	var (
		m = o.Messages[0]

		sentAt time.Time
	)

	if attr, ok := m.MessageAttributes[platformSQS.AttributeSentAt]; ok {
		t, err := time.Parse(platformSQS.FormatSentAt, *attr.StringValue)
		if err != nil {
			return nil, err
		}

		sentAt = t
	}

	f := sqsStateChange{}

	err = json.Unmarshal([]byte(*m.Body), &f)
	if err != nil {
		return nil, err
	}

	return &StateChange{
		AckID:     *m.ReceiptHandle,
		ID:        *m.MessageId,
		Namespace: f.Namespace,
		New:       f.New,
		Old:       f.Old,
		SentAt:    sentAt,
	}, nil
}

func (s *sqsSource) Propagate(ns string, old, new *Reaction) (string, error) {
	r, err := json.Marshal(&sqsStateChange{
		Namespace: ns,
		New:       new,
		Old:       old,
	})
	if err != nil {
		return "", err
	}

	o, err := s.api.SendMessage(platformSQS.MessageInput(r, s.queueURL))
	if err != nil {
		return "", err
	}

	return *o.MessageId, nil
}

type sqsStateChange struct {
	Namespace string    `json:"namespace"`
	New       *Reaction `json:"new"`
	Old       *Reaction `json:"old"`
}
