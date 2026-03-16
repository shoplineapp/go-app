//go:build otel
// +build otel

package pulsar

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPulsarMessageCarrier_Get_ExistingKey(t *testing.T) {
	carrier := PulsarMessageCarrier{"traceparent": "00-abc-def-01"}
	assert.Equal(t, "00-abc-def-01", carrier.Get("traceparent"))
}

func TestPulsarMessageCarrier_Get_MissingKey(t *testing.T) {
	carrier := PulsarMessageCarrier{}
	assert.Equal(t, "", carrier.Get("missing"))
}

func TestPulsarMessageCarrier_Set(t *testing.T) {
	carrier := PulsarMessageCarrier{}
	carrier.Set("traceparent", "00-abc-def-01")
	assert.Equal(t, "00-abc-def-01", carrier["traceparent"])
}

func TestPulsarMessageCarrier_Keys(t *testing.T) {
	carrier := PulsarMessageCarrier{"a": "1", "b": "2"}
	keys := carrier.Keys()
	sort.Strings(keys)
	assert.Equal(t, []string{"a", "b"}, keys)
}
