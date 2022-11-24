//go:build grpc
// +build grpc

package interceptors

import (
	"context"

	"github.com/shoplineapp/go-app/plugins"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewValidateInterceptor)
}

type ProtobufValidator interface {
	Validate() error
	ValidateAll() error
}

type ValidateInterceptor struct{}

func (i ValidateInterceptor) Handler() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		// Check if request object implements protobuf validate, if so, validate it in interceptor layer
		if v, ok := req.(ProtobufValidator); ok {
			err := v.ValidateAll()

			if err != nil {
				return nil, status.Error(codes.InvalidArgument, "Invalid request params - "+err.Error())
			}
		}
		resp, err = handler(ctx, req)
		return resp, err
	}
}

func NewValidateInterceptor() *ValidateInterceptor {
	return &ValidateInterceptor{}
}
