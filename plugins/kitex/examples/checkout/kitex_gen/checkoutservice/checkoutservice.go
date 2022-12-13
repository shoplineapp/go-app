// Code generated by Kitex v0.4.2. DO NOT EDIT.

package checkoutservice

import (
	kitex_gen "checkout_service/kitex_gen"
	"context"
	"fmt"
	client "github.com/cloudwego/kitex/client"
	kitex "github.com/cloudwego/kitex/pkg/serviceinfo"
	streaming "github.com/cloudwego/kitex/pkg/streaming"
	proto "google.golang.org/protobuf/proto"
)

func serviceInfo() *kitex.ServiceInfo {
	return checkoutServiceServiceInfo
}

var checkoutServiceServiceInfo = NewServiceInfo()

func NewServiceInfo() *kitex.ServiceInfo {
	serviceName := "CheckoutService"
	handlerType := (*kitex_gen.CheckoutService)(nil)
	methods := map[string]kitex.MethodInfo{
		"Healthcheck": kitex.NewMethodInfo(healthcheckHandler, newHealthcheckArgs, newHealthcheckResult, false),
	}
	extra := map[string]interface{}{
		"PackageName": "",
	}
	svcInfo := &kitex.ServiceInfo{
		ServiceName:     serviceName,
		HandlerType:     handlerType,
		Methods:         methods,
		PayloadCodec:    kitex.Protobuf,
		KiteXGenVersion: "v0.4.2",
		Extra:           extra,
	}
	return svcInfo
}

func healthcheckHandler(ctx context.Context, handler interface{}, arg, result interface{}) error {
	switch s := arg.(type) {
	case *streaming.Args:
		st := s.Stream
		req := new(kitex_gen.Request)
		if err := st.RecvMsg(req); err != nil {
			return err
		}
		resp, err := handler.(kitex_gen.CheckoutService).Healthcheck(ctx, req)
		if err != nil {
			return err
		}
		if err := st.SendMsg(resp); err != nil {
			return err
		}
	case *HealthcheckArgs:
		success, err := handler.(kitex_gen.CheckoutService).Healthcheck(ctx, s.Req)
		if err != nil {
			return err
		}
		realResult := result.(*HealthcheckResult)
		realResult.Success = success
	}
	return nil
}
func newHealthcheckArgs() interface{} {
	return &HealthcheckArgs{}
}

func newHealthcheckResult() interface{} {
	return &HealthcheckResult{}
}

type HealthcheckArgs struct {
	Req *kitex_gen.Request
}

func (p *HealthcheckArgs) FastRead(buf []byte, _type int8, number int32) (n int, err error) {
	if !p.IsSetReq() {
		p.Req = new(kitex_gen.Request)
	}
	return p.Req.FastRead(buf, _type, number)
}

func (p *HealthcheckArgs) FastWrite(buf []byte) (n int) {
	if !p.IsSetReq() {
		return 0
	}
	return p.Req.FastWrite(buf)
}

func (p *HealthcheckArgs) Size() (n int) {
	if !p.IsSetReq() {
		return 0
	}
	return p.Req.Size()
}

func (p *HealthcheckArgs) Marshal(out []byte) ([]byte, error) {
	if !p.IsSetReq() {
		return out, fmt.Errorf("No req in HealthcheckArgs")
	}
	return proto.Marshal(p.Req)
}

func (p *HealthcheckArgs) Unmarshal(in []byte) error {
	msg := new(kitex_gen.Request)
	if err := proto.Unmarshal(in, msg); err != nil {
		return err
	}
	p.Req = msg
	return nil
}

var HealthcheckArgs_Req_DEFAULT *kitex_gen.Request

func (p *HealthcheckArgs) GetReq() *kitex_gen.Request {
	if !p.IsSetReq() {
		return HealthcheckArgs_Req_DEFAULT
	}
	return p.Req
}

func (p *HealthcheckArgs) IsSetReq() bool {
	return p.Req != nil
}

type HealthcheckResult struct {
	Success *kitex_gen.Response
}

var HealthcheckResult_Success_DEFAULT *kitex_gen.Response

func (p *HealthcheckResult) FastRead(buf []byte, _type int8, number int32) (n int, err error) {
	if !p.IsSetSuccess() {
		p.Success = new(kitex_gen.Response)
	}
	return p.Success.FastRead(buf, _type, number)
}

func (p *HealthcheckResult) FastWrite(buf []byte) (n int) {
	if !p.IsSetSuccess() {
		return 0
	}
	return p.Success.FastWrite(buf)
}

func (p *HealthcheckResult) Size() (n int) {
	if !p.IsSetSuccess() {
		return 0
	}
	return p.Success.Size()
}

func (p *HealthcheckResult) Marshal(out []byte) ([]byte, error) {
	if !p.IsSetSuccess() {
		return out, fmt.Errorf("No req in HealthcheckResult")
	}
	return proto.Marshal(p.Success)
}

func (p *HealthcheckResult) Unmarshal(in []byte) error {
	msg := new(kitex_gen.Response)
	if err := proto.Unmarshal(in, msg); err != nil {
		return err
	}
	p.Success = msg
	return nil
}

func (p *HealthcheckResult) GetSuccess() *kitex_gen.Response {
	if !p.IsSetSuccess() {
		return HealthcheckResult_Success_DEFAULT
	}
	return p.Success
}

func (p *HealthcheckResult) SetSuccess(x interface{}) {
	p.Success = x.(*kitex_gen.Response)
}

func (p *HealthcheckResult) IsSetSuccess() bool {
	return p.Success != nil
}

type kClient struct {
	c client.Client
}

func newServiceClient(c client.Client) *kClient {
	return &kClient{
		c: c,
	}
}

func (p *kClient) Healthcheck(ctx context.Context, Req *kitex_gen.Request) (r *kitex_gen.Response, err error) {
	var _args HealthcheckArgs
	_args.Req = Req
	var _result HealthcheckResult
	if err = p.c.Call(ctx, "Healthcheck", &_args, &_result); err != nil {
		return
	}
	return _result.GetSuccess(), nil
}
