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

### Consumer

Define a consumer handler, gracefully shutdown of consumers and auto-ack/nack is included.

```golang
type EmailSender struct {
	pulsar_plugin.PulsarConsumerInterface
}

// Identifier of the consumer 
func (c *EmailSender) Label() string {
	return "EmailSender"
}

// Return a string, a slice of string or a regexp, it will be mapped into the consumer options
func (c *EmailSender) Topic() interface{} {
	return "persistent://my_tenant/my_namespace/my_action"
}

// Modify native consumer options config
func (c *EmailSender) ConsumerOptions() *pulsar.ConsumerOptions {
	return &pulsar.ConsumerOptions{
		SubscriptionName: "email-service-notification-subscription",
		Type:             pulsar.Shared,
	}
}

// On receive message from consumer
func (c *EmailSender) Receive(ctx context.Context, msg pulsar.ConsumerMessage) error {
	data := map[string]interface{}{}
	json.Unmarshal(msg.Payload(), &data)
	fmt.Println("Received message", data)
	err := SendEmail(data)

	// Consumer will automatically ack or nack based on the err return
	return err
}
```

Add consumer handler into the consumer manager

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

		_, err := cm.AddConsumer(&EmailSender{})
		if err != nil {
			logger.Fatal(err)
		}
	})
}
```

Then you will see the consumer running and ready to consume message

```
INFO[0000] PROVIDE plugin *env.Env                      
INFO[0000] PROVIDE plugin *logger.Logger                
INFO[0000] PROVIDE plugin *pulsar.PulsarServer          
INFO[0000] PROVIDE plugin *pulsar.PulsarProducerManager 
INFO[0000] PROVIDE plugin *pulsar.PulsarConsumerManager 
INFO[0000] PROVIDE plugin fx.Lifecycle                  
INFO[0000] PROVIDE plugin fx.Shutdowner                 
INFO[0000] PROVIDE plugin fx.DotGraph                   
INFO[0000] Pulsar client configured                     
INFO[0000] Running                                      
INFO[0000] Connecting to broker                          remote_addr="pulsar://broker.pulsar.com:8501"
INFO[0000] TCP connection established                    local_addr="192.168.241.19:61129" remote_addr="pulsar://broker.pulsar.com:8501"
INFO[0000] Connection is ready                           local_addr="192.168.241.19:61129" remote_addr="pulsar://broker.pulsar.com:8501"
INFO[0000] Connecting to broker                          remote_addr="pulsar://broker.pulsar.com:8501"
INFO[0000] TCP connection established                    local_addr="192.168.241.19:61130" remote_addr="pulsar://broker.pulsar.com:8501"
INFO[0000] Connection is ready                           local_addr="192.168.241.19:61130" remote_addr="pulsar://broker.pulsar.com:8501"
INFO[0000] Connecting to broker                          remote_addr="pulsar://10.55.2.131:8501"
INFO[0000] TCP connection established                    local_addr="192.168.241.19:61131" remote_addr="pulsar://10.55.2.131:8501"
INFO[0000] Connection is ready                           local_addr="192.168.241.19:61131" remote_addr="pulsar://10.55.2.131:8501"
INFO[0000] Connected consumer                            consumerID=1 name=qlrag subscription=philip-local topic="persistent://my_tenant/my_namespace/my_action"
INFO[0000] Created consumer                              consumerID=1 name=qlrag subscription=philip-local topic="persistent://my_tenant/my_namespace/my_action"
INFO[0000] Consumer added                                consumer="map[consumer_label:TestConsumer consumer_topic:persistent://my_tenant/my_namespace/my_action]"
INFO[0000] Ready to consumer message                     consumer="map[consumer_label:TestConsumer consumer_topic:persistent://my_tenant/my_namespace/my_action]"
INFO[0000] Application RUNNING
```