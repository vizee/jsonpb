package jsonpb

import (
	"encoding/base64"
	"errors"
	"math"
	"strconv"

	"github.com/vizee/jsonpb/proto"
	"google.golang.org/protobuf/encoding/protowire"
)

var (
	ErrInvalidWireType = errors.New("invalid wire type")
)

type protoValue struct {
	x uint64
	s []byte
}

func readProtoValue(p *proto.Decoder, wire protowire.Type) (val protoValue, e int) {
	switch wire {
	case protowire.VarintType:
		val.x, e = p.ReadVarint()
	case protowire.Fixed32Type:
		var t uint32
		t, e = p.ReadFixed32()
		val.x = uint64(t)
	case protowire.Fixed64Type:
		val.x, e = p.ReadFixed64()
	case protowire.BytesType:
		val.s, e = p.ReadBytes()
	default:
		e = -100
	}
	return
}

var wireTypeOfKind = [...]protowire.Type{
	DoubleKind:   protowire.Fixed64Type,
	FloatKind:    protowire.Fixed32Type,
	Int32Kind:    protowire.VarintType,
	Int64Kind:    protowire.VarintType,
	Uint32Kind:   protowire.VarintType,
	Uint64Kind:   protowire.VarintType,
	Sint32Kind:   protowire.VarintType,
	Sint64Kind:   protowire.VarintType,
	Fixed32Kind:  protowire.Fixed32Type,
	Fixed64Kind:  protowire.Fixed64Type,
	Sfixed32Kind: protowire.Fixed32Type,
	Sfixed64Kind: protowire.Fixed64Type,
	BoolKind:     protowire.VarintType,
	// StringKind:   protowire.BytesType,
	// BytesKind:    protowire.BytesType,
	// MapKind:      protowire.BytesType,
	// MessageKind:  protowire.BytesType,
}

func getFieldWireType(kind Kind, repeated bool) protowire.Type {
	// 如果字段设置 repeated，那么值应该是 packed/string/bytes/message，所以 wire 一定是 BytesType
	if !repeated && int(kind) < len(wireTypeOfKind) {
		return wireTypeOfKind[kind]
	}
	return protowire.BytesType
}

var defaultValues = [...]string{
	DoubleKind:   `0`,
	FloatKind:    `0`,
	Int32Kind:    `0`,
	Int64Kind:    `0`,
	Uint32Kind:   `0`,
	Uint64Kind:   `0`,
	Sint32Kind:   `0`,
	Sint64Kind:   `0`,
	Fixed32Kind:  `0`,
	Fixed64Kind:  `0`,
	Sfixed32Kind: `0`,
	Sfixed64Kind: `0`,
	BoolKind:     `false`,
	StringKind:   `""`,
	BytesKind:    `""`,
	MapKind:      `{}`,
	MessageKind:  `{}`,
}

func writeDefaultValue(j *JsonBuilder, repeated bool, kind Kind) {
	if repeated {
		j.AppendString("[]")
	} else {
		j.AppendString(defaultValues[kind])
	}
}

func transProtoMap(j *JsonBuilder, p *proto.Decoder, tag uint32, entry *Message, s []byte) error {
	j.AppendByte('{')

	keyField, valueField := entry.FieldByTag(1), entry.FieldByTag(2)
	// assert(keyField != nil && valueField != nil)
	keyWire := getFieldWireType(keyField.Kind, keyField.Repeated)
	valueWire := getFieldWireType(valueField.Kind, valueField.Repeated)
	// 暂不检查 keyField.Kind

	more := false
	for {
		if !more {
			more = true
		} else {
			j.AppendByte(',')
		}

		// 上下文比较复杂，直接嵌套逻辑读取 KV

		var values [2]protoValue
		assigned := 0
		dec := proto.NewDecoder(s)
		for !dec.EOF() && assigned != 3 {
			tag, wire, e := dec.ReadTag()
			if e < 0 {
				return protowire.ParseError(e)
			}
			val, e := readProtoValue(dec, wire)
			if e < 0 {
				return protowire.ParseError(e)
			}
			switch tag {
			case 1:
				if wire != keyWire {
					return ErrInvalidWireType
				}
				values[0] = val
				assigned |= 1
			case 2:
				if wire != valueWire {
					return ErrInvalidWireType
				}
				values[1] = val
				assigned |= 2
			}
		}

		if assigned&1 != 0 {
			if keyField.Kind == StringKind {
				transProtoString(j, values[0].s)
			} else {
				j.AppendByte('"')
				transProtoSimpleValue(j, keyField.Kind, values[0].x)
				j.AppendByte('"')
			}
		} else {
			j.AppendString(`""`)
		}

		j.AppendByte(':')

		if assigned&2 != 0 {
			switch valueField.Kind {
			case StringKind:
				transProtoString(j, values[1].s)
			case BytesKind:
				transProtoBytes(j, values[1].s)
			case MessageKind:
				err := transProtoMessage(j, proto.NewDecoder(values[1].s), valueField.Ref)
				if err != nil {
					return err
				}
			default:
				transProtoSimpleValue(j, valueField.Kind, values[1].x)
			}
		} else {
			writeDefaultValue(j, valueField.Repeated, valueField.Kind)
		}

		if p.EOF() {
			break
		}
		nextTag, wire, e := p.PeekTag()
		if e < 0 {
			return protowire.ParseError(e)
		}
		if nextTag != tag {
			break
		}
		if wire != protowire.BytesType {
			return ErrInvalidWireType
		}
		p.ReadVarint() // consume tag
		s, e = p.ReadBytes()
		if e < 0 {
			return protowire.ParseError(e)
		}
	}

	j.AppendByte('}')
	return nil
}

