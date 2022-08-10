//go:build sqs && sqs_worker
// +build sqs,sqs_worker

package sqs_worker

import "github.com/shoplineapp/go-app/plugins/sqs"

type EventHandlerInterface interface {
	Topic() sqs.Topic

	// OnEvent message is raw string
	OnEvent(topic *sqs.Topic, message string) error

	// OnError return true means ignore error and delete message, otherwise keep message on queue
	OnError(topic *sqs.Topic, err error) bool
}
