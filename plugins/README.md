# Plugins

## Overview

The plugin architecture heavy rely on the dependency injection mechanism to construct the dependency tree and inject variable instances.

## Getting Started

- [Plugins](#plugins)
  - [Overview](#overview)
  - [Getting Started](#getting-started)
  - [Contents](#contents)
  - [Usage](#usage)
  - [Build Tags](#build-tags)

## Contents

Contents of this repository:

| Plugin | Description |
| --------- | ----------------------------------------------------------------|
| Env    | Load environment variables from `.env` file with default values. |
| gRPC    | gRPC server with gracefully shutdown and common interceptors |
| Logger     | Provide a formatted Logrus logger with your presets. |
| Newrelic     | The base framework of Newrelic agent and gRPC stats handler for transaction tracing |
| Add your plugin here | ... |

## Usage

Plugins are expected to be autoloaded, by using `init` function to inject the plugin into  the registry.

Here is the base of a plugin

```golang
package my_plugin

func init() {
	plugins.Registry = append(plugins.Registry, MyPluginConstructor)
}

type MyPlugin struct {}

func MyPluginConstructor() *MyPlugin {
  return &MyPlugin{}
}
```

Underlying the Go App instance will automatically load your plugin as providers. 

You might also inject the dependencies manually by using `app.SetPlugins`

```golang
app := &Application{}
app.SetPlugins(NewPluginOne, NewPluginTwo, NewPluginThree)
```

## Build Tags

For plugins that might not be always used, add build tags to only load it explicitly.

```golang
//go:build sentry
// +build sentry

package sentry

func init() {
  plugins.Registry = append(plugins.Registry, NewSentryPlugin)
}

type Sentry struct{}

func NewSentryPlugin() *Sentry {
  return &Sentry{}
}
```

Build or run application with build tags

```sh
go run -tags sentry cmd/api.go
go build -tags grpc,sentry -o build/api cmd/api.go
```
