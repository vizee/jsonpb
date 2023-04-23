package proto

import (
	"bytes"
	"testing"
)

func TestEncode(t *testing.T) {
	e := NewEncoder(nil)
	e.EmitVarint(1, 233)
	e.EmitBytes(2, []byte(`test`))
	e.EmitFixed32(3, 987)
	e.EmitZigzag(4, -233)
	want := []byte{8, 233, 1, 18, 4, 116, 101, 115, 116, 29, 219, 3, 0, 0, 32, 209, 3}
	if !bytes.Equal(e.Bytes(), want) {
		t.Fail()
	}
}
