package jsonpb

import (
	"testing"
)

func TestJsonBuilder(t *testing.T) {
	var b JsonBuilder
	b.AppendByte('{')
	b.AppendByte('"')
	b.AppendEscapedString("b\tc")
	b.AppendString(`":`)
	b.AppendString("123")
	b.AppendString("}")
	if b.String() != `{"b\tc":123}` {
		t.Fatal("b.String():", b.String())
	}
	b1 := UnsafeJsonBuilder([]byte(`{"a":`))
	b1.Reserve(b.Len() + 1)
	b1.AppendBytes(b.IntoBytes()...)
	b1.AppendByte('}')
	if b1.String() != `{"a":{"b\tc":123}}` {
		t.Fatal("b1.String():", b1.String())
	}
}
