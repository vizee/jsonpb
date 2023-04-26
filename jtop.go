package jsonpb

import (
	"encoding/base64"
	"errors"
	"io"
	"math"
	"strconv"

	"github.com/vizee/jsonpb/jsonlit"
	"github.com/vizee/jsonpb/proto"
)

type JsonIter = jsonlit.Iter[[]byte]

var (
	ErrUnexpectedToken = errors.New("unexpected token")
	ErrTypeMismatch    = errors.New("field type mismatch")
)

func transJsonRepeatedMessage(p *proto.Encoder, j *JsonIter, field *Field) error {
	var buf proto.Encoder
	for !j.EOF() {
		tok, _ := j.Next()
		switch tok {
		case jsonlit.ArrayClose:
			return nil
		case jsonlit.Comma:
		case jsonlit.Object:
			buf.Clear()
			err := transJsonObject(&buf, j, field.Ref)
			if err != nil {
				return err
			}
			p.EmitBytes(field.Tag, buf.Bytes())
		case jsonlit.Null:
			// null 会表达为一个空对象占位
			p.EmitBytes(field.Tag, nil)
		default:
			return ErrUnexpectedToken
		}
	}
	return io.ErrUnexpectedEOF
}

func walkJsonArray(j *JsonIter, expect jsonlit.Kind, f func([]byte) error) error {
	for !j.EOF() {
		tok, s := j.Next()
		switch tok {
		case jsonlit.ArrayClose:
			return nil
		case jsonlit.Comma:
		case expect:
			err := f(s)
			if err != nil {
				return err
			}
		default:
			return ErrUnexpectedToken
		}
	}
	return io.ErrUnexpectedEOF
}

func transJsonArrayField(p *proto.Encoder, j *JsonIter, field *Field) error {
	switch field.Kind {
	case MessageKind:
		return transJsonRepeatedMessage(p, j, field)
	case BytesKind:
		// 暂不允许 null 转到 bytes
		err := walkJsonArray(j, jsonlit.String, func(s []byte) error {
			return transJsonBytes(p, field.Tag, false, s)
		})
		if err != nil {
			return err
		}
	case StringKind:
		err := walkJsonArray(j, jsonlit.String, func(s []byte) error {
			return transJsonString(p, field.Tag, false, s)
		})
		if err != nil {
			return err
		}
	default:
		var (
			packed proto.Encoder
			err    error
		)
		switch field.Kind {
		case DoubleKind:
			err = walkJsonArray(j, jsonlit.Number, func(s []byte) error {
				x, err := strconv.ParseFloat(asString(s), 64)
				if err != nil {
					return err
				}
				packed.WriteFixed64(math.Float64bits(x))
				return nil
			})
		case FloatKind:
			err = walkJsonArray(j, jsonlit.Number, func(s []byte) error {
				x, err := strconv.ParseFloat(asString(s), 32)
				if err != nil {
					return err
				}
				packed.WriteFixed32(math.Float32bits(float32(x)))
				return nil
			})
		case Int32Kind:
			err = walkJsonArray(j, jsonlit.Number, func(s []byte) error {
				x, err := strconv.ParseInt(asString(s), 10, 32)
				if err != nil {
					return err
				}
				packed.WriteVarint(uint64(x))
				return nil
			})
		case Int64Kind:
			err = walkJsonArray(j, jsonlit.Number, func(s []byte) error {
				x, err := strconv.ParseInt(asString(s), 10, 64)
				if err != nil {
					return err
				}
				packed.WriteVarint(uint64(x))
				return nil
			})
		case Uint32Kind:
			err = walkJsonArray(j, jsonlit.Number, func(s []byte) error {
				x, err := strconv.ParseUint(asString(s), 10, 32)
				if err != nil {
					return err
				}
				packed.WriteVarint(x)
				return nil
			})
		case Uint64Kind:
			err = walkJsonArray(j, jsonlit.Number, func(s []byte) error {
				x, err := strconv.ParseUint(asString(s), 10, 64)
				if err != nil {
					return err
				}
				packed.WriteVarint(x)
				return nil
			})
		case Sint32Kind:
			err = walkJsonArray(j, jsonlit.Number, func(s []byte) error {
				x, err := strconv.ParseInt(asString(s), 10, 32)
				if err != nil {
					return err
				}
				packed.WriteZigzag(x)
				return nil
			})
		case Sint64Kind:
			err = walkJsonArray(j, jsonlit.Number, func(s []byte) error {
				x, err := strconv.ParseInt(asString(s), 10, 64)
				if err != nil {
					return err
				}
				packed.WriteZigzag(x)
				return nil
			})
		case Fixed32Kind:
			err = walkJsonArray(j, jsonlit.Number, func(s []byte) error {
				x, err := strconv.ParseUint(asString(s), 10, 32)
				if err != nil {
					return err
				}
				packed.WriteFixed32(uint32(x))
				return nil
			})
		case Fixed64Kind:
			err = walkJsonArray(j, jsonlit.Number, func(s []byte) error {
				x, err := strconv.ParseUint(asString(s), 10, 64)
				if err != nil {
					return err
				}
				packed.WriteFixed64(x)
				return nil
			})
		case Sfixed32Kind:
			err = walkJsonArray(j, jsonlit.Number, func(s []byte) error {
				x, err := strconv.ParseInt(asString(s), 10, 32)
				if err != nil {
					return err
				}
				packed.WriteFixed32(uint32(x))
				return nil
			})
		case Sfixed64Kind:
			err = walkJsonArray(j, jsonlit.Number, func(s []byte) error {
				x, err := strconv.ParseInt(asString(s), 10, 64)
				if err != nil {
					return err
				}
				packed.WriteFixed64(uint64(x))
				return nil
			})
		case BoolKind:
			err = walkJsonArray(j, jsonlit.Bool, func(s []byte) error {
				var x uint64
				if len(s) == 4 {
					x = 1
				} else {
					x = 0
				}
				packed.WriteVarint(x)
				return nil
			})
		default:
			err = ErrTypeMismatch
		}
		if err != nil {
			return err
		}
		if packed.Len() != 0 {
			p.EmitBytes(field.Tag, packed.Bytes())
		}
	}
	return nil
}

