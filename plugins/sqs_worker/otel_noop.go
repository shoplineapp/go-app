//go:build sqs && sqs_worker && !otel
// +build sqs,sqs_worker,!otel

package sqs_worker

// No-op implementation: uses the default processHook defined in worker.go.
