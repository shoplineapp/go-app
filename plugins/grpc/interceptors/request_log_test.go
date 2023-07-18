package interceptors

import (
	"testing"
)

type testStruct struct {
	Name     string
	Age      int
	Password string
}

func BenchmarkRedactor(b *testing.B) {
	s := &testStruct{Name: "JC", Age: 1, Password: "random"}
	for i := 0; i < b.N; i++ {
		redactor.Redact(s)
	}
}

func mask(data any) map[string]interface{} {
	newReq := map[string]interface{}{}
	mapReq, _ := StructToMap(data)
	for key, value := range mapReq {
		markReqParams(nil, key, value, newReq)
	}
	return newReq
}
func BenchmarkMasking(b *testing.B) {
	s := &testStruct{Name: "JC", Age: 1, Password: "random"}
	for i := 0; i < b.N; i++ {
		mask(s)
	}
}
