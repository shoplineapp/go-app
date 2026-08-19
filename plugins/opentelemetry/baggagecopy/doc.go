// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package baggagecopy provides an OpenTelemetry span processor that copies
// baggage members from a span's parent context onto the span as attributes.
//
// This package is based on go.opentelemetry.io/contrib/processors/baggagecopy
// v0.11.0 and contains SHOPLINE-specific modifications.
//
// Do not put sensitive information in baggage.
package baggagecopy
