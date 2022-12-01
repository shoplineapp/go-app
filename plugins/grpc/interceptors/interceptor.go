//go:build grpc
// +build grpc

package interceptors

import (
	"go.uber.org/fx"
	"google.golang.org/grpc"
)

type Interceptor interface {
	Handler() grpc.UnaryServerInterceptor
	GetOrder() int
}

type InterceptorResult struct {
	fx.Out

	Interceptor Interceptor `group:"interceptors"`
}

type InterceptorParams struct {
	fx.In

	Interceptors []Interceptor `group:"interceptors"`
}
