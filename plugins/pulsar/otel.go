//go:build pulsar
// +build pulsar

package pulsar

const tracerName = "github.com/shoplineapp/go-app/plugins/pulsar"

type PulsarMessageCarrier map[string]string

func (c PulsarMessageCarrier) Get(key string) string {
	return c[key]
}

func (c PulsarMessageCarrier) Set(key string, value string) {
	c[key] = value
}

func (c PulsarMessageCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}
