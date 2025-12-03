# Sentry

The base framework for Sentry error tracking and performance monitoring. Build tag `sentry` **MUST** be added to enable this plugin.

## Overview

This plugin integrates Sentry's error tracking and performance monitoring into your Go application. It provides automatic error capture, performance tracing, and detailed error reporting.

## Installation

To build your application with Sentry support, add the `sentry` build tag:

```bash
go build -tags sentry
```

To enable automatic OpenTelemetry trace context propagation, add both `sentry` and `otel` build tags:

```bash
go build -tags "sentry,otel"
```

## Usage

### Basic Usage with Dependency Injection

```golang
package main

import (
  "time"
  
  "github.com/getsentry/sentry-go"
  go_app "github.com/shoplineapp/go-app"
  sentry_plugin "github.com/shoplineapp/go-app/plugins/sentry"
)

func main() {
  app := go_app.NewApplication()
  app.Run(func(
    sentryAgent *sentry_plugin.SentryAgent,
  ) {
    // Initialize Sentry from environment variables
    if err := sentryAgent.Configure(); err != nil {
      panic(err)
    }
    defer sentry.Flush(2 * time.Second)

    // Your application code here
  })
}
```

### Configuration with Custom Options

```golang
package main

import (
  "time"
  
  "github.com/getsentry/sentry-go"
  go_app "github.com/shoplineapp/go-app"
  sentry_plugin "github.com/shoplineapp/go-app/plugins/sentry"
)

func main() {
  app := go_app.NewApplication()
  app.Run(func(
    sentryAgent *sentry_plugin.SentryAgent,
  ) {
    // Configure Sentry with custom options
    err := sentryAgent.Configure(func(opts *sentry.ClientOptions) {
      opts.TracesSampleRate = 1.0
      opts.Release = "v1.0.0"
    })
    if err != nil {
      panic(err)
    }
    defer sentry.Flush(2 * time.Second)

    // Your application code here
  })
}
```

### Capturing Errors and Messages

The SentryAgent provides wrapper methods that automatically add OpenTelemetry trace context:

```golang
package main

import (
  "context"
  "errors"
  
  sentry_plugin "github.com/shoplineapp/go-app/plugins/sentry"
)

func someFunction(ctx context.Context, sentryAgent *sentry_plugin.SentryAgent) {
  // Capture an exception with automatic trace context
  err := doSomething()
  if err != nil {
    eventID := sentryAgent.CaptureException(ctx, err)
    log.Printf("Error captured with ID: %s", *eventID)
  }

  // Capture a message with automatic trace context
  eventID := sentryAgent.CaptureMessage(ctx, "Something important happened")
  log.Printf("Message captured with ID: %s", *eventID)
}
```

### Recovering from Panics

Use `RecoverWithContext` to capture panics with automatic trace context:

```golang
func riskyOperation(ctx context.Context, sentryAgent *sentry_plugin.SentryAgent) {
  defer sentryAgent.RecoverWithContext(ctx)
  
  // Code that might panic
  panic("something went wrong")
}
```

### Using the Sentry Hub for Advanced Features

For advanced usage, you can access the Sentry hub directly:

```golang
func advancedUsage(ctx context.Context, sentryAgent *sentry_plugin.SentryAgent) {
  // Get hub from context (or clone current hub if not found)
  hub := sentryAgent.HubFromContext(ctx)
  
  // Configure scope
  hub.ConfigureScope(func(scope *sentry.Scope) {
    scope.SetUser(sentry.User{
      ID:       "user-123",
      Email:    "user@example.com",
      Username: "john_doe",
    })
    scope.SetTag("environment", "production")
    scope.SetContext("custom", map[string]interface{}{
      "request_id": "req-456",
    })
  })

  // Capture with additional context
  hub.CaptureException(errors.New("something went wrong"))
}
```

## Environment Variables

The plugin supports the following environment variables when using `Configure()` without options:

| Variable | Description | Default |
|----------|-------------|---------|
| `SENTRY_DSN` | Your Sentry project DSN (required) | - |
| `ENVIRONMENT` | Environment name (e.g., production, staging) | - |
| `RELEASE` | Release version or commit hash | - |
| `SENTRY_DEBUG` | Enable debug mode (true/false) | `false` |
| `SENTRY_SEND_DEFAULT_PII` | Send default PII (true/false) | `false` |
| `SENTRY_SAMPLE_RATE` | Sample rate for events (0.0 to 1.0) | `0.0` |

### Example `.env` File

```env
SENTRY_DSN=https://your-public-key@sentry.io/project-id
ENVIRONMENT=production
RELEASE=v1.2.3
SENTRY_DEBUG=false
SENTRY_SEND_DEFAULT_PII=false
SENTRY_SAMPLE_RATE=0.1
```

## API Reference

### SentryAgent

The main agent struct that provides access to Sentry functionality. It is automatically registered and injected via dependency injection.

#### Methods

- **`Configure(opts ...ConfigOption) error`**: Initializes Sentry using environment variables with optional configuration overrides. Returns `ErrSentryNotInitialized` if `SENTRY_DSN` is not set.
- **`CaptureException(ctx context.Context, err error) *sentry.EventID`**: Captures an exception and automatically adds the trace ID to the Sentry event when built with the `otel` tag.
- **`CaptureMessage(ctx context.Context, message string) *sentry.EventID`**: Captures a message and automatically adds the trace ID to the Sentry event when built with the `otel` tag.
- **`Hub() *sentry.Hub`**: Returns the current Sentry hub for advanced usage.
- **`HubFromContext(ctx context.Context) *sentry.Hub`**: Returns the Sentry hub from context, or clones the current hub if not found.
- **`RecoverWithContext(ctx context.Context) *sentry.EventID`**: Recovers from a panic and captures it with Sentry. Automatically adds the trace ID when built with the `otel` tag.

### Errors

- **`ErrSentryNotInitialized`**: Returned when `SENTRY_DSN` environment variable is not set.

### Trace Context Integration

When built with both `sentry` and `otel` build tags, `CaptureException`, `CaptureMessage`, and `RecoverWithContext` automatically add the trace context from the OpenTelemetry span context as Sentry tags:

| Tag | Description |
|-----|-------------|
| `otel.trace_id` | The trace ID from the OpenTelemetry span context |
| `otel.span_id` | The span ID from the OpenTelemetry span context |

This allows you to correlate Sentry errors with distributed traces across your services.

**Note:** When built without the `otel` tag, the trace context methods are no-ops.

### ConfigOption

A function type used to customize Sentry client options:

```golang
type ConfigOption func(*sentry.ClientOptions)
```

Usage example:
```golang
sentryAgent.Configure(func(opts *sentry.ClientOptions) {
  opts.TracesSampleRate = 1.0
  opts.MaxBreadcrumbs = 50
})
```

## Resources

- [Sentry Go SDK Documentation](https://docs.sentry.io/platforms/go/)
- [Sentry Go SDK GitHub](https://github.com/getsentry/sentry-go)
- [Sentry Configuration Options](https://docs.sentry.io/platforms/go/configuration/options/)
