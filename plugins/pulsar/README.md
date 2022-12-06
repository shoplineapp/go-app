# Pulsar

Pulsar consumer and producer with common handling, client management and gracefully shutdown

## Usage

Require build tag: `pulsar`

### Producer

```golang
package main

import (
	go_app "github.com/shoplineapp/go-app"
	"github.com/shoplineapp/go-app/plugins/pulsar"
)

func main() {
  app := go_app.NewApplication()
  app.Run(func(
	logger *logger.Logger,
	p *pulsar_plugin.PulsarServer,
	pm *pulsar_plugin.PulsarProducerManager,
  ) {
	p.Connect("pulsar://broker.pulsar.com:8501")

	// Use AddProducer if you want to reuse the producer client
	// Otherwise use pm.CreateProducer instead
	producer, err := pm.AddProducer(
		pulsar_plugin.WithProducerLabel("some_events"),
		pulsar_plugin.WithProducerTopic("persistent://my_tenant/my_namespace/my_action"),
	)

	if err != nil {
		logger.Error("Unable to create producer", err)
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, "trace_id", uuid.New().String())

	// Send message to producer with trace info
	// it will include trace id from context, producer hostname and ip for source tracing
	properties := producer.TapTraceProperties(ctx, map[string]string{})

	// Works with native ProducerMessage
	producer.SendAsync(ctx, &pulsar.ProducerMessage{
		Payload:    []byte("hello"),
		EventTime:  time.Now(),
		Properties: properties,
	}, func(id pulsar.MessageID, message *pulsar.ProducerMessage, err error) {
		if err != nil {
			logger.Error("Unable to send message", err)
		}
	})
  })
}
```