func transJsonToMap(p *proto.Encoder, j *JsonIter, tag uint32, entry *Message) error {
	keyField, valueField := entry.FieldByTag(1), entry.FieldByTag(2)
	// assert(keyField != nil && valueField != nil)

	var buf proto.Encoder
	expectValue := false
	for !j.EOF() {
		lead, s := j.Next()
		switch lead {
		case jsonlit.ObjectClose:
			if expectValue {
				return ErrUnexpectedToken
			}
			return nil
		case jsonlit.Comma, jsonlit.Colon:
			// 忽略语法检查
			continue
		default:
			if expectValue {
				// NOTE: transJsonField 会跳过 0 值字段，导致结果比 proto.Marshal 的结果字节数更少，但不影响反序列化结果
				err := transJsonField(&buf, j, valueField, lead, s)
				if err != nil {
					return err
				}
				if buf.Len() != 0 {
					p.EmitBytes(tag, buf.Bytes())
				}
				expectValue = false
			} else if lead == jsonlit.String {
				buf.Clear()
				if keyField.Kind == StringKind {
					err := transJsonString(&buf, 1, true, s)
					if err != nil {
						return err
					}
				} else if IsNumericKind(keyField.Kind) {
					// 允许把 json key 转为将数值类型的 map key
					err := transJsonNumeric(&buf, 1, keyField.Kind, s[1:len(s)-1])
					if err != nil {
						return err
					}
				} else {
					return ErrTypeMismatch
				}
				expectValue = true
			} else {
				return ErrUnexpectedToken
			}
		}
	}
	return io.ErrUnexpectedEOF
}

func transJsonNumeric(p *proto.Encoder, tag uint32, kind Kind, s []byte) error {
	if !IsNumericKind(kind) {
		return ErrTypeMismatch
	}
	// 提前检查 0 值
	if len(s) == 1 && s[0] == '0' {
		return nil
	}
	switch kind {
	case DoubleKind:
		x, err := strconv.ParseFloat(asString(s), 64)
		if err != nil {
			return err
		}
		p.EmitFixed64(tag, math.Float64bits(x))
	case FloatKind:
		x, err := strconv.ParseFloat(asString(s), 32)
		if err != nil {
			return err
		}
		p.EmitFixed32(tag, math.Float32bits(float32(x)))
	case Int32Kind:
		x, err := strconv.ParseInt(asString(s), 10, 32)
		if err != nil {
			return err
		}
		p.EmitVarint(tag, uint64(x))
	case Int64Kind:
		x, err := strconv.ParseInt(asString(s), 10, 64)
		if err != nil {
			return err
		}
		p.EmitVarint(tag, uint64(x))
	case Uint32Kind:
		x, err := strconv.ParseUint(asString(s), 10, 32)
		if err != nil {
			return err
		}
		p.EmitVarint(tag, uint64(x))
	case Uint64Kind:
		x, err := strconv.ParseUint(asString(s), 10, 64)
		if err != nil {
			return err
		}
		p.EmitVarint(tag, x)
	case Sint32Kind:
		x, err := strconv.ParseInt(asString(s), 10, 32)
		if err != nil {
			return err
		}
		p.EmitZigzag(tag, x)
	case Sint64Kind:
		x, err := strconv.ParseInt(asString(s), 10, 64)
		if err != nil {
			return err
		}
		p.EmitZigzag(tag, x)
	case Fixed32Kind:
		x, err := strconv.ParseUint(asString(s), 10, 32)
		if err != nil {
			return err
		}
		p.EmitFixed32(tag, uint32(x))
	case Fixed64Kind:
		x, err := strconv.ParseUint(asString(s), 10, 64)
		if err != nil {
			return err
		}
		p.EmitFixed64(tag, x)
	case Sfixed32Kind:
		x, err := strconv.ParseInt(asString(s), 10, 32)
		if err != nil {
			return err
		}
		p.EmitFixed32(tag, uint32(x))
	case Sfixed64Kind:
		x, err := strconv.ParseInt(asString(s), 10, 64)
		if err != nil {
			return err
		}
		p.EmitFixed64(tag, uint64(x))
	}
	return nil
}

