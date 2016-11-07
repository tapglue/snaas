package main

import (
	"encoding/json"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"

	"github.com/tapglue/snaas/core"
	"github.com/tapglue/snaas/platform/sns"
	platformSQS "github.com/tapglue/snaas/platform/sqs"
	"github.com/tapglue/snaas/service/app"
	"github.com/tapglue/snaas/service/connection"
	"github.com/tapglue/snaas/service/event"
	"github.com/tapglue/snaas/service/object"
)

type ackFunc func() error

type batch struct {
	ackFunc  ackFunc
	app      *app.App
	messages messages
}

type message struct {
	message   string
	recipient uint64
	urn       string
}

type messages []*message

func consumeConnection(
	appFetch core.AppFetchFunc,
	conSource connection.Source,
	batchc chan<- batch,
	ruleFns ...conRuleFunc,
) error {
	for {
		c, err := conSource.Consume()
		if err != nil {
			if connection.IsEmptySource(err) {
				continue
			}
			return err
		}

		currentApp, err := appForNamespace(appFetch, c.Namespace)
		if err != nil {
			return err
		}

		ms := messages{}

		for _, rule := range ruleFns {
			msgs, err := rule(currentApp, c)
			if err != nil {
				return err
			}

			for _, msg := range msgs {
				ms = append(ms, msg)
			}
		}

		if len(ms) == 0 {
			err := conSource.Ack(c.AckID)
			if err != nil {
				return err
			}

			continue
		}

		batchc <- batchMessages(currentApp, conSource, c.AckID, ms)
	}
}

func consumeEndpointChange(
	api platformSQS.API,
	queueURL string,
	changec chan endpointChange,
) error {
	for {
		o, err := platformSQS.ReceiveMessage(api, queueURL)
		if err != nil {
			return err
		}

		for _, msg := range o.Messages {
			ack := func() error {
				_, err := api.DeleteMessage(&sqs.DeleteMessageInput{
					QueueUrl:      aws.String(queueURL),
					ReceiptHandle: msg.ReceiptHandle,
				})
				return err
			}

			if msg.Body == nil {
				_ = ack()

				continue
			}

			f := struct {
				Message string `json:"Message"`
				Type    string `json:"Type"`
			}{}

			if err := json.Unmarshal([]byte(*msg.Body), &f); err != nil {
				return err
			}

			if f.Type != sns.TypeNotification {
				_ = ack()

				continue
			}

			c := endpointChange{}

			if err := json.Unmarshal([]byte(f.Message), &c); err != nil {
				return err
			}

			c.ack = ack

			changec <- c
		}
	}
}
func consumeEvent(
	appFetch core.AppFetchFunc,
	eventSource event.Source,
	batchc chan<- batch,
	ruleFns ...eventRuleFunc,
) error {
	for {
		c, err := eventSource.Consume()
		if err != nil {
			if event.IsEmptySource(err) {
				continue
			}
			return err
		}

		currentApp, err := appForNamespace(appFetch, c.Namespace)
		if err != nil {
			return err
		}

		ms := messages{}

		for _, rule := range ruleFns {
			rs, err := rule(currentApp, c)
			if err != nil {
				return err
			}

			for _, msg := range rs {
				ms = append(ms, msg)
			}
		}

		if len(ms) == 0 {
			err = eventSource.Ack(c.AckID)
			if err != nil {
				return err
			}

			continue
		}

		batchc <- batchMessages(currentApp, eventSource, c.AckID, ms)
	}
}

func consumeObject(
	appFetch core.AppFetchFunc,
	objectSource object.Source,
	batchc chan<- batch,
	ruleFns ...objectRuleFunc,
) error {
	for {
		c, err := objectSource.Consume()
		if err != nil {
			if object.IsEmptySource(err) {
				continue
			}
			return err
		}

		currentApp, err := appForNamespace(appFetch, c.Namespace)
		if err != nil {
			return err
		}

		ms := messages{}

		for _, rule := range ruleFns {
			rs, err := rule(currentApp, c)
			if err != nil {
				return err
			}

			for _, msg := range rs {
				ms = append(ms, msg)
			}
		}

		if len(ms) == 0 {
			err := objectSource.Ack(c.AckID)
			if err != nil {
				return err
			}

			continue
		}

		batchc <- batchMessages(currentApp, objectSource, c.AckID, ms)
	}
}

func batchMessages(
	currentApp *app.App,
	acker platformSQS.Acker,
	ackID string,
	ms messages,
) batch {
	return batch{
		ackFunc: func(acked bool, ackID string) ackFunc {
			return func() error {
				if acked {
					return nil
				}

				err := acker.Ack(ackID)
				if err == nil {
					acked = true
				}
				return err
			}
		}(false, ackID),
		app:      currentApp,
		messages: ms,
	}
}
