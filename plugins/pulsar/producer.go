//go:build pulsar
// +build pulsar

package pulsar

import (
	"context"
	"errors"
	"strings"
	"sync"

	ap "github.com/apache/pulsar-client-go/pulsar"
	"github.com/shoplineapp/go-app/common"
	"github.com/shoplineapp/go-app/plugins/logger"
	"go.uber.org/fx"
)

type WithPulsarProducerOption func(*ap.ProducerOptions)

type PulsarProducer struct {
	ap.Producer
	label           string
	topic           string
	producerOptions *ap.ProducerOptions
}

type PulsarProducerManager struct {
	logger       *logger.Logger
	pulsarServer *PulsarServer
	producers    map[string]*PulsarProducer

	mu sync.Mutex
}

type PulsarProducerOption func(*PulsarProducer)

func WithProducerLabel(label string) PulsarProducerOption {
	return func(p *PulsarProducer) {
		p.label = label
	}
}

func WithProducerTopic(topic string) PulsarProducerOption {
	return func(p *PulsarProducer) {
		p.topic = topic
	}
}

func WithProducerOptions(opts ap.ProducerOptions) PulsarProducerOption {
	return func(p *PulsarProducer) {
		p.producerOptions = &opts
	}
}

func (pm *PulsarProducerManager) TraceInfo(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"producers": pm.producers,
	}
}

func (pm *PulsarProducerManager) Producer(label string) *PulsarProducer {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.producers[label]
}

// Add producer into the manager pool to keeping the producer alive.
// Use original CreateProducer if you want to fire an one-off message
func (pm *PulsarProducerManager) AddProducer(opts ...PulsarProducerOption) (*PulsarProducer, error) {
	if pm.pulsarServer.Client == nil {
		return nil, errors.New("pulsar is not connected")
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	p := &PulsarProducer{}
	for _, opt := range opts {
		opt(p)
	}
	if p.topic == "" {
		return nil, errors.New("topic is required")
	}
	if p.label == "" {
		return nil, errors.New("producer label is required")
	}

	if p.producerOptions == nil {
		p.producerOptions = &ap.ProducerOptions{}
	}

	p.producerOptions.Topic = p.topic

	// producerOpts.Topic = topic
	producer, err := pm.pulsarServer.CreateProducer(*p.producerOptions)
	if err != nil {
		return nil, err
	}
	p.Producer = producer

	pm.producers[p.label] = p
	return p, nil
}

func (pm *PulsarProducerManager) Shutdown() {
	if pm.pulsarServer.Client == nil {
		return
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	for _, producer := range pm.producers {
		producer.Close()
	}
}

func (p *PulsarProducer) TapTraceProperties(ctx context.Context, properties map[string]string) map[string]string {
	sb := strings.Builder{}
	if p.label != "" {
		sb.WriteString(p.label)
		sb.WriteString("_")
	}
	sb.WriteString("producer")

	traceID := common.GetTraceID(ctx)
	properties = common.MergeMap(properties, map[string]string{
		"name":       sb.String(),
		"host":       common.GetHostname(),
		"ip_address": common.GetInstanceIP(),
		"trace_id":   traceID,
	})

	return properties
}

func NewPulsarProducerManager(
	lc fx.Lifecycle,
	logger *logger.Logger,
	pulsarServer *PulsarServer,
) *PulsarProducerManager {
	pm := &PulsarProducerManager{
		logger:       logger,
		pulsarServer: pulsarServer,
		producers:    map[string]*PulsarProducer{},
	}
	if lc != nil {
		lc.Append(fx.Hook{
			OnStop: func(ctx context.Context) error {
				pm.Shutdown()
				return nil
			},
		})
	}
	return pm
}
