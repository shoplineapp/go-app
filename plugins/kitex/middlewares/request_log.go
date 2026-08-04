package middlewares

import (
	"context"
	"time"

	"github.com/cloudwego/kitex/pkg/endpoint"
	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/shoplineapp/go-app/plugins"
	"github.com/shoplineapp/go-app/plugins/logger"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewKitexRequestLogMiddleware)
}

type KitexRequestLogMiddleware struct {
	logger *logger.Logger
}

func (m KitexRequestLogMiddleware) Handler(next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, request, response interface{}) error {
		ri := rpcinfo.GetRPCInfo(ctx)
		var traceID string
		if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
			traceID = sc.TraceID().String()
		}
		logger := logrus.WithFields(logrus.Fields{
			"Method":   ri.To().Method(),
			"trace_id": traceID,
		})
		ctx = context.WithValue(ctx, "logger", logger)

		logger.Info("Incoming Request")

		start := time.Now()
		err := next(ctx, request, response)
		stop := time.Now()
		resLogger := logger.WithFields(
			logrus.Fields{"res_time": stop.Sub(start).String(), "req": request},
		)

		if err != nil {
			resLogger.WithFields(logrus.Fields{"err": err}).Errorf("Request Executed with Error: %+v", err)
		} else {
			resLogger.WithFields(logrus.Fields{"res": response}).Info("Request Executed")
		}
		return err
	}
}

func NewKitexRequestLogMiddleware(logger *logger.Logger) *KitexRequestLogMiddleware {
	return &KitexRequestLogMiddleware{logger: logger}
}
