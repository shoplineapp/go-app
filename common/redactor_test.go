package common

import (
	"reflect"
	"testing"
)

type testStruct struct {
	Name string
}

type anotherTestStruct struct {
	Name string
}

type nestedStruct struct {
	StructP *testStruct `json:"structp,omitempty"`
	Struct  testStruct  `json:"-"`
	Name    string
	private string
}

type structWithMap struct {
	Payload map[string]string
}

type structWithSlice struct {
	Slice []any
}

type testInterface any

func makeTestInterface() testInterface {
	return &nestedStruct{}
}

func TestAddFilters(t *testing.T) {
	t.Run("adds a general field filter", func(t *testing.T) {
		r := &Redactor{}
		r.AddFilters(NewFilter(nil, "s", PartialRedact))
		if r.GeneralFieldFilters[0].Field != "s" {
			t.Fatal("no general field filter is found with field 's'")
		}
		if len(r.StructFilters) != 0 {
			t.Fatal("unexpected struct filter is added")
		}
		if len(r.StructFieldFilters) != 0 {
			t.Fatal("unexpected struct field filter is added")
		}
	})
	t.Run("adds a struct filter", func(tt *testing.T) {
		r := &Redactor{}
		r.AddFilters(NewFilter(&testStruct{}, "", PartialRedact))
		if r.StructFilters[0].Struct != reflect.ValueOf(testStruct{}).Type() {
			t.Fatal("no struct filter is found with struct 'testStruct'")
		}
		if len(r.GeneralFieldFilters) != 0 {
			t.Fatal("unexpected general field filter is added")
		}
		if len(r.StructFieldFilters) != 0 {
			t.Fatal("unexpected struct field filter is added")
		}
	})
	t.Run("adds a struct field filter", func(t *testing.T) {
		r := &Redactor{}
		r.AddFilters(NewFilter(&testStruct{}, "Name", PartialRedact))
		if r.StructFieldFilters[0].Struct != reflect.ValueOf(testStruct{}).Type() ||
			r.StructFieldFilters[0].Field != "Name" {
			t.Fatal("no struct field filter is found with struct 'testStruct' and field 'Name'")
		}
		if len(r.GeneralFieldFilters) != 0 {
			t.Fatal("unexpected general field filter is added")
		}
		if len(r.StructFilters) != 0 {
			t.Fatal("unexpected struct filter is added")
		}
	})
}

func TestFilterStruct(t *testing.T) {
	type test struct {
		filter *Filter
		data   any
		result bool
	}
	tests := []test{
		{NewFilter(&testStruct{}, "", PartialRedact), testStruct{}, true},
		{NewFilter(&testStruct{}, "", PartialRedact), anotherTestStruct{}, false},
		{NewFilter(&testStruct{}, "", PartialRedact), "a", false},
		{NewFilter(&testStruct{}, "", PartialRedact), 1, false},
		{NewFilter(&testStruct{}, "", PartialRedact), map[string]string{}, false},
		{NewFilter(&testStruct{}, "", PartialRedact), []string{"1"}, false},
		{NewFilter(&testStruct{}, "", PartialRedact), true, false},
	}
	for i, tt := range tests {
		r := &Redactor{}
		r.AddFilters(tt.filter)
		if ok, _ := r.filterStruct(reflect.ValueOf(tt.data).Type()); ok != tt.result {
			t.Fatalf("struct is not filtered at case: %d", i+1)
		}
	}
}

func TestFilterField(t *testing.T) {
	type test struct {
		filter *Filter
		data   any
		field  string
		result bool
	}
	tests := []test{
		{NewFilter(&testStruct{}, "A", PartialRedact), testStruct{}, "A", true},
		{NewFilter(&testStruct{}, "A", PartialRedact), testStruct{}, "B", false},
		{NewFilter(nil, "A", PartialRedact), map[string]string{}, "A", true},
		{NewFilter(nil, "A", PartialRedact), testStruct{}, "A", true},
		{NewFilter(nil, "A", PartialRedact), &testStruct{}, "A", true},
		{NewFilter(nil, "A", PartialRedact), anotherTestStruct{}, "A", true},
		{NewFilter(nil, "A", PartialRedact), &anotherTestStruct{}, "A", true},
		{NewFilter(nil, "A", PartialRedact), testStruct{}, "B", false},
	}
	for i, tt := range tests {
		r := &Redactor{}
		r.AddFilters(tt.filter)
		if ok, _ := r.filterField(reflect.ValueOf(tt.data).Type(), tt.field); ok != tt.result {
			t.Fatalf("field is not filtered at case: %d", i+1)
		}
	}
}

