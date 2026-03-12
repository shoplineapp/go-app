//go:build pyroscope
// +build pyroscope

package pyroscope

import (
	"context"
	"errors"
	"time"

	"github.com/grafana/pyroscope-go"
	"github.com/shoplineapp/go-app/plugins"
	"go.uber.org/fx"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewPyroscopeAgent)
}

var (
	ErrAgentConfig     = errors.New("must set up pyroscope config")
	ErrApplicationName = errors.New("must set up application name")
	ErrServerAddress   = errors.New("must set up server address")
)

type PyroscopeAgent struct {
	config  *pyroscope.Config
	profile *pyroscope.Profiler
}

func (p *PyroscopeAgent) Configure(configs ...PyroscopeAgentConfigOption) error {
	agentConfig := &PyroscopeAgentConfig{}
	for _, config := range configs {
		config(agentConfig)
	}
	if err := agentConfig.Valid(); err != nil {
		return err
	}

	serviceConfig := pyroscope.Config{
		ApplicationName: agentConfig.applicationName,
		ServerAddress:   agentConfig.serverAddress,
		ProfileTypes:    agentConfig.profileTypes,
		DisableGCRuns:   agentConfig.disableGCRuns,
	}

	if len(agentConfig.tags) > 0 {
		serviceConfig.Tags = agentConfig.tags
	}

	if agentConfig.uploadRate != nil {
		serviceConfig.UploadRate = *agentConfig.uploadRate
	}

	if agentConfig.logger != nil {
		serviceConfig.Logger = agentConfig.logger
	}

	p.config = &serviceConfig
	return nil
}

func (p *PyroscopeAgent) Start() error {
	if p.config == nil {
		return ErrAgentConfig
	}

	profile, err := pyroscope.Start(*p.config)
	if err != nil {
		return err
	}

	p.profile = profile
	return nil
}

func (p *PyroscopeAgent) Stop() error {
	if p.profile == nil {
		return nil
	}

	p.profile.Flush(true)
	return p.profile.Stop()
}

type PyroscopeAgentConfig struct {
	disableGCRuns   bool
	applicationName string
	serverAddress   string
	profileTypes    []pyroscope.ProfileType
	tags            map[string]string
	uploadRate      *time.Duration
	logger          pyroscope.Logger
}

func (c *PyroscopeAgentConfig) Valid() error {
	if len(c.applicationName) == 0 {
		return ErrApplicationName
	}

	if len(c.serverAddress) == 0 {
		return ErrServerAddress
	}

	return nil
}

type PyroscopeAgentConfigOption func(q *PyroscopeAgentConfig)

func WithApplicationName(applicationName string) PyroscopeAgentConfigOption {
	return func(p *PyroscopeAgentConfig) {
		p.applicationName = applicationName
	}
}

func WithServerAddress(serverAddress string) PyroscopeAgentConfigOption {
	return func(p *PyroscopeAgentConfig) {
		p.serverAddress = serverAddress
	}
}

func WithTags(tags map[string]string) PyroscopeAgentConfigOption {
	return func(p *PyroscopeAgentConfig) {
		p.tags = tags
	}
}

func WithProfileTypes(profileTypes []pyroscope.ProfileType) PyroscopeAgentConfigOption {
	return func(p *PyroscopeAgentConfig) {
		p.profileTypes = profileTypes
	}
}

func WithUploadRate(uploadRate time.Duration) PyroscopeAgentConfigOption {
	return func(p *PyroscopeAgentConfig) {
		p.uploadRate = &uploadRate
	}
}

func WithDisableGCRuns(disableGCRuns bool) PyroscopeAgentConfigOption {
	return func(p *PyroscopeAgentConfig) {
		p.disableGCRuns = disableGCRuns
	}
}

func WithLogger(logger pyroscope.Logger) PyroscopeAgentConfigOption {
	return func(p *PyroscopeAgentConfig) {
		p.logger = logger
	}
}

func NewPyroscopeAgent(lc fx.Lifecycle) *PyroscopeAgent {
	agent := &PyroscopeAgent{}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return agent.Start()
		},
		OnStop: func(ctx context.Context) error {
			return agent.Stop()
		},
	})
	return agent
}
