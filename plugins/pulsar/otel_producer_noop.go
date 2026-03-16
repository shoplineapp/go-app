//go:build !otel
// +build !otel

package pulsar

import ap "github.com/apache/pulsar-client-go/pulsar"

func wrapProducer(producer ap.Producer, _ string) ap.Producer {
	return producer
}
