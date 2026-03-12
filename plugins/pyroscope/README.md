# Pyroscope

The base framework for Pyroscope continuous profiling. Build tag `pyroscope` **MUST** be added to enable this plugin.

## Overview

This plugin integrates [Grafana Pyroscope](https://grafana.com/oss/pyroscope/) continuous profiling into your Go application. It exposes a `PyroscopeAgent` that can be configured through functional options and is started/stopped automatically via Fx lifecycle hooks.

## Installation

To build your application with Pyroscope support, add the `pyroscope` build tag:

```bash
go build -tags pyroscope
```

## Usage

### Basic Usage with Dependency Injection

```golang
package main

import (
	"os"
	"time"

	pyroscope_go "github.com/grafana/pyroscope-go"
	go_app "github.com/shoplineapp/go-app"
	pyroscope_plugin "github.com/shoplineapp/go-app/plugins/pyroscope"
)

func main() {
	app := go_app.NewApplication()
	app.Run(func(agent *pyroscope_plugin.PyroscopeAgent) {
		err := agent.Configure(
			pyroscope_plugin.WithApplicationName(os.Getenv("PYROSCOPE_APPLICATION_NAME")),
			pyroscope_plugin.WithServerAddress(os.Getenv("PYROSCOPE_SERVER_ADDRESS")),
			pyroscope_plugin.WithProfileTypes([]pyroscope_go.ProfileType{
				pyroscope_go.ProfileCPU,
				pyroscope_go.ProfileAllocObjects,
				pyroscope_go.ProfileAllocSpace,
				pyroscope_go.ProfileInuseObjects,
				pyroscope_go.ProfileInuseSpace,
				pyroscope_go.ProfileGoroutines,
				pyroscope_go.ProfileMutexCount,
				pyroscope_go.ProfileMutexDuration,
				pyroscope_go.ProfileBlockCount,
				pyroscope_go.ProfileBlockDuration,
			}),
			pyroscope_plugin.WithUploadRate(10*time.Second),
		)
		if err != nil {
			panic(err)
		}

		// Your application code here
	})
}
```

## Environment Variables

The plugin itself does not read environment variables directly. A common pattern is reading env vars in application code, then passing values to `Configure`:

| Variable | Description | Required |
|----------|-------------|----------|
| `PYROSCOPE_APPLICATION_NAME` | Pyroscope application name (for grouping profiles) | Yes |
| `PYROSCOPE_SERVER_ADDRESS` | Pyroscope server URL (e.g. `http://pyroscope:4040`) | Yes |
| `PYROSCOPE_UPLOAD_RATE` | Optional upload rate duration (e.g. `10s`) | No |
| `PYROSCOPE_DISABLE_GC_RUNS` | Optional `true/false` to disable extra GC runs | No |

### Example `.env` File

```env
PYROSCOPE_APPLICATION_NAME=my-service.production
PYROSCOPE_SERVER_ADDRESS=http://pyroscope:4040
PYROSCOPE_UPLOAD_RATE=10s
PYROSCOPE_DISABLE_GC_RUNS=false
```

## API Reference

### PyroscopeAgent

- **`Configure(configs ...PyroscopeAgentConfigOption) error`**: Configures agent startup options. Returns error when required fields are missing.
- **`Start() error`**: Starts profiler with configured options.
- **`Stop() error`**: Flushes profiles and stops profiler.

### Errors

- **`ErrAgentConfig`**: Returned when `Start` is called before `Configure`.
- **`ErrApplicationName`**: Returned when application name is empty.
- **`ErrServerAddress`**: Returned when server address is empty.

### Options

- `WithApplicationName(string)`
- `WithServerAddress(string)`
- `WithTags(map[string]string)`
- `WithProfileTypes([]pyroscope.ProfileType)`
- `WithUploadRate(time.Duration)`
- `WithDisableGCRuns(bool)`
- `WithLogger(pyroscope.Logger)`
