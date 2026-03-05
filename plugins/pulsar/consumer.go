package pulsar

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	ap "github.com/apache/pulsar-client-go/pulsar"
	"github.com/shoplineapp/go-app/common"
	"github.com/shoplineapp/go-app/plugins/logger"
	"github.com/sirupsen/logrus"
	"go.uber.org/fx"
)

type ConsumerTopic interface {
	string | []string | regexp.Regexp
}

type PulsarConsumerInterface interface {
	Label() string

	// Return a string, a slice of string or a regexp
	Topic() interface{}
	ConsumerOptions() *ap.ConsumerOptions
	Receive(ctx context.Context, msg ap.ConsumerMessage) error
}

type PulsarConsumer struct {
	ap.Consumer
	options *ap.ConsumerOptions
	Handler PulsarConsumerInterface
}

type PulsarConsumerManager struct {
	logger       *logger.Logger
	pulsarServer *PulsarServer
	consumers    map[string]*PulsarConsumer
	IsStopped    bool

	ctx       context.Context
	ctxCancel context.CancelFunc
	wg        *sync.WaitGroup
	mu        sync.Mutex
}

type PulsarConsumerManagerParams struct {
	fx.In

	Lifecycle    fx.Lifecycle `optional:"true"`
	Logger       *logger.Logger
	PulsarServer *PulsarServer
}

func (c *PulsarConsumer) TraceInfo() map[string]string {
	return map[string]string{
		"consumer_label": c.Handler.Label(),
		"consumer_topic": fmt.Sprintf("%+v", c.Handler.Topic()),
	}
}

func (cm *PulsarConsumerManager) AddConsumer(consumer PulsarConsumerInterface) (*PulsarConsumer, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	c := &PulsarConsumer{
		Handler: consumer,
	}

	options := c.Handler.ConsumerOptions()
	if options == nil {
		options = &ap.ConsumerOptions{}
	}

	switch consumer.Topic().(type) {
	case string:
		options.Topic = consumer.Topic().(string)
	case []string:
		options.Topics = consumer.Topic().([]string)
	case *regexp.Regexp:
		options.TopicsPattern = consumer.Topic().(*regexp.Regexp).String()
	default:
		return nil, errors.New("Topic must be a string, a slice of string or a regexp")
	}

	if options.Topic == "" && options.Topics == nil && options.TopicsPattern == "" {
		return nil, errors.New("Topic is required")
	}

	// Take over the message channel for gracefully shutdown
	options.MessageChannel = make(chan ap.ConsumerMessage)

	if options.BackoffPolicy == nil {
		options.BackoffPolicy = DefaultBackoffPolicy
	}

	// Add default subscription name
	if options.SubscriptionName == "" {
		sb := strings.Builder{}
		sb.WriteString(strings.ReplaceAll(c.Handler.Label(), "-", "_"))
		sb.WriteString("_subscription")
		options.SubscriptionName = sb.String()
	}

	c.options = options
	pulsarConsumer, err := cm.pulsarServer.Subscribe(*options)
	if err != nil {
		return nil, err
	}
	c.Consumer = pulsarConsumer

	cm.logger.WithFields(logrus.Fields{"consumer": c.TraceInfo()}).Info("Consumer added")
	cm.consumers[consumer.Label()] = c
	return c, nil
}

func (cm *PulsarConsumerManager) onMessageReceive(consumer *PulsarConsumer, msg ap.ConsumerMessage) {
	defer func() {
		if r := recover(); r != nil {
			cm.logger.WithFields(logrus.Fields{
				"consumer": consumer.TraceInfo(),
				"message":  msg,
				"error":    r,
			}).Error("Failed to process message")
		}
	}()

	ctx := context.Background()
	var traceId string
	props := msg.Properties()
	if props != nil {
		traceId = props["trace_id"]
	}

	err := consumer.Handler.Receive(common.NewContextWithTraceID(ctx, traceId), msg)
	if err != nil {
		cm.logger.WithFields(logrus.Fields{"consumer": consumer.TraceInfo(), "error": err, "message": msg}).Error("Failed to process message, response with nack")
		consumer.Consumer.Nack(msg)
		return
	}

	consumer.Consumer.Ack(msg)
}

func (cm *PulsarConsumerManager) Start() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.wg = &sync.WaitGroup{}
	for _, consumer := range cm.consumers {
		cm.wg.Add(1)
		cm.logger.WithFields(logrus.Fields{"consumer": consumer.TraceInfo()}).Info("Ready to consumer message")
		go func(cm *PulsarConsumerManager, c *PulsarConsumer) {
		CONSUMER_LOOP:
			for {
				if cm.IsStopped {
					cm.wg.Done()
					break CONSUMER_LOOP
				}

				select {
				case msg := <-c.Consumer.Chan():
					cm.logger.WithFields(logrus.Fields{"consumer": c.TraceInfo(), "message": msg}).Trace("Received message")
					cm.onMessageReceive(c, msg)
				case <-cm.ctx.Done():
					cm.wg.Done()
					break CONSUMER_LOOP
				}
			}
		}(cm, consumer)
	}
}

func (cm *PulsarConsumerManager) Shutdown() {
	if cm.pulsarServer.Client == nil {
		return
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.IsStopped = true
	cm.ctxCancel()
	cm.wg.Wait()

	for _, consumer := range cm.consumers {
		consumer.Close()
	}
}

func NewPulsarConsumerManager(
	params PulsarConsumerManagerParams,
) *PulsarConsumerManager {
	cm := &PulsarConsumerManager{
		logger:       params.Logger,
		pulsarServer: params.PulsarServer,
		consumers:    map[string]*PulsarConsumer{},
	}
	cm.ctx, cm.ctxCancel = context.WithCancel(context.Background())

	if params.Lifecycle != nil {
		params.Lifecycle.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				cm.Start()
				return nil
			},
			OnStop: func(ctx context.Context) error {
				cm.Shutdown()
				return nil
			},
		})
	}
	return cm
}
