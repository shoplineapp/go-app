//go:build pyroscope
// +build pyroscope

package pyroscope

import (
	"fmt"

	"github.com/grafana/pyroscope-go"
	"github.com/shoplineapp/go-app/plugins"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewPyroscopeAgent)
}

type PyroscopeAgent struct{}

type PyroscopeConfig struct {
	AppName string
}

func Configure(config PyroscopeConfig) error {
	serverAddress := "http://alloy.kube-system.svc.cluster.local:4040"
	_, err := pyroscope.Start(pyroscope.Config{
		ApplicationName: config.AppName,
		ServerAddress:   serverAddress,
		Logger:          pyroscope.StandardLogger,
	})

	if err != nil {
		return fmt.Errorf("failed to start pyroscope: %w", err)
	}

	return nil
}

func NewPyroscopeAgent() *PyroscopeAgent {
	return &PyroscopeAgent{}
}
