//go:build grpc
// +build grpc

package interceptors

import (
	"sort"

	"github.com/samber/lo"
	"github.com/shoplineapp/go-app/plugins"
	"google.golang.org/grpc"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewInteterceptorsStore)
}

type InterceptorsStore struct {
	interceptors []Interceptor
}

func (s *InterceptorsStore) All() []grpc.UnaryServerInterceptor {
	sort.Slice(s.interceptors, func(i, j int) bool {
		return s.interceptors[i].GetOrder() < s.interceptors[j].GetOrder()
	})
	ans := lo.Map(s.interceptors, func(interceptor Interceptor, _ int) grpc.UnaryServerInterceptor {
		return interceptor.Handler()
	})
	return ans
}

func NewInteterceptorsStore(p InterceptorParams) *InterceptorsStore {
	s := &InterceptorsStore{
		interceptors: p.Interceptors,
	}
	return s
}
