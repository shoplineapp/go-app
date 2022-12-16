package pulsar

import (
	"time"

	ap "github.com/apache/pulsar-client-go/pulsar"
)

const (
	DefaultDLQReceives = 10
)

var (
	DefaultDLQMaxReconnect = uint(5)
)

func DefaultDLQPolicy(producerOptions *ap.ProducerOptions) *ap.DLQPolicy {
	options := producerOptions
	if options == nil {
		options = &ap.ProducerOptions{
			SendTimeout:          3 * time.Second,
			MaxReconnectToBroker: &DefaultDLQMaxReconnect,
		}
	}
	return &ap.DLQPolicy{
		MaxDeliveries:   DefaultDLQReceives,
		ProducerOptions: *options,
	}
}