func transProtoRepeatedBytes(j *JsonBuilder, p *proto.Decoder, field *Field, s []byte) error {
	j.AppendByte('[')

	more := false
	for {
		if !more {
			more = true
		} else {
			j.AppendByte(',')
		}

		switch field.Kind {
		case StringKind:
			transProtoString(j, s)
		case BytesKind:
			transProtoBytes(j, s)
		case MessageKind:
			err := transProtoMessage(j, proto.NewDecoder(s), field.Ref)
			if err != nil {
				return err
			}
		}

		if p.EOF() {
			break
		}
		tag, wire, e := p.PeekTag()
		if e < 0 {
			return protowire.ParseError(e)
		}
		if tag != field.Tag {
			break
		}
		if wire != protowire.BytesType {
			return ErrInvalidWireType
		}
		p.ReadVarint() // consume tag
		s, e = p.ReadBytes()
		if e < 0 {
			return protowire.ParseError(e)
		}
	}

	j.AppendByte(']')
	return nil
}

func transProtoPackedArray(j *JsonBuilder, s []byte, field *Field) error {
	p := proto.NewDecoder(s)

	j.AppendByte('[')

	wire := getFieldWireType(field.Kind, false)
	more := false
	for !p.EOF() {
		if !more {
			more = true
		} else {
			j.AppendByte(',')
		}
		val, e := readProtoValue(p, wire)
		if e < 0 {
			return protowire.ParseError(e)
		}
		transProtoSimpleValue(j, field.Kind, val.x)
	}

	j.AppendByte(']')
	return nil
}

func transProtoBytes(j *JsonBuilder, s []byte) {
	j.AppendByte('"')
	n := base64.StdEncoding.EncodedLen(len(s))
	j.Reserve(n)
	m := len(j.buf)
	d := j.buf[m : m+n]
	base64.StdEncoding.Encode(d, s)
	j.buf = j.buf[:m+n]
	j.AppendByte('"')
}

func transProtoString(j *JsonBuilder, s []byte) {
	j.AppendByte('"')
	j.AppendEscapedString(asString(s))
	j.AppendByte('"')
}

func transProtoSimpleValue(j *JsonBuilder, kind Kind, x uint64) {
	switch kind {
	case DoubleKind:
		j.buf = strconv.AppendFloat(j.buf, math.Float64frombits(x), 'f', -1, 64)
	case FloatKind:
		j.buf = strconv.AppendFloat(j.buf, float64(math.Float32frombits(uint32(x))), 'f', -1, 32)
	case Int32Kind, Int64Kind, Sfixed64Kind:
		j.buf = strconv.AppendInt(j.buf, int64(x), 10)
	case Uint32Kind, Uint64Kind, Fixed32Kind, Fixed64Kind:
		j.buf = strconv.AppendUint(j.buf, x, 10)
	case Sint32Kind, Sint64Kind:
		j.buf = strconv.AppendInt(j.buf, protowire.DecodeZigZag(x), 10)
	case Sfixed32Kind:
		j.buf = strconv.AppendInt(j.buf, int64(int32(x)), 10)
	case BoolKind:
		if x != 0 {
			j.AppendString("true")
		} else {
			j.AppendString("false")
		}
	}
}

func transProtoMessage(j *JsonBuilder, p *proto.Decoder, msg *Message) error {
	j.AppendByte('{')

	const preAllocSize = 16
	var (
		preAlloc [preAllocSize]bool
		emitted  []bool
	)
	if len(msg.Fields) <= preAllocSize {
		emitted = preAlloc[:]
	} else {
		emitted = make([]bool, len(msg.Fields))
	}

	more := false
	for !p.EOF() {
		tag, wire, e := p.ReadTag()
		if e < 0 {
			return protowire.ParseError(e)
		}

		val, e := readProtoValue(p, wire)
		if e < 0 {
			return protowire.ParseError(e)
		}

		fieldIdx := msg.FieldIndexByTag(tag)
		if fieldIdx < 0 {
			continue
		}
		field := &msg.Fields[fieldIdx]
		expectedWire := getFieldWireType(field.Kind, field.Repeated)
		if expectedWire != wire {
			return ErrInvalidWireType
		}

		if emitted[fieldIdx] {
			continue
		}

		if !more {
			more = true
		} else {
			j.AppendByte(',')
		}
		j.AppendByte('"')
		j.AppendString(field.Name)
		j.AppendByte('"')
		j.AppendByte(':')

		var err error
		if field.Repeated {
			switch field.Kind {
			case StringKind, BytesKind, MessageKind:
				err = transProtoRepeatedBytes(j, p, field, val.s)
			default:
				err = transProtoPackedArray(j, val.s, field)
			}
		} else if field.Kind == MapKind {
			err = transProtoMap(j, p, field.Tag, field.Ref, val.s)
		} else {
			switch field.Kind {
			case StringKind:
				transProtoString(j, val.s)
			case BytesKind:
				transProtoBytes(j, val.s)
			case MessageKind:
				err = transProtoMessage(j, proto.NewDecoder(val.s), field.Ref)
			default:
				transProtoSimpleValue(j, field.Kind, val.x)
			}
		}
		if err != nil {
			return err
		}

		emitted[fieldIdx] = true
	}

	for i := range msg.Fields {
		field := &msg.Fields[i]
		if emitted[i] || field.OmitEmpty {
			continue
		}
		if !more {
			more = true
		} else {
			j.AppendByte(',')
		}
		j.AppendByte('"')
		j.AppendString(field.Name)
		j.AppendByte('"')
		j.AppendByte(':')
		writeDefaultValue(j, field.Repeated, field.Kind)
	}

	j.AppendByte('}')
	return nil
}

func TranscodeToJson(j *JsonBuilder, p *proto.Decoder, msg *Message) error {
	return transProtoMessage(j, p, msg)
}
