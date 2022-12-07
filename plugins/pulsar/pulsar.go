//go:build pulsar
// +build pulsar

package pulsar

import (
	"context"
	"time"

	ap "github.com/apache/pulsar-client-go/pulsar"
	ap_log "github.com/apache/pulsar-client-go/pulsar/log"
	"github.com/shoplineapp/go-app/plugins"
	"github.com/shoplineapp/go-app/plugins/logger"
	"go.uber.org/fx"
)

var (
	DEFAULT_PULSAR_OPERATION_TIMEOUT  = 30 * time.Second
	DEFAULT_PULSAR_CONNECTION_TIMEOUT = 10 * time.Second
)

func init() {
	plugins.Registry = append(plugins.Registry,
		[]interface{}{
			NewPulsarServer,
			NewPulsarProducerManager,
		}...,
	)
}

type PulsarServer struct {
	ap.Client

	logger *logger.Logger
}

type PulsarClientOption func(*ap.ClientOptions)

func (p *PulsarServer) Connect(URL string, opts ...PulsarClientOption) error {
	clientOpts := ap.ClientOptions{
		URL:               URL,
		OperationTimeout:  DEFAULT_PULSAR_OPERATION_TIMEOUT,
		ConnectionTimeout: DEFAULT_PULSAR_CONNECTION_TIMEOUT,
		Logger:            ap_log.NewLoggerWithLogrus(&p.logger.Logger),
	}
	for _, opt := range opts {
		opt(&clientOpts)
	}
	client, err := ap.NewClient(clientOpts)
	if err != nil {
		p.logger.Error("Unable to initialize Pulsar client", err)
		return err
	}
	p.Client = client
	p.logger.Info("Pulsar client configured")

	return nil
}

func (p *PulsarServer) Shutdown() {
	p.Close()
}

func NewPulsarServer(
	lc fx.Lifecycle,
	logger *logger.Logger,
) *PulsarServer {
	p := &PulsarServer{
		logger: logger,
	}
	if lc != nil {
		lc.Append(fx.Hook{
			OnStop: func(ctx context.Context) error {
				p.Shutdown()
				return nil
			},
		})
	}
	return p
}
