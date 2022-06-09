package interceptors

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/shoplineapp/go-app/plugins"
	"github.com/shoplineapp/go-app/plugins/logger"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewGrpcRequestLogInterceptor)
}

type RequestLogInterceptor struct {
	logger *logger.Logger
}

type contextKey string

var (
	ContextKeyTraceId        = contextKey("trace_id")
	ContextKeyLogger         = contextKey("logger")
	ContextKeyControllerData = contextKey("controller_data")
)

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
	controllerData := ctx.Value(ContextKeyControllerData).(map[string]interface{})
	controllerData["whitelist_req_keys"] = keys
}

func (i RequestLogInterceptor) Handler() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// initial a ContextKeyControllerData
		ctx = context.WithValue(ctx, ContextKeyControllerData, map[string]interface{}{
			"whitelist_req_keys": []interface{}{},
		})

		log := i.logger.WithFields(logrus.Fields{
			"trace_id": ctx.Value(ContextKeyTraceId),
			"service":  path.Dir(info.FullMethod)[1:],
			"method":   path.Base(info.FullMethod),
		})

		ctx = context.WithValue(ctx, ContextKeyLogger, log)

		log.Info("Incoming Request")

		start := time.Now()
		res, err := handler(ctx, req)
		stop := time.Now()

		controllerData := ctx.Value(ContextKeyControllerData).(map[string]interface{})
		whiteListKeys := controllerData["whitelist_req_keys"].([]interface{})

		// whitelist and mask the req
		newReq := map[string]interface{}{}
		mapReq, _ := StructToMap(req)
		for key, value := range mapReq {
			markReqParams(whiteListKeys, key, value, newReq)
		}

		resLogger := log.WithFields(
			logrus.Fields{"res_time": stop.Sub(start).String(), "req": newReq},
		)

		if err != nil || res == nil {
			resLogger.WithFields(logrus.Fields{"err": err}).Error("Request Executed with Error")
		} else {
			resLogger.WithFields(logrus.Fields{"res": res}).Info("Request Executed")
		}

		return res, err
	}
}

func NewGrpcRequestLogInterceptor(logger *logger.Logger) *RequestLogInterceptor {
	return &RequestLogInterceptor{
		logger: logger,
	}
}