func TestPartialRedactString(t *testing.T) {
	type test struct {
		str    string
		result string
	}

	tests := []test{
		{"", ""},
		{"a", "*"},
		{"aa", "**"},
		{"aaa", "***"},
		{"aaaa", "a***"},
		{"very long string", "very*****"},
		{"這邊將進行", "這****"},
		{"🤬 🤯 😳 🥵 🥶 😱 😨 😰 😥 😓 🫣 🤗 🫡 🤔", "🤬 🤯 😳 🥵 *****"},
	}

	for _, tt := range tests {
		if r := partialRedactString(tt.str); r != tt.result {
			t.Fatalf("%s is not redacted as %s but %s", tt.str, tt.result, r)
		}
	}
}

func TestPartialRedact(t *testing.T) {
	type test struct {
		data   any
		result any
	}

	t.Run("primitive type", func(t *testing.T) {
		tests := []test{
			{1, "*"},
			{12, "**"},
			{123, "***"},
			{1234, "1***"},
			{"", ""},
			{"a", "*"},
			{"aa", "**"},
			{"aaa", "***"},
			{"aaaa", "a***"},
			{"very long string", "very*****"},
			{"這邊將進行", "這****"},
			{"🤬 🤯 😳 🥵 🥶 😱 😨 😰 😥 😓 🫣 🤗 🫡 🤔", "🤬 🤯 😳 🥵 *****"},
		}
		for _, tt := range tests {
			if r := PartialRedact(reflect.ValueOf(tt.data)); !reflect.DeepEqual(r, tt.result) {
				t.Fatalf("%v is not redacted as %v but %v", tt.data, tt.result, r)
			}
		}
	})
	t.Run("struct", func(t *testing.T) {
		tests := []test{
			{nil, nil},
			{(testInterface)(nil), nil},
			{makeTestInterface(), map[string]any{
				"Name":    "",
				"structp": (*testStruct)(nil),
				"Struct": map[string]any{
					"Name": "",
				},
			}},
			{
				nestedStruct{},
				map[string]any{
					"Name":    "",
					"structp": (*testStruct)(nil),
					"Struct": map[string]any{
						"Name": "",
					},
				},
			},
			{
				&nestedStruct{},
				map[string]any{
					"Name":    "",
					"structp": (*testStruct)(nil),
					"Struct": map[string]any{
						"Name": "",
					},
				},
			},
		}

		for _, tt := range tests {
			if r := PartialRedact(reflect.ValueOf(tt.data)); !reflect.DeepEqual(r, tt.result) {
				t.Fatalf("%v is not redacted as %v but %v", tt.data, tt.result, r)
			}
		}
	})
	t.Run("map", func(tt *testing.T) {
		tests := []test{
			{map[string]string{}, map[string]any{}},
			{map[int]int{1: 2}, map[string]any{"1": "*"}},
			{
				map[string]string{
					"1": "abcd",
				},
				map[string]any{
					"1": "a***",
				},
			},
			{
				map[string]map[string]string{
					"a": {
						"a": "abcd",
					},
				},
				map[string]any{
					"a": map[string]any{
						"a": "a***",
					},
				},
			},
			{
				map[string]any{
					"nested": map[string]any{
						"1": "abcd",
					},
					"1": "abcd",
				},
				map[string]any{
					"nested": map[string]any{
						"1": "a***",
					},
					"1": "a***",
				},
			},
			{(*int)(nil), (*int)(nil)},
			{[]*int{nil}, []any{(*int)(nil)}},
			{[]string{"1", "2", "3", "4"}, []any{"*", "*", "*", "*"}},
		}

		for _, tt := range tests {
			if r := PartialRedact(reflect.ValueOf(tt.data)); !reflect.DeepEqual(r, tt.result) {
				t.Fatalf("%v is not redacted as %v but %v", tt.data, tt.result, r)
			}
		}
	})
	t.Run("slice", func(t *testing.T) {
		tests := []test{
			{(*int)(nil), (*int)(nil)},
			{[]*int{nil}, []any{(*int)(nil)}},
			{[]string{"1", "2", "3", "4"}, []any{"*", "*", "*", "*"}},
		}

		for _, tt := range tests {
			if r := PartialRedact(reflect.ValueOf(tt.data)); !reflect.DeepEqual(r, tt.result) {
				t.Fatalf("%v is not redacted as %v but %v", tt.data, tt.result, r)
			}
		}
	})
}

