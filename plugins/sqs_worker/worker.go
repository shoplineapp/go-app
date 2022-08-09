//go:build sqs && sqs_worker

package sqs_worker

import (
	"context"
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	aws_sqs "github.com/aws/aws-sdk-go/service/sqs"
	"github.com/shoplineapp/go-app/plugins"
	"github.com/shoplineapp/go-app/plugins/logger"
	"github.com/shoplineapp/go-app/plugins/sqs"
	log "github.com/sirupsen/logrus"
	"go.uber.org/fx"
	"sync"
)

type AwsSqsWorker struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     *sync.WaitGroup

	topicMgr *sqs.AwsTopicManager
	logger   *logger.Logger
	handler  EventHandlerInterface

	enabled bool
	started bool
}

type awsMessage struct {
	topicName string
	*aws_sqs.Message
}

func init() {
	plugins.Registry = append(plugins.Registry, NewAwsSqsWorker)
}

func (w *AwsSqsWorker) SetRegion(region string) {
	w.topicMgr.SetRegion(region)
}

func (w *AwsSqsWorker) Serve() {

	topic := w.topicMgr.GetTopic(w.handler.Topic().Name)

	if topic == nil {
		w.logger.WithFields(log.Fields{"topic": w.handler.Topic().Name, "queueArn": w.handler.Topic().Arn}).Error("topic not found, stop worker")
		w.Shutdown()
		return
	}

	maxMsg := *w.getReceiveInput(topic).MaxNumberOfMessages
	chnMessages := make(chan []*awsMessage, maxMsg)

	go func(wg *sync.WaitGroup) {
		wg.Add(1)
		w.messageLoop(chnMessages, topic)
		wg.Done()
	}(w.wg)

	w.logger.WithFields(log.Fields{"topic": topic.Name, "queueArn": topic.Arn}).Info("Listening on SQS queue")

	go func(ctx context.Context, wg *sync.WaitGroup) {
		wg.Add(1)
		for {
			select {
			case msg := <-chnMessages:
				w.handleMessage(msg)
			case <-ctx.Done():
				w.logger.Info("shutdown handle message goroutine")
				wg.Done()
				return
			default:
			}
		}
	}(w.ctx, w.wg)

	w.started = true
}

func (w *AwsSqsWorker) messageLoop(chn chan<- []*awsMessage, topic *sqs.Topic) {
	for w.enabled {
		output, err := topic.ReceiveMessage(w.getReceiveInput(topic))

		if err != nil {
			w.logger.WithFields(log.Fields{"error": err}).Error("Failed to fetch sqs message")
			continue
		}

		var messages []*awsMessage
		for _, message := range output.Messages {
			messages = append(messages, &awsMessage{
				topicName: topic.Name,
				Message:   message,
			})
		}

		chn <- messages
	}
}

func (w *AwsSqsWorker) handleMessage(messages []*awsMessage) {
	var wg sync.WaitGroup
	for _, message := range messages {
		msg := message
		wg.Add(1)

		go func(awsMsg *awsMessage, wg *sync.WaitGroup) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					err := r.(error)
					w.logger.WithFields(log.Fields{"message": *awsMsg.Body, "error": err}).Error("Failed to invoke event")
				}
			}()

			w.logger.WithFields(log.Fields{"message": *awsMsg.Body}).Debug("Message received")

			topic := w.topicMgr.GetTopic(awsMsg.topicName)
			deleteMessage := true
			err := w.handler.OnEvent(topic, *awsMsg.Body)
			if err != nil {
				deleteMessage = w.handler.OnError(topic, err)
			}

			if deleteMessage {
				deleteMessageInput := aws_sqs.DeleteMessageInput{QueueUrl: &topic.Arn, ReceiptHandle: awsMsg.ReceiptHandle}
				_, err = topic.DeleteMessage(&deleteMessageInput)
				if err != nil {
					w.logger.WithFields(log.Fields{"message": *awsMsg.Body, "error": err}).Error("Fail to delete message")
				}
			}
		}(msg, &wg)
	}

	wg.Wait()
}

func (w *AwsSqsWorker) getReceiveInput(topic *sqs.Topic) *aws_sqs.ReceiveMessageInput {
	input := topic.AwsReceiveMessageInput
	if input == nil {
		input = &aws_sqs.ReceiveMessageInput{
			AttributeNames: []*string{
				aws.String(aws_sqs.MessageSystemAttributeNameSentTimestamp),
			},
			MessageAttributeNames: []*string{
				aws.String(aws_sqs.QueueAttributeNameAll),
			},
			MaxNumberOfMessages: aws.Int64(10),
			VisibilityTimeout:   aws.Int64(20), // 20 seconds
			WaitTimeSeconds:     aws.Int64(0),
		}
	}

	input.QueueUrl = &topic.Arn
	return input
}

func (w *AwsSqsWorker) Shutdown() {
	w.enabled = false
	w.started = false
	w.cancel()
	w.wg.Wait()
	w.logger.Info("Shutdown Complete")
}

func (w *AwsSqsWorker) Register(handlerInterface EventHandlerInterface) (topic *sqs.Topic, err error) {
	if w.started {
		err = errors.New("not allow register after Serve")
		return
	}

	if w.topicMgr.Region == "" {
		err = errors.New("region is empty")
		return
	}

	t := handlerInterface.Topic()
	topic = w.topicMgr.AddTopic(t.Name, t.Arn)
	w.handler = handlerInterface

	return
}

func NewAwsSqsWorker(lc fx.Lifecycle, topicMgr *sqs.AwsTopicManager, logger *logger.Logger) *AwsSqsWorker {
	ctx, cancel := context.WithCancel(context.Background())
	worker := &AwsSqsWorker{
		ctx:      ctx,
		cancel:   cancel,
		wg:       new(sync.WaitGroup),
		topicMgr: topicMgr,
		logger:   logger,

		enabled: true,
		started: false,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			worker.Serve()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			worker.Shutdown()
			return nil
		},
	})

	return worker
}
