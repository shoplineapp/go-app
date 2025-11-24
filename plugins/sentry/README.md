# Sentry

The base framework for Sentry error tracking and performance monitoring. Build tag `sentry` **MUST** be added to enable this plugin.

## Overview

This plugin integrates Sentry's error tracking and performance monitoring into your Go application. It provides automatic error capture, performance tracing, and detailed error reporting.

## Installation

To build your application with Sentry support, add the `sentry` build tag:

```bash
go build -tags sentry
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

```golang
package main

import (
  "github.com/getsentry/sentry-go"
  go_app "github.com/shoplineapp/go-app"
  sentry_plugin "github.com/shoplineapp/go-app/plugins/sentry"
)

func someFunction(sentryAgent *sentry_plugin.SentryAgent) {
  hub := sentryAgent.Hub()
  
  // Capture an exception
  err := doSomething()
  if err != nil {
    eventID := hub.CaptureException(err)
    log.Printf("Error captured with ID: %s", eventID)
  }

  // Capture a message
  eventID := hub.CaptureMessage("Something important happened", sentry.LevelInfo)
  log.Printf("Message captured with ID: %s", eventID)
}
```

### Using the Sentry Hub for Advanced Features

```golang
func advancedUsage(sentryAgent *sentry_plugin.SentryAgent) {
  hub := sentryAgent.Hub()
  
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

- **`Hub() *sentry.Hub`**: Returns the current Sentry hub instance for advanced usage and direct access to Sentry operations
- **`Configure(opts ...ConfigOption) error`**: Initializes Sentry using environment variables with optional configuration overrides. Returns nil if `SENTRY_DSN` is not set (logs a warning instead).

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
