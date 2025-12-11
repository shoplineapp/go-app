//go:build grpc && sentry
// +build grpc,sentry

package interceptors

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/getsentry/sentry-go"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/shoplineapp/go-app/plugins"
	app_grpc "github.com/shoplineapp/go-app/plugins/grpc"
	sentry_plugin "github.com/shoplineapp/go-app/plugins/sentry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewSentryInterceptor)
}

type SentryInterceptor struct {
	sentry *sentry_plugin.SentryAgent
}

func (i *SentryInterceptor) Handler() grpc.UnaryServerInterceptor {
	customSentryInterceptor := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		hub := i.sentry.HubFromContext(ctx)
		ctx = sentry.SetHubOnContext(ctx, hub)
		rpcService, rpcMethod := parseFullMethod(info.FullMethod)
		hub.ConfigureScope(func(scope *sentry.Scope) {
			scope.SetTag("rpc.system", "grpc")
			scope.SetTag("rpc.type", "unary")
			scope.SetTag("rpc.service", rpcService)
			scope.SetTag("rpc.method", rpcMethod)

			if md, ok := metadata.FromIncomingContext(ctx); ok {
				scope.SetContext("rpc.grpc.request.metadata", extractMetadata(md))
			}
			p, ok := peer.FromContext(ctx)
			if ok {
				ipAddr := p.Addr.String()
				if host, _, err := net.SplitHostPort(ipAddr); err == nil {
					ipAddr = host
				}
				scope.SetUser(sentry.User{
					IPAddress: ipAddr,
				})
			}
		})

		// Execute the handler
		resp, err = handler(ctx, req)
		if err != nil {
			st, _ := status.FromError(err)
			hub.Scope().SetContext("rpc.grpc", map[string]any{
				"status_message": st.Message(),
			})
			hub.Scope().SetTag("rpc.grpc.status_code", fmt.Sprintf("%d", st.Code()))

			var ae *app_grpc.ApplicationError
			if errors.As(err, &ae) {
				hub.Scope().SetTag("error.expected", fmt.Sprintf("%v", ae.Expected()))
				hub.Scope().SetTag("error.code", ae.Code())

				// Add details if available
				if details := ae.Details(); len(details) > 0 {
					hub.Scope().SetContext("error.details", map[string]any{
						"details": fmt.Sprintf("%#v", details),
					})
				}
				// Only report unexpected errors
				if !ae.Expected() {
					i.sentry.CaptureException(ctx, err)
				}
			} else {
				// Report any error that is not caught as ApplicationError
				i.sentry.CaptureException(ctx, err)
			}
		}

		return resp, err
	}

	return grpc_middleware.ChainUnaryServer(
		customSentryInterceptor,
	)
}

// parseFullMethod extracts service and method from gRPC FullMethod
// FullMethod format: /$package.$service/$method
func parseFullMethod(fullMethod string) (service, method string) {
	fullMethod = strings.TrimPrefix(fullMethod, "/")
	if idx := strings.LastIndex(fullMethod, "/"); idx >= 0 {
		return fullMethod[:idx], fullMethod[idx+1:]
	}
	return fullMethod, ""
}

// extractMetadata extracts relevant metadata for Sentry context
func extractMetadata(md metadata.MD) map[string]any {
	result := make(map[string]any)
	relevantHeaders := []struct{ from, to string }{
		{from: "user-agent"},
		{from: "x-request-start", to: "x-request-start"},
		{from: "x-queue-start", to: "x-queue-start"},
		{from: "grpcgateway-x-request-start", to: "x-request-start"},
		{from: "grpcgateway-x-queue-start", to: "x-queue-start"},
	}
	for _, key := range relevantHeaders {
		if values := md.Get(key.from); len(values) > 0 {
			to := key.to
			if to == "" {
				to = key.from
			}
			if len(values) == 1 {
				result[to] = values[0]
			} else {
				result[to] = values
			}
		}
	}

	return result
}

// NewSentryInterceptor creates a new Sentry interceptor instance
func NewSentryInterceptor(sentryAgent *sentry_plugin.SentryAgent) *SentryInterceptor {
	return &SentryInterceptor{
		sentry: sentryAgent,
	}
}
