//go:build pulsar
// +build pulsar

package pulsar

import "go.opentelemetry.io/otel/propagation"

// PulsarMessageCarrier adapts Pulsar message properties to the propagation.TextMapCarrier interface.
type PulsarMessageCarrier map[string]string

var _ propagation.TextMapCarrier = PulsarMessageCarrier{}

func (c PulsarMessageCarrier) Get(key string) string {
	return c[key]
}

func (c PulsarMessageCarrier) Set(key, val string) {
	c[key] = val
}

func (c PulsarMessageCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}