func TestRedact(t *testing.T) {
	type test struct {
		filters []*Filter
		data    any
		result  any
	}
	t.Run("struct", func(t *testing.T) {
		tests := []test{
			{
				[]*Filter{},
				nil,
				nil,
			},
			{
				[]*Filter{},
				testStruct{},
				testStruct{},
			},
			{
				[]*Filter{
					NewFilter(testStruct{}, "", FullRedact),
				},
				testStruct{},
				"<REDACTED>",
			},
			{
				[]*Filter{
					NewFilter(testStruct{}, "", FullRedact),
				},
				&testStruct{},
				"<REDACTED>",
			},
			{
				[]*Filter{
					NewFilter(testStruct{}, "", PartialRedact),
				},
				testStruct{},
				map[string]any{"Name": ""},
			},
			{
				[]*Filter{
					NewFilter(testStruct{}, "", PartialRedact),
				},
				&testStruct{},
				map[string]any{"Name": ""},
			},
			{
				[]*Filter{
					NewFilter(structWithSlice{}, "slice", PartialRedact),
				},
				structWithSlice{Slice: []any{"abcd", "bcde"}},
				map[string]any{"Slice": []any{"a***", "b***"}},
			},
			{
				[]*Filter{
					NewFilter(structWithSlice{}, "slice", FullRedact),
				},
				structWithSlice{Slice: []any{"abcd", "bcde"}},
				map[string]any{"Slice": "<REDACTED>"},
			},
			{
				[]*Filter{
					NewFilter(structWithSlice{}, "slice", PartialRedact),
				},
				structWithSlice{Slice: []any{[]string{"abcd"}, map[string]string{"password": "abcd"}, "bcde"}},
				map[string]any{"Slice": []any{[]any{"a***"},
					map[string]any{"password": "a***"}, "b***"}},
			},
			{
				[]*Filter{
					NewFilter(structWithMap{}, "payload", PartialRedact),
				},
				structWithMap{Payload: map[string]string{"password": "hello"}},
				map[string]any{"Payload": map[string]any{"password": "h****"}},
			},
			{
				[]*Filter{
					NewFilter(structWithMap{}, "payload", FullRedact),
				},
				structWithMap{Payload: map[string]string{"password": "hello"}},
				map[string]any{"Payload": "<REDACTED>"},
			},
			{
				[]*Filter{
					NewFilter(anotherTestStruct{}, "", FullRedact),
				},
				testStruct{},
				testStruct{},
			},
		}
		for _, tt := range tests {
			r := &Redactor{}
			r.AddFilters(tt.filters...)
			if r := r.Redact(tt.data); !reflect.DeepEqual(r, tt.result) {
				t.Fatalf("%v is not redacted as %v but %v", tt.data, tt.result, r)
			}
		}
	})
	t.Run("map", func(t *testing.T) {
		tests := []test{
			{
				[]*Filter{},
				map[string]string{
					"password": "testpw",
				},
				map[string]string{
					"password": "testpw",
				},
			},
			{
				[]*Filter{
					NewFilter(nil, "password", FullRedact),
				},
				map[string]string{
					"password": "testpw",
				},
				map[string]any{
					"password": "<REDACTED>",
				},
			},
		}
		for _, tt := range tests {
			r := &Redactor{}
			r.AddFilters(tt.filters...)
			if r := r.Redact(tt.data); !reflect.DeepEqual(r, tt.result) {
				t.Fatalf("%v is not redacted as %v but %v", tt.data, tt.result, r)
			}
		}
	})
	t.Run("slice", func(t *testing.T) {
		tests := []test{
			{
				[]*Filter{
					NewFilter(nil, "password", FullRedact),
				},
				[]int{1, 2, 3},
				[]int{1, 2, 3},
			},
		}
		for _, tt := range tests {
			r := &Redactor{}
			r.AddFilters(tt.filters...)
			if r := r.Redact(tt.data); !reflect.DeepEqual(r, tt.result) {
				t.Fatalf("%v is not redacted as %v but %v", tt.data, tt.result, r)
			}
		}
	})
	t.Run("primitive", func(t *testing.T) {
		tests := []test{
			{
				[]*Filter{
					NewFilter(nil, "password", FullRedact),
				},
				1,
				1,
			},
			{
				[]*Filter{
					NewFilter(nil, "password", FullRedact),
				},
				"abcd",
				"abcd",
			},
			{
				[]*Filter{
					NewFilter(nil, "password", FullRedact),
				},
				true,
				true,
			},
		}
		for _, tt := range tests {
			r := &Redactor{}
			r.AddFilters(tt.filters...)
			if r := r.Redact(tt.data); !reflect.DeepEqual(r, tt.result) {
				t.Fatalf("%v is not redacted as %v but %v", tt.data, tt.result, r)
			}
		}
	})
}

func TestDefaultRedactor(t *testing.T) {
	data := map[string]string{
		"password": "password",
		"address":  "address",
		"cipher":   "cipher",
		"email":    "email@email.com",
	}
	redacted := DefaultRedactor.Redact(data)
	if !reflect.DeepEqual(redacted, map[string]any{
		"password": "<REDACTED>",
		"address":  "<REDACTED>",
		"cipher":   "<REDACTED>",
		"email":    "emai*****",
	}) {
		t.Fatal("default redactor is not working as expected", redacted)
	}
}
