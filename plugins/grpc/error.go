package grpc

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ApplicationError struct {
	traceID  string
	error    error
	code     codes.Code
	expected bool
	details  []interface{}
}

func NewApplicationError(
	traceID string,
	err error,
	code codes.Code,
	expected bool,
	details ...interface{},
) *ApplicationError {
	return &ApplicationError{
		traceID:  traceID,
		error:    err,
		code:     code,
		expected: expected,
		details:  details,
	}
}

func (ae ApplicationError) TraceID() string {
	return ae.traceID
}

func (ae ApplicationError) Code() string {
	return ae.code.String()
}

func (ae *ApplicationError) Error() string {
	return ae.Code() + ": " + ae.error.Error()
}

func (ae ApplicationError) Details() []interface{} {
	return ae.details
}

// for abiding the gRPC error interface
func (ae ApplicationError) GRPCStatus() *status.Status {
	return status.New(ae.code, ae.error.Error())
}

func (ae *ApplicationError) Expected() bool { return ae.expected }

// for go 1.13+ error unwrapping
func (ae *ApplicationError) Unwrap() error { return ae.error }

// for github.com/pkg/errors error unwrapping
func (ae *ApplicationError) Cause() error { return ae.error }
