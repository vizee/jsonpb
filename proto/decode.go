package proto

import "google.golang.org/protobuf/encoding/protowire"

type Decoder struct {
	buf []byte
	i   int
}

func (d *Decoder) EOF() bool {
	return d.i >= len(d.buf)
}

func (d *Decoder) ReadVarint() (uint64, int) {
	v, n := protowire.ConsumeVarint(d.buf[d.i:])
	if n < 0 {
		return 0, n
	}
	d.i += n
	return v, 0
}

func (d *Decoder) ReadZigzag() (int64, int) {
	v, e := d.ReadVarint()
	return protowire.DecodeZigZag(v), e
}

func (d *Decoder) PeekTag() (uint32, protowire.Type, int) {
	v, n := protowire.ConsumeVarint(d.buf[d.i:])
	if n < 0 {
		return 0, 0, n
	}
	tag, wire := protowire.DecodeTag(v)
	return uint32(tag), wire, 0
}

func (d *Decoder) ReadTag() (uint32, protowire.Type, int) {
	v, e := d.ReadVarint()
	if e < 0 {
		return 0, 0, e
	}
	tag, wire := protowire.DecodeTag(v)
	return uint32(tag), wire, 0
}

func (d *Decoder) ReadFixed32() (uint32, int) {
	v, n := protowire.ConsumeFixed32(d.buf[d.i:])
	if n < 0 {
		return 0, n
	}
	d.i += n
	return v, 0
}

func (d *Decoder) ReadFixed64() (uint64, int) {
	v, n := protowire.ConsumeFixed64(d.buf[d.i:])
	if n < 0 {
		return 0, n
	}
	d.i += n
	return v, 0
}

func (d *Decoder) ReadBytes() ([]byte, int) {
	v, n := protowire.ConsumeBytes(d.buf[d.i:])
	if n < 0 {
		return nil, n
	}
	d.i += n
	return v, 0
}

func NewDecoder(buf []byte) *Decoder {
	return &Decoder{buf: buf}
}
