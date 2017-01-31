package main

import (
	"encoding/json"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"

	"github.com/tapglue/snaas/core"
	serr "github.com/tapglue/snaas/error"
	"github.com/tapglue/snaas/platform/sns"
	"github.com/tapglue/snaas/platform/source"
	platformSQS "github.com/tapglue/snaas/platform/sqs"
	"github.com/tapglue/snaas/service/app"
	"github.com/tapglue/snaas/service/connection"
	"github.com/tapglue/snaas/service/event"
	"github.com/tapglue/snaas/service/object"
	"github.com/tapglue/snaas/service/reaction"
	"github.com/tapglue/snaas/service/rule"
)

type ackFunc func() error

type batch struct {
	ackFunc  ackFunc
	app      *app.App
	messages core.Messages
}

func consumeConnection(
	appFetch core.AppFetchFunc,
	conSource connection.Source,
	batchc chan<- batch,
	pipeline core.PipelineConnectionFunc,
	rules core.RuleListActiveFunc,
) error {
	for {
		change, err := conSource.Consume()
		if err != nil {
			if connection.IsEmptySource(err) {
				continue
			}
			return err
		}

		currentApp, err := appForNamespace(appFetch, change.Namespace)
		if err != nil {
			return err
		}

		rs, err := rules(currentApp, rule.TypeConnection)
		if err != nil {
			return err
		}

		ms, err := pipeline(currentApp, change, rs...)
		if err != nil {
			return err
		}

		if len(ms) == 0 {
			err := conSource.Ack(change.AckID)
			if err != nil {
				return err
			}

			continue
		}

		batchc <- batchMessages(currentApp, conSource, change.AckID, ms)
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
	pipeline core.PipelineEventFunc,
	rules core.RuleListActiveFunc,
) error {
	for {
		change, err := eventSource.Consume()
		if err != nil {
			if event.IsEmptySource(err) {
				continue
			}
			return err
		}

		currentApp, err := appForNamespace(appFetch, change.Namespace)
		if err != nil {
			return err
		}

		rs, err := rules(currentApp, rule.TypeEvent)
		if err != nil {
			return err
		}

		ms, err := pipeline(currentApp, change, rs...)
		if err != nil {
			return err
		}

		if len(ms) == 0 {
			err = eventSource.Ack(change.AckID)
			if err != nil {
				return err
			}

			continue
		}

		batchc <- batchMessages(currentApp, eventSource, change.AckID, ms)
	}
}

func consumeReaction(
	appFetch core.AppFetchFunc,
	reactionSource reaction.Source,
	batchc chan<- batch,
	pipeline core.PipelineReactionFunc,
	rules core.RuleListActiveFunc,
) error {
	for {
		change, err := reactionSource.Consume()
		if err != nil {
			if serr.IsEmptySource(err) {
				continue
			}
			return err
		}

		currentApp, err := appForNamespace(appFetch, change.Namespace)
		if err != nil {
			return err
		}

		rs, err := rules(currentApp, rule.TypeReaction)
		if err != nil {
			return err
		}

		ms, err := pipeline(currentApp, change, rs...)
		if err != nil {
			return err
		}

		if len(ms) == 0 {
			err = reactionSource.Ack(change.AckID)
			if err != nil {
				return err
			}

			continue
		}

		batchc <- batchMessages(currentApp, reactionSource, change.AckID, ms)
	}
}

func consumeObject(
	appFetch core.AppFetchFunc,
	objectSource object.Source,
	batchc chan<- batch,
	pipeline core.PipelineObjectFunc,
	rules core.RuleListActiveFunc,
) error {
	for {
		change, err := objectSource.Consume()
		if err != nil {
			if object.IsEmptySource(err) {
				continue
			}
			return err
		}

		currentApp, err := appForNamespace(appFetch, change.Namespace)
		if err != nil {
			return err
		}

		rs, err := rules(currentApp, rule.TypeObject)
		if err != nil {
			return err
		}

		ms, err := pipeline(currentApp, change, rs...)
		if err != nil {
			return err
		}

		if len(ms) == 0 {
			err := objectSource.Ack(change.AckID)
			if err != nil {
				return err
			}

			continue
		}

		batchc <- batchMessages(currentApp, objectSource, change.AckID, ms)
	}
}

func batchMessages(
	currentApp *app.App,
	acker source.Acker,
	ackID string,
	ms core.Messages,
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
