package common

import (
	"fmt"
	"math"
	"reflect"
	"strings"

	"golang.org/x/exp/utf8string"
)

const maxPartialRedactLen = 5

var (
	DefaultRedactor *Redactor
	DefaultFilters  = []*Filter{
		NewFilter(nil, "password", FullRedact),
		NewFilter(nil, "address", FullRedact),
		NewFilter(nil, "cipher", FullRedact),
		NewFilter(nil, "email", PartialRedact),
	}
)

func init() {
	DefaultRedactor = &Redactor{}
	DefaultRedactor.AddFilters(DefaultFilters...)
}

type Redactor struct {
	StructFilters       []*Filter
	StructFieldFilters  []*Filter
	GeneralFieldFilters []*Filter
}

type Mode func(reflect.Value) any

type Filter struct {
	Struct     reflect.Type
	Field      string
	RedactFunc Mode
}

func mapKey(field reflect.Value) string {
	switch field.Kind() {
	case reflect.String:
		return field.String()
	}

	return fmt.Sprint(field.Interface())
}

func fieldKey(field reflect.StructField) string {
	if jsonField := field.Tag.Get("json"); jsonField != "" {
		jsonTags := strings.Split(jsonField, ",") // e,g json:"-", json:"name,omitempty"`
		if jsonTags[0] != "" && jsonTags[0] != "-" {
			return jsonTags[0]
		}
	}

	return field.Name
}

func FullRedact(T reflect.Value) any {
	return "<REDACTED>"
}

func PartialRedact(rT reflect.Value) any {
	if !rT.IsValid() {
		return nil
	}

	if rT.Kind() == reflect.Pointer && rT.IsNil() {
		return rT.Interface()
	}

	rT = reflect.Indirect(rT)
	switch rT.Kind() {
	case reflect.Struct:
		nField := rT.Type().NumField()
		redacted := make(map[string]any, nField)
		for i := 0; i < nField; i++ {
			field := rT.Field(i)
			fieldType := rT.Type().Field(i)
			if !field.CanInterface() {
				continue
			}

			redacted[fieldKey(fieldType)] = PartialRedact(field)
		}
		return redacted
	case reflect.Map:
		keys := rT.MapKeys()
		length := len(keys)
		redacted := make(map[string]any, length)
		for i := 0; i < length; i++ {
			redacted[mapKey(keys[i])] = PartialRedact(rT.MapIndex(keys[i]))
		}
		return redacted
	case reflect.Interface:
		return PartialRedact(rT.Elem())
	case reflect.Slice, reflect.Array:
		length := rT.Len()
		cap := rT.Cap()
		redacted := make([]any, length, cap)
		for i := 0; i < length; i++ {
			redacted[i] = PartialRedact(rT.Index(i))
		}
		return redacted
	}

	str := fmt.Sprint(rT.Interface())
	return partialRedactString(str)
}

// redact 30% of the string
// e.g abcd => a***
func partialRedactString(str string) string {
	utf8str := utf8string.NewString(str)
	initialMask := int(math.Floor(float64(utf8str.RuneCount()) * 3 / 10))
	redactedLen := int(math.Min(maxPartialRedactLen, float64(utf8str.RuneCount()-initialMask)))
	return utf8str.Slice(0, initialMask) + strings.Repeat("*", redactedLen)
}

func NewFilter(data any, fieldName string, mode Mode) *Filter {
	f := &Filter{}
	if data != nil {
		f.Struct = reflect.Indirect(reflect.ValueOf(data)).Type()
	}
	f.Field = fieldName
	f.RedactFunc = mode
	return f
}

func (r *Redactor) AddFilters(filters ...*Filter) {
	for _, f := range filters {
		if f.Struct == nil {
			r.GeneralFieldFilters = append(r.GeneralFieldFilters, f)
			continue
		}
		if f.Field == "" {
			r.StructFilters = append(r.StructFilters, f)
			continue
		}
		r.StructFieldFilters = append(r.StructFieldFilters, f)
	}
}

func (r *Redactor) filterStruct(dataT reflect.Type) (bool, *Filter) {
	for _, f := range r.StructFilters {
		if dataT == f.Struct {
			return true, f
		}
	}
	return false, nil
}

func (r *Redactor) filterField(dataT reflect.Type, field string) (bool, *Filter) {
	if dataT != nil {
		for _, f := range r.StructFieldFilters {
			if dataT == f.Struct && strings.EqualFold(field, f.Field) {
				return true, f
			}
		}
	}

	for _, f := range r.GeneralFieldFilters {
		if strings.EqualFold(field, f.Field) {
			return true, f
		}
	}

	return false, nil
}

func (r *Redactor) Redact(T any) any {
	if T == nil {
		return nil
	}

	redacted, _ := r.redact(nil, "", reflect.ValueOf(T))
	return redacted
}

func (r *Redactor) redact(structType reflect.Type, field string, rT reflect.Value) (any, bool) {
	if !rT.IsValid() {
		return nil, false
	}

	if rT.Kind() == reflect.Pointer && rT.IsNil() {
		return rT.Interface(), false
	}

	rT = reflect.Indirect(rT)
	switch rT.Kind() {
	case reflect.Struct:
		if ok, filter := r.filterStruct(rT.Type()); ok {
			return filter.RedactFunc(rT), true
		}

		nField := rT.Type().NumField()
		redacted := make(map[string]any, nField)
		var someReacted bool
		for i := 0; i < nField; i++ {
			var fieldReacted bool
			field := rT.Field(i)
			fieldType := rT.Type().Field(i)
			if !field.CanInterface() {
				continue
			}

			redacted[fieldKey(fieldType)], fieldReacted = r.redact(rT.Type(), fieldKey(fieldType), field)
			someReacted = someReacted || fieldReacted
		}
		if !someReacted {
			return rT.Interface(), false
		}

		return redacted, true
	case reflect.Map:
		if ok, filter := r.filterField(structType, field); ok {
			return filter.RedactFunc(rT), true
		}

		keys := rT.MapKeys()
		length := len(keys)
		var someReacted bool
		redacted := make(map[string]any, length)
		for i := 0; i < length; i++ {
			var fieldReacted bool
			redacted[mapKey(keys[i])], fieldReacted = r.redact(rT.Type(), keys[i].String(), rT.MapIndex(keys[i]))
			someReacted = someReacted || fieldReacted
		}
		if !someReacted {
			return rT.Interface(), false
		}

		return redacted, true
	case reflect.Slice, reflect.Array:
		if ok, filter := r.filterField(structType, field); ok {
			return filter.RedactFunc(rT), true
		}

		length := rT.Len()
		cap := rT.Cap()
		redacted := make([]any, length, cap)
		var someReacted bool
		for i := 0; i < length; i++ {
			var fieldReacted bool
			redacted[i], fieldReacted = r.redact(structType, field, rT.Index(i))
			someReacted = someReacted || fieldReacted
		}
		if !someReacted {
			return rT.Interface(), false
		}

		return redacted, true
	case reflect.Interface:
		return r.redact(structType, field, rT.Elem())
	}

	if ok, filter := r.filterField(structType, field); ok {
		return filter.RedactFunc(rT), true
	}

	return rT.Interface(), false
}
