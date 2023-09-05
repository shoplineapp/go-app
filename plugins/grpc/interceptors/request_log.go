//go:build grpc
// +build grpc

package interceptors

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/shoplineapp/go-app/common"
	"github.com/shoplineapp/go-app/plugins"
	"github.com/shoplineapp/go-app/plugins/env"
	"github.com/shoplineapp/go-app/plugins/logger"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var (
	redactor *common.Redactor
)

func init() {
	redactor = common.DefaultRedactor
	plugins.Registry = append(plugins.Registry, NewGrpcRequestLogInterceptor)
}

func SetRedactor(r *common.Redactor) {
	redactor = r
}

type RequestLogInterceptor struct {
	logger *logger.Logger
	env    *env.Env
}

type contextKey string

// var (
// 	ContextKeyTraceId        = contextKey("trace_id")
// 	ContextKeyLogger         = contextKey("logger")
// 	ContextKeyControllerData = contextKey("controller_data")
// )

func (c contextKey) String() string {
	return string(c)
}

func StructToMap(s interface{}) (res map[string]interface{}, err error) {
	data, _ := json.Marshal(s)
	err = json.Unmarshal(data, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func ItemInSlice(a interface{}, list []interface{}) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func MapToSlice(mapData map[string]interface{}) []interface{} {
	res := make([]interface{}, 0, len(mapData))
	for _, value := range mapData {
		res = append(res, value)
	}
	return res
}

func maskValue(value string) string {
	digit := int(len(value) / 2)
	head := value[:len(value)-digit]
	tail := value[len(value)-digit:]

	mask := strings.Repeat("*", len(head))
	return fmt.Sprintf("%v%v", mask, tail)
}

func markReqParams(whiteListKeys []interface{}, key string, value interface{}, res map[string]interface{}) {
	assignKey := key[strings.LastIndex(key, ".")+1:]
	switch value := value.(type) {
	case []interface{}:
		mapData := map[string]interface{}{}
		for _, elem := range value {
			markReqParams(whiteListKeys, key, elem, mapData)
		}
		res[assignKey] = MapToSlice(mapData)
	case map[string]interface{}:
		mapData := map[string]interface{}{}
		mValue, _ := StructToMap(value)
		for mKey, mValue := range mValue {
			markReqParams(whiteListKeys, fmt.Sprintf("%v.%v", key, mKey), mValue, mapData)
		}
		res[assignKey] = mapData
	default:
		if ItemInSlice(key, whiteListKeys) {
			res[assignKey] = value
		} else {
			res[assignKey] = maskValue(fmt.Sprintf("%v", value))
		}
	}
}

func SetWhitelistReqKeysInContext(ctx context.Context, keys []interface{}) {
	controllerData := ctx.Value("controller_data").(map[string]interface{})
	controllerData["whitelist_req_keys"] = keys
}

func (i RequestLogInterceptor) Handler() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ interface{}, err error) {
		// initial a ContextKeyControllerData
		ctx = context.WithValue(ctx, "controller_data", map[string]interface{}{
			"whitelist_req_keys": []interface{}{},
		})

		service := path.Dir(info.FullMethod)[1:]

		log := i.logger.WithFields(logrus.Fields{
			"trace_id": ctx.Value("trace_id"),
			"service":  service,
			"method":   path.Base(info.FullMethod),
		})

		// ignore health check request
		if service == "grpc.health.v1.Health" {
			return handler(ctx, req)
		}

		ctx = context.WithValue(ctx, "logger", log)

		log.Info("Incoming Request")

		start := time.Now()
		res, err := handler(ctx, req)
		stop := time.Now()

		resLogger := log.WithFields(
			logrus.Fields{"res_time": stop.Sub(start).String(), "req": redactor.Redact(req)},
		)

		if err != nil || res == nil {
			resLogger.WithFields(logrus.Fields{"err": err}).Errorf("Request Executed with Error: %+v", err)
		} else {
			resLogger.WithFields(logrus.Fields{"res": res}).Info("Request Executed")
		}

		return res, err
	}
}

func NewGrpcRequestLogInterceptor(logger *logger.Logger, env *env.Env) *RequestLogInterceptor {
	return &RequestLogInterceptor{
		logger: logger,
		env:    env,
	}
}