func transJsonString(p *proto.Encoder, tag uint32, omitEmpty bool, s []byte) error {
	if len(s) == 2 && omitEmpty {
		return nil
	}
	z := make([]byte, 0, len(s)-2)
	z, ok := jsonlit.UnescapeString(z, s[1:len(s)-1])
	if !ok {
		return errors.New("unescape malformed string")
	}
	p.EmitBytes(tag, z)
	return nil
}

func transJsonBytes(p *proto.Encoder, tag uint32, omitEmpty bool, s []byte) error {
	if len(s) == 2 && omitEmpty {
		return nil
	}
	z := make([]byte, base64.StdEncoding.DecodedLen(len(s)-2))
	n, err := base64.StdEncoding.Decode(z, s[1:len(s)-1])
	if err != nil {
		return err
	}
	p.EmitBytes(tag, z[:n])
	return nil
}

func transJsonField(p *proto.Encoder, j *JsonIter, field *Field, lead jsonlit.Kind, s []byte) error {
	switch lead {
	case jsonlit.String:
		switch field.Kind {
		case BytesKind:
			return transJsonBytes(p, field.Tag, true, s)
		case StringKind:
			return transJsonString(p, field.Tag, true, s)
		default:
			return ErrTypeMismatch
		}
	case jsonlit.Number:
		return transJsonNumeric(p, field.Tag, field.Kind, s)
	case jsonlit.Bool:
		if field.Kind == BoolKind {
			if len(s) == 4 {
				p.EmitVarint(field.Tag, 1)
			}
			return nil
		} else {
			return ErrTypeMismatch
		}
	case jsonlit.Null:
		// 忽略所有 null
		return nil
	case jsonlit.Object:
		switch field.Kind {
		case MessageKind:
			var buf proto.Encoder
			err := transJsonObject(&buf, j, field.Ref)
			if err != nil {
				return err
			}
			if buf.Len() != 0 {
				p.EmitBytes(field.Tag, buf.Bytes())
			}
			return nil
		case MapKind:
			return transJsonToMap(p, j, field.Tag, field.Ref)
		default:
			return ErrTypeMismatch
		}
	case jsonlit.Array:
		if field.Repeated {
			return transJsonArrayField(p, j, field)
		}
		return ErrTypeMismatch
	}
	return ErrUnexpectedToken
}

func skipJsonValue(j *JsonIter, lead jsonlit.Kind) error {
	switch lead {
	case jsonlit.Null, jsonlit.Bool, jsonlit.Number, jsonlit.String:
		return nil
	case jsonlit.Object:
		for !j.EOF() {
			tok, _ := j.Next()
			switch tok {
			case jsonlit.ObjectClose:
				return nil
			case jsonlit.Comma, jsonlit.Colon:
			default:
				err := skipJsonValue(j, tok)
				if err != nil {
					return err
				}
			}
		}
	case jsonlit.Array:
		for !j.EOF() {
			tok, _ := j.Next()
			switch tok {
			case jsonlit.ArrayClose:
				return nil
			case jsonlit.Comma:
			default:
				err := skipJsonValue(j, tok)
				if err != nil {
					return err
				}
			}
		}
		return io.ErrUnexpectedEOF
	}
	return ErrUnexpectedToken
}

func transJsonObject(p *proto.Encoder, j *JsonIter, msg *Message) error {
	var key []byte
	for !j.EOF() {
		lead, s := j.Next()
		switch lead {
		case jsonlit.ObjectClose:
			if len(key) == 0 {
				return nil
			}
			return ErrUnexpectedToken
		case jsonlit.Comma, jsonlit.Colon:
			// 忽略语法检查
			continue
		default:
			if len(key) != 0 {
				// 暂不转义 key
				field := msg.FieldByName(asString(key[1 : len(key)-1]))
				if field != nil {
					err := transJsonField(p, j, field, lead, s)
					if err != nil {
						return err
					}
				} else {
					err := skipJsonValue(j, lead)
					if err != nil {
						return err
					}
				}
				key = nil
			} else if lead == jsonlit.String {
				key = s
			} else {
				return ErrUnexpectedToken
			}
		}
	}
	return io.ErrUnexpectedEOF
}

// TranscodeToProto 通过 JsonIter 解析 JSON，并且根据 msg 将 JSON 内容转译到 protobuf 二进制。
// 注意，受限于 metadata 可表达的结构和一些取舍，对 JSON 的解析并不按照 JSON 标准。
func TranscodeToProto(p *proto.Encoder, j *JsonIter, msg *Message) error {
	tok, _ := j.Next()
	switch tok {
	case jsonlit.Object:
		return transJsonObject(p, j, msg)
	case jsonlit.EOF:
		return io.ErrUnexpectedEOF
	}
	return ErrUnexpectedToken
}
