//go:build grpc
// +build grpc

package interceptors

import (
	"context"

	"github.com/shoplineapp/go-app/plugins"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewLocaleInterceptor)
}

type LocaleInterceptor struct {
}

func (i LocaleInterceptor) Handler() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		var locale string
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if v := md.Get("locale"); len(v) > 0 {
				locale = v[0]
			}
		}

		if locale == "" {
			return handler(ctx, req)
		}

		ctx = context.WithValue(ctx, "locale", locale)

		grpc.SetHeader(ctx, metadata.Pairs("locale", locale))

		resp, err = handler(ctx, req)

		return resp, err
	}
}

func NewLocaleInterceptor() *LocaleInterceptor {
	return &LocaleInterceptor{}
}
