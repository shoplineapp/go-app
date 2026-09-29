//go:build pulsar
// +build pulsar

// Shared test helpers for plugins/pulsar's OTel test files. Kept in
// a dedicated file (rather than duplicated across otel_producer_test.go
// and otel_consumer_test.go) because the helpers are package-private
// and used by multiple test files in this package.

package pulsar

import (
	"context"
	"testing"

	ap "github.com/apache/pulsar-client-go/pulsar"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// installTestTracer wires an in-memory SpanRecorder + W3C TraceContext
// propagator into the OTel globals for the duration of t, restoring
// the previous globals on cleanup. Returns the recorder so the test
// can assert on emitted spans.
func installTestTracer(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()

	prevTP := otel.GetTracerProvider()
	prevProp := otel.GetTextMapPropagator()

	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
		otel.SetTracerProvider(prevTP)
		otel.SetTextMapPropagator(prevProp)
	})
	return sr
}

// fakeMessageID is a minimal ap.MessageID implementation for tests.
// The interface only requires Serialize, LedgerID, EntryID, BatchIdx,
// PartitionIdx; the rest are unused.
type fakeMessageID struct {
	id         string
	partition  int32
	ledger     int64
	entry      int64
	batchIdx   int32
	serialized []byte
}

func (f fakeMessageID) Serialize() []byte   { return f.serialized }
func (f fakeMessageID) LedgerID() int64     { return f.ledger }
func (f fakeMessageID) EntryID() int64      { return f.entry }
func (f fakeMessageID) BatchIdx() int32     { return f.batchIdx }
func (f fakeMessageID) PartitionIdx() int32 { return f.partition }

// attrsToMap flattens attribute KVs to a map keyed by attribute.Key
// with stringified values. Used to assert on emitted attributes.
func attrsToMap(kvs []attribute.KeyValue) map[attribute.Key]string {
	out := make(map[attribute.Key]string, len(kvs))
	for _, kv := range kvs {
		out[kv.Key] = kv.Value.Emit()
	}
	return out
}

func mustAttr(t *testing.T, m map[attribute.Key]string, key, want string) {
	t.Helper()
	got, ok := m[attribute.Key(key)]
	require.Truef(t, ok, "missing attribute %q (have: %v)", key, m)
	assert.Equal(t, want, got, "attribute %q value", key)
}

// mustAttrPresent asserts the attribute key is set and its value is
// non-empty. Use for attributes whose exact value is brittle to assert
// (e.g. messaging.message.id is fmt.Sprintf("%v", id) on the concrete
// ap.MessageID, which is Go's default struct print) — presence is
// what matters for spec compliance.
func mustAttrPresent(t *testing.T, m map[attribute.Key]string, key string) {
	t.Helper()
	got, ok := m[attribute.Key(key)]
	require.Truef(t, ok, "missing attribute %q (have: %v)", key, m)
	assert.NotEmptyf(t, got, "attribute %q must be non-empty", key)
}

// mustAttrAbsent asserts the attribute key is NOT present. Use to
// guard conditional attributes (e.g.
// messaging.destination.partition.id is only emitted for partitioned
// topics).
func mustAttrAbsent(t *testing.T, m map[attribute.Key]string, key string) {
	t.Helper()
	if got, ok := m[attribute.Key(key)]; ok {
		assert.Failf(t, "unexpected attribute present", "key=%q value=%q (have: %v)", key, got, m)
	}
}

// Ensure fakeMessageID satisfies the ap.MessageID interface.
var _ ap.MessageID = fakeMessageID{}
