package proto

import (
	"testing"

	"google.golang.org/protobuf/encoding/protowire"
)

func assert2[A, B comparable](t *testing.T, f func() (A, B), a2 A, b2 B) {
	a1, b1 := f()
	if a1 != a2 {
		t.Fatal("assert", a1, a2)
	}
	if b1 != b2 {
		t.Fatal("assert", b1, b2)
	}
}

func TestDecode(t *testing.T) {
	raw := []byte{8, 233, 1, 18, 4, 116, 101, 115, 116, 29, 219, 3, 0, 0, 32, 209, 3}
	dec := NewDecoder(raw)
	readTag := func() (uint32, protowire.Type) {
		a, b, _ := dec.ReadTag()
		return a, b
	}
	assert2(t, readTag, 1, protowire.VarintType)
	assert2(t, dec.ReadVarint, 233, 0)
	assert2(t, readTag, 2, protowire.BytesType)
	data, n := dec.ReadBytes()
	if n < 0 || string(data) != "test" {
		t.Fatal("ReadBytes", string(data))
	}
	assert2(t, readTag, 3, protowire.Fixed32Type)
	assert2(t, dec.ReadFixed32, 987, 0)
	assert2(t, readTag, 4, protowire.VarintType)
	assert2(t, dec.ReadZigzag, -233, 0)
}
