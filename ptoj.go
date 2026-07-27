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

// fieldScan 记录某个字段在 wire 流中的一次出现。
type fieldScan struct {
	wire protowire.Type
	val  protoValue
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
	// 如果字段设置 repeated，那么 packed/string/bytes/message 的 wire 一定是 BytesType
	// （但 repeated 标量也允许 unpacked 编码，见 acceptFieldWire）。
	if !repeated && int(kind) < len(wireTypeOfKind) {
		return wireTypeOfKind[kind]
	}
	return protowire.BytesType
}

// acceptFieldWire 判断 wire 类型是否可被该字段接受。
// 重复标量字段允许 packed (BytesType) 或 unpacked (标量自身 wire 类型) 两种编码，
// 这是 proto3 规范要求的：解析器必须同时接受两种编码。
func acceptFieldWire(field *Field, wire protowire.Type) bool {
	if field.Repeated {
		if wire == protowire.BytesType {
			return true
		}
		if int(field.Kind) < len(wireTypeOfKind) && wireTypeOfKind[field.Kind] == wire {
			return true
		}
		return false
	}
	return getFieldWireType(field.Kind, false) == wire
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

// transProtoMapEntry 把一条 map entry 的字节解码并追加 "key":value 到 j
// （不含外层大括号与元素间逗号）。
func transProtoMapEntry(j *JsonBuilder, entry *Message, s []byte) error {
	keyField, valueField := entry.FieldByTag(1), entry.FieldByTag(2)
	// assert(keyField != nil && valueField != nil)
	keyWire := getFieldWireType(keyField.Kind, keyField.Repeated)
	valueWire := getFieldWireType(valueField.Kind, valueField.Repeated)
	// 暂不检查 keyField.Kind

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

	// key 缺省时按字段类型输出默认值（数值 key 为 "0"，字符串 key 为 ""）
	if assigned&1 != 0 {
		if keyField.Kind == StringKind {
			transProtoString(j, values[0].s)
		} else {
			j.AppendByte('"')
			transProtoSimpleValue(j, keyField.Kind, values[0].x)
			j.AppendByte('"')
		}
	} else {
		if keyField.Kind == StringKind {
			j.AppendString(`""`)
		} else {
			j.AppendByte('"')
			transProtoSimpleValue(j, keyField.Kind, 0)
			j.AppendByte('"')
		}
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

// appendFloat 把浮点数按 protobuf JSON 规范追加到 j：
// NaN/±Inf 输出为 "NaN"/"Infinity"/"-Infinity"，其余用最短 'f' 表示。
func appendFloat(j *JsonBuilder, f float64, bits int) {
	switch {
	case math.IsNaN(f):
		j.AppendString("NaN")
	case math.IsInf(f, 1):
		j.AppendString("Infinity")
	case math.IsInf(f, -1):
		j.AppendString("-Infinity")
	default:
		j.buf = strconv.AppendFloat(j.buf, f, 'f', -1, bits)
	}
}

func transProtoSimpleValue(j *JsonBuilder, kind Kind, x uint64) {
	switch kind {
	case DoubleKind:
		appendFloat(j, math.Float64frombits(x), 64)
	case FloatKind:
		appendFloat(j, float64(math.Float32frombits(uint32(x))), 32)
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

// transProtoSingular 输出一个非重复字段的单值。
func transProtoSingular(j *JsonBuilder, field *Field, o fieldScan) error {
	switch field.Kind {
	case StringKind:
		transProtoString(j, o.val.s)
	case BytesKind:
		transProtoBytes(j, o.val.s)
	case MessageKind:
		return transProtoMessage(j, proto.NewDecoder(o.val.s), field.Ref)
	default:
		transProtoSimpleValue(j, field.Kind, o.val.x)
	}
	return nil
}

// transProtoRepeated 输出一个重复字段的所有出现（跨非连续位置已拼接），含外层方括号。
func transProtoRepeated(j *JsonBuilder, field *Field, occ []fieldScan) error {
	j.AppendByte('[')
	first := true
	sep := func() {
		if first {
			first = false
		} else {
			j.AppendByte(',')
		}
	}
	for _, o := range occ {
		switch field.Kind {
		case StringKind:
			sep()
			transProtoString(j, o.val.s)
		case BytesKind:
			sep()
			transProtoBytes(j, o.val.s)
		case MessageKind:
			sep()
			if err := transProtoMessage(j, proto.NewDecoder(o.val.s), field.Ref); err != nil {
				return err
			}
		default:
			// 数值/bool：可能是 packed (BytesType) 或 unpacked (单元素)
			if int(field.Kind) >= len(wireTypeOfKind) {
				return ErrTypeMismatch
			}
			if o.wire == protowire.BytesType {
				dec := proto.NewDecoder(o.val.s)
				elemWire := wireTypeOfKind[field.Kind]
				for !dec.EOF() {
					v, e := readProtoValue(dec, elemWire)
					if e < 0 {
						return protowire.ParseError(e)
					}
					sep()
					transProtoSimpleValue(j, field.Kind, v.x)
				}
			} else {
				sep()
				transProtoSimpleValue(j, field.Kind, o.val.x)
			}
		}
	}
	j.AppendByte(']')
	return nil
}

func transProtoMessage(j *JsonBuilder, p *proto.Decoder, msg *Message) error {
	// 两遍处理：先收集每个字段的所有出现，再按字段定义顺序输出。
	// 这样才能正确拼接非连续出现的重复字段，并对非重复字段实现 last-one-wins。
	const preAllocSize = 16
	var preAlloc [preAllocSize][]fieldScan
	var occurrences [][]fieldScan
	if len(msg.Fields) <= preAllocSize {
		occurrences = preAlloc[:]
	} else {
		occurrences = make([][]fieldScan, len(msg.Fields))
	}

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
		if !acceptFieldWire(field, wire) {
			return ErrInvalidWireType
		}
		occurrences[fieldIdx] = append(occurrences[fieldIdx], fieldScan{wire: wire, val: val})
	}

	j.AppendByte('{')
	more := false
	emitHeader := func(name string) {
		if more {
			j.AppendByte(',')
		} else {
			more = true
		}
		j.AppendByte('"')
		j.AppendString(name)
		j.AppendByte('"')
		j.AppendByte(':')
	}
	for i := range msg.Fields {
		field := &msg.Fields[i]
		if field.Omit == OmitAlways {
			continue
		}
		occ := occurrences[i]
		switch {
		case field.Kind == MapKind:
			if len(occ) == 0 {
				if field.Omit >= OmitEmpty {
					continue
				}
				emitHeader(field.Name)
				j.AppendString("{}")
				continue
			}
			emitHeader(field.Name)
			j.AppendByte('{')
			for k, o := range occ {
				if k > 0 {
					j.AppendByte(',')
				}
				if err := transProtoMapEntry(j, field.Ref, o.val.s); err != nil {
					return err
				}
			}
			j.AppendByte('}')
		case field.Repeated:
			if len(occ) == 0 {
				if field.Omit >= OmitEmpty {
					continue
				}
				emitHeader(field.Name)
				j.AppendString("[]")
				continue
			}
			emitHeader(field.Name)
			if err := transProtoRepeated(j, field, occ); err != nil {
				return err
			}
		default:
			if len(occ) == 0 {
				if field.Omit >= OmitEmpty {
					continue
				}
				emitHeader(field.Name)
				writeDefaultValue(j, false, field.Kind)
				continue
			}
			// proto3 语义：非重复字段重复出现时 last-one-wins。
			emitHeader(field.Name)
			if err := transProtoSingular(j, field, occ[len(occ)-1]); err != nil {
				return err
			}
		}
	}
	j.AppendByte('}')
	return nil
}

// TranscodeToJson 通过 proto.Decoder 解析 pb，并且追加到 JsonBuilder 中
func TranscodeToJson(j *JsonBuilder, p *proto.Decoder, msg *Message) error {
	return transProtoMessage(j, p, msg)
}
