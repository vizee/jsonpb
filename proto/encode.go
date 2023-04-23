package proto

import (
	"google.golang.org/protobuf/encoding/protowire"
)

type Encoder struct {
	buf []byte
}

func (e *Encoder) Len() int {
	return len(e.buf)
}

func (e *Encoder) Clear() {
	e.buf = e.buf[:0]
}

func (e *Encoder) Bytes() []byte {
	return e.buf
}

func (e *Encoder) WriteBytes(s []byte) {
	e.buf = append(e.buf, s...)
}

func (e *Encoder) WriteVarint(v uint64) {
	e.buf = protowire.AppendVarint(e.buf, v)
}

func (e *Encoder) WriteZigzag(x int64) {
	e.buf = protowire.AppendVarint(e.buf, protowire.EncodeZigZag(x))
}

func (e *Encoder) WriteFixed32(v uint32) {
	e.buf = protowire.AppendFixed32(e.buf, v)
}

func (e *Encoder) WriteFixed64(v uint64) {
	e.buf = protowire.AppendFixed64(e.buf, v)
}

func (e *Encoder) EmitVarint(tag uint32, v uint64) {
	e.buf = protowire.AppendVarint(e.buf, protowire.EncodeTag(protowire.Number(tag), protowire.VarintType))
	e.buf = protowire.AppendVarint(e.buf, v)
}

func (e *Encoder) EmitZigzag(tag uint32, x int64) {
	e.buf = protowire.AppendVarint(e.buf, protowire.EncodeTag(protowire.Number(tag), protowire.VarintType))
	e.buf = protowire.AppendVarint(e.buf, protowire.EncodeZigZag(x))
}

func (e *Encoder) EmitFixed32(tag uint32, v uint32) {
	e.buf = protowire.AppendVarint(e.buf, protowire.EncodeTag(protowire.Number(tag), protowire.Fixed32Type))
	e.buf = protowire.AppendFixed32(e.buf, v)
}

func (e *Encoder) EmitFixed64(tag uint32, v uint64) {
	e.buf = protowire.AppendVarint(e.buf, protowire.EncodeTag(protowire.Number(tag), protowire.Fixed64Type))
	e.buf = protowire.AppendFixed64(e.buf, v)
}

func (e *Encoder) EmitBytes(tag uint32, s []byte) {
	e.buf = protowire.AppendVarint(e.buf, protowire.EncodeTag(protowire.Number(tag), protowire.BytesType))
	e.buf = protowire.AppendVarint(e.buf, uint64(len(s)))
	e.buf = append(e.buf, s...)
}

func (e *Encoder) EmitString(tag uint32, s string) {
	e.buf = protowire.AppendVarint(e.buf, protowire.EncodeTag(protowire.Number(tag), protowire.BytesType))
	e.buf = protowire.AppendVarint(e.buf, uint64(len(s)))
	e.buf = append(e.buf, s...)
}

func NewEncoder(buf []byte) *Encoder {
	return &Encoder{
		buf: buf,
	}
}
