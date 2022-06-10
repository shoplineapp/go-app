//go:build newrelic
// +build newrelic

package newrelic

import (
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/shoplineapp/go-app/plugins"
	"github.com/sirupsen/logrus"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewNewrelicAgent)
}

var app *newrelic.Application

type NewrelicAgent struct{}

func (a NewrelicAgent) App() *newrelic.Application {
	return app
}

func Configure(configs ...newrelic.ConfigOption) {
	a, err := newrelic.NewApplication(configs...)
	if err != nil {
		logrus.Error("Unable to load Newrelic application")
	}

	app = a
}

func NewNewrelicAgent() *NewrelicAgent {
	return &NewrelicAgent{}
}
