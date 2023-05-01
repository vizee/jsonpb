package jsonpb

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/vizee/jsonpb/jsonlit"
	"github.com/vizee/jsonpb/proto"
)

func skipJsonValueCase(j string) error {
	it := jsonlit.NewIter([]byte(j))
	tok, _ := it.Next()
	err := skipJsonValue(it, tok)
	if err != nil {
		return err
	}
	if !it.EOF() {
		return fmt.Errorf("incomplete")
	}
	return nil
}

func Test_skipJsonValue(t *testing.T) {
	tests := []struct {
		name    string
		arg     string
		wantErr bool
	}{
		{name: "array", arg: `[1,"hello",false,{"k1":"v1","k2":null}]`, wantErr: false},
		{name: "object", arg: `{"a":1,"b":"hello","c":[1,"hello",false,{"k1":"v1","k2":"v2"}],"d":{"k1":"v1","k2":"v2"}}`, wantErr: false},
		{name: "ignore_syntax", arg: `{"a" 1 "b" "hello" "c":[1 "hello" false {"k1":"v1","k2":"v2"}],"d":{"k1":"v1","k2":"v2"}}`, wantErr: false},
		{name: "bad_token", arg: `:`, wantErr: true},
		{name: "unterminated", arg: `{"k1":1,"k2":2`, wantErr: true},
		{name: "unterminated_sub", arg: `{"k1":1,"k2":[`, wantErr: true},
		{name: "unexpected_token", arg: `{"k1":1,"k2":[}`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := skipJsonValueCase(tt.arg); (err != nil) != tt.wantErr {
				t.Errorf("skipJsonValueCase() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func transJsonBytesCase(tag uint32, omitEmpty bool, s []byte) (string, error) {
	var buf proto.Encoder
	err := transJsonBytes(&buf, tag, omitEmpty, s)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(buf.Bytes()), nil
}

func Test_transJsonBytes(t *testing.T) {
	type args struct {
		tag       uint32
		omitEmpty bool
		s         []byte
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{name: "empty", args: args{tag: 1, s: []byte(`""`)}, want: "0a00"},
		{name: "omit_empty", args: args{tag: 1, omitEmpty: true, s: []byte(`""`)}, want: ""},
		{name: "simple", args: args{tag: 1, s: []byte(`"aGVsbG8gd29ybGQ="`)}, want: "0a0b68656c6c6f20776f726c64"},
		{name: "illegal_base64", args: args{tag: 1, s: []byte(`"aGVsbG8gd29ybGQ"`)}, want: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := transJsonBytesCase(tt.args.tag, tt.args.omitEmpty, tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("transJsonBytesCase() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("transJsonBytesCase() = %v, want %v", got, tt.want)
			}
		})
	}
}

func transJsonStringCase(tag uint32, omitEmpty bool, s []byte) (string, error) {
	var buf proto.Encoder
	err := transJsonString(&buf, tag, omitEmpty, s)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(buf.Bytes()), nil
}

func Test_transJsonStringCase(t *testing.T) {
	type args struct {
		tag       uint32
		omitEmpty bool
		s         []byte
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{name: "empty", args: args{tag: 1, s: []byte(`""`)}, want: "0a00"},
		{name: "omit_empty", args: args{tag: 1, omitEmpty: true, s: []byte(`""`)}, want: ""},
		{name: "simple", args: args{tag: 1, s: []byte(`"hello world"`)}, want: "0a0b68656c6c6f20776f726c64"},
		{name: "escape", args: args{tag: 1, s: []byte(`"\u4f60\u597d"`)}, want: "0a06e4bda0e5a5bd"},
		{name: "illegal_escape", args: args{tag: 1, s: []byte(`"\z"`)}, want: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := transJsonStringCase(tt.args.tag, tt.args.omitEmpty, tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("transJsonStringCase() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("transJsonStringCase() = %v, want %v", got, tt.want)
			}
		})
	}
}

func transJsonNumericCase(tag uint32, kind Kind, s []byte) (string, error) {
	var buf proto.Encoder
	err := transJsonNumeric(&buf, tag, kind, s)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(buf.Bytes()), nil
}

func Test_transJsonNumeric(t *testing.T) {
	type args struct {
		tag  uint32
		kind Kind
		s    []byte
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{name: "omit_zero", args: args{tag: 1, kind: Int32Kind, s: []byte(`0`)}, want: ""},
		{name: "double", args: args{tag: 1, kind: DoubleKind, s: []byte(`1`)}, want: "09000000000000f03f"},
		{name: "bad_double", args: args{tag: 1, kind: DoubleKind, s: []byte(`a`)}, wantErr: true},
		{name: "float", args: args{tag: 2, kind: FloatKind, s: []byte(`2`)}, want: "1500000040"},
		{name: "bad_float", args: args{tag: 2, kind: FloatKind, s: []byte(`a`)}, wantErr: true},
		{name: "int32", args: args{tag: 3, kind: Int32Kind, s: []byte(`3`)}, want: "1803"},
		{name: "bad_int32", args: args{tag: 3, kind: Int32Kind, s: []byte(`a`)}, wantErr: true},
		{name: "int64", args: args{tag: 4, kind: Int64Kind, s: []byte(`4`)}, want: "2004"},
		{name: "bad_int64", args: args{tag: 4, kind: Int64Kind, s: []byte(`a`)}, wantErr: true},
		{name: "uint32", args: args{tag: 5, kind: Uint32Kind, s: []byte(`5`)}, want: "2805"},
		{name: "bad_uint32", args: args{tag: 5, kind: Uint32Kind, s: []byte(`a`)}, wantErr: true},
		{name: "uint64", args: args{tag: 6, kind: Uint64Kind, s: []byte(`6`)}, want: "3006"},
		{name: "bad_uint64", args: args{tag: 6, kind: Uint64Kind, s: []byte(`a`)}, wantErr: true},
		{name: "sint32", args: args{tag: 7, kind: Sint32Kind, s: []byte(`7`)}, want: "380e"},
		{name: "bad_sint32", args: args{tag: 7, kind: Sint32Kind, s: []byte(`a`)}, wantErr: true},
		{name: "sint64", args: args{tag: 8, kind: Sint64Kind, s: []byte(`8`)}, want: "4010"},
		{name: "bad_sint64", args: args{tag: 8, kind: Sint64Kind, s: []byte(`a`)}, wantErr: true},
		{name: "fixed32", args: args{tag: 9, kind: Fixed32Kind, s: []byte(`9`)}, want: "4d09000000"},
		{name: "bad_fixed32", args: args{tag: 9, kind: Fixed32Kind, s: []byte(`a`)}, wantErr: true},
		{name: "fixed64", args: args{tag: 10, kind: Fixed64Kind, s: []byte(`10`)}, want: "510a00000000000000"},
		{name: "bad_fixed64", args: args{tag: 10, kind: Fixed64Kind, s: []byte(`a`)}, wantErr: true},
		{name: "sfixed32", args: args{tag: 11, kind: Sfixed32Kind, s: []byte(`11`)}, want: "5d0b000000"},
		{name: "bad_sfixed32", args: args{tag: 11, kind: Sfixed32Kind, s: []byte(`a`)}, wantErr: true},
		{name: "sfixed64", args: args{tag: 12, kind: Sfixed64Kind, s: []byte(`12`)}, want: "610c00000000000000"},
		{name: "bad_sfixed64", args: args{tag: 12, kind: Sfixed64Kind, s: []byte(`a`)}, wantErr: true},
		{name: "invalid_kind", args: args{tag: 1, kind: BoolKind, s: []byte(`1`)}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := transJsonNumericCase(tt.args.tag, tt.args.kind, tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("transJsonNumericCase() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("transJsonNumericCase() = %v, want %v", got, tt.want)
			}
		})
	}
}

func transJsonArrayFieldCase(j string, field *Field) (string, error) {
	var buf proto.Encoder
	it := jsonlit.NewIter([]byte(j))
	it.Next()
	err := transJsonArrayField(&buf, it, field)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(buf.Bytes()), nil
}

func Test_transJsonArrayFieldCase(t *testing.T) {
	type args struct {
		j     string
		field *Field
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{name: "empty", args: args{j: `[]`, field: &Field{Tag: 1, Kind: Int32Kind, Repeated: true}}, want: ""},
		{name: "strings", args: args{j: `["hello","中文","🚀"]`, field: &Field{Tag: 2, Kind: StringKind, Repeated: true}}, want: "120568656c6c6f1206e4b8ade696871204f09f9a80"},
		{name: "bytes", args: args{j: `["YWJj","aGVsbG8=","d29ybGQ="]`, field: &Field{Tag: 2, Kind: BytesKind, Repeated: true}}, want: "1203616263120568656c6c6f1205776f726c64"},
		{name: "packed_double", args: args{j: `[0,1,2]`, field: &Field{Tag: 2, Kind: DoubleKind, Repeated: true}}, want: "12180000000000000000000000000000f03f0000000000000040"},
		{name: "packed_float", args: args{j: `[0,1,2]`, field: &Field{Tag: 2, Kind: FloatKind, Repeated: true}}, want: "120c000000000000803f00000040"},
		{name: "packed_int32", args: args{j: `[0,1,2]`, field: &Field{Tag: 2, Kind: Int32Kind, Repeated: true}}, want: "1203000102"},
		{name: "packed_int64", args: args{j: `[0,1,2]`, field: &Field{Tag: 2, Kind: Int64Kind, Repeated: true}}, want: "1203000102"},
		{name: "packed_uint32", args: args{j: `[0,1,2]`, field: &Field{Tag: 2, Kind: Uint32Kind, Repeated: true}}, want: "1203000102"},
		{name: "packed_uint64", args: args{j: `[0,1,2]`, field: &Field{Tag: 2, Kind: Uint64Kind, Repeated: true}}, want: "1203000102"},
		{name: "packed_sint32", args: args{j: `[0,1,2]`, field: &Field{Tag: 2, Kind: Sint32Kind, Repeated: true}}, want: "1203000204"},
		{name: "packed_sint64", args: args{j: `[0,1,2]`, field: &Field{Tag: 2, Kind: Sint64Kind, Repeated: true}}, want: "1203000204"},
		{name: "packed_fixed32", args: args{j: `[0,1,2]`, field: &Field{Tag: 2, Kind: Fixed32Kind, Repeated: true}}, want: "120c000000000100000002000000"},
		{name: "packed_fixed64", args: args{j: `[0,1,2]`, field: &Field{Tag: 2, Kind: Fixed64Kind, Repeated: true}}, want: "1218000000000000000001000000000000000200000000000000"},
		{name: "packed_sfixed32", args: args{j: `[0,1,2]`, field: &Field{Tag: 2, Kind: Sfixed32Kind, Repeated: true}}, want: "120c000000000100000002000000"},
		{name: "packed_sfixed64", args: args{j: `[0,1,2]`, field: &Field{Tag: 2, Kind: Sfixed64Kind, Repeated: true}}, want: "1218000000000000000001000000000000000200000000000000"},
		{name: "packed_bool", args: args{j: `[false,true,false]`, field: &Field{Tag: 2, Kind: BoolKind, Repeated: true}}, want: "1203000100"},
		{name: "messages", args: args{j: `[{},null,{"name":"string","age":123},{"age":456}]`, field: &Field{Tag: 2, Kind: MessageKind, Ref: getTestSimpleMessage(), Repeated: true}}, want: "12001200120a0a06737472696e67107b120310c803"},
		{name: "unterminated", args: args{j: `[0,1,2`, field: &Field{Tag: 2, Kind: Int32Kind, Repeated: true}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := transJsonArrayFieldCase(tt.args.j, tt.args.field)
			if (err != nil) != tt.wantErr {
				t.Errorf("transJsonArrayFieldCase() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("transJsonArrayFieldCase() = %v, want %v", got, tt.want)
			}
		})
	}
}

func transJsonToMapCase(j string, tag uint32, entry *Message) (string, error) {
	var buf proto.Encoder
	it := jsonlit.NewIter([]byte(j))
	it.Next()
	err := transJsonToMap(&buf, it, tag, entry)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(buf.Bytes()), nil
}

func Test_transJsonToMapCase(t *testing.T) {
	type args struct {
		j     string
		tag   uint32
		entry *Message
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{name: "empty", args: args{j: `{}`, tag: 2, entry: getTestMapEntry(StringKind, Int32Kind, nil)}, want: ""},
		{name: "string_key", args: args{j: `{"a":1,"b":2}`, tag: 2, entry: getTestMapEntry(StringKind, Int32Kind, nil)}, want: "12050a0161100112050a01621002"},
		{name: "numeric_key", args: args{j: `{"1":"a","2":"b"}`, tag: 2, entry: getTestMapEntry(Int32Kind, StringKind, nil)}, want: "1205080112016112050802120162"},
		{name: "message_value", args: args{j: `{"v":{"name":"ok"}}`, tag: 2, entry: getTestMapEntry(StringKind, MessageKind, getTestSimpleMessage())}, want: "12090a017612040a026f6b"},
		{name: "type_mismatched", args: args{j: `{"1":"a","2":"b"}`, tag: 2, entry: getTestMapEntry(BoolKind, StringKind, nil)}, wantErr: true},
		{name: "unexpected_key", args: args{j: `{null`, tag: 2, entry: getTestMapEntry(StringKind, Int32Kind, nil)}, wantErr: true},
		{name: "unexpected_termination", args: args{j: `{"key":}`, tag: 2, entry: getTestMapEntry(StringKind, Int32Kind, nil)}, wantErr: true},
		{name: "eof", args: args{j: `{`, tag: 2, entry: getTestMapEntry(StringKind, Int32Kind, nil)}, wantErr: true},
		{name: "zero_value", args: args{j: `{"v":0}`, tag: 3, entry: getTestMapEntry(StringKind, Int32Kind, nil)}, want: "1a030a0176"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := transJsonToMapCase(tt.args.j, tt.args.tag, tt.args.entry)
			if (err != nil) != tt.wantErr {
				t.Errorf("transJsonToMapCase() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("transJsonToMapCase() = %v, want %v", got, tt.want)
			}
		})
	}
}

func transJsonObjectCase(j string, msg *Message) (string, error) {
	var buf proto.Encoder
	it := jsonlit.NewIter([]byte(j))
	it.Next()
	err := transJsonObject(&buf, it, msg)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(buf.Bytes()), nil
}

func Test_transJsonObjectCase(t *testing.T) {
	const (
		complexJson = `{"noexisted":null,"fdouble":123,"ffloat":123,"fint32":123,"fint64":123,"fuint32":123,"fuint64":123,"fsint32":123,"fsint64":123,"ffixed32":123,"ffixed64":123,"fsfixed32":123,"fsfixed64":123,"fbool":true,"fstring":"okk","fbytes":"AQID","fmap1":{"k":1},"fmap2":{"u":{"name":"abc","age":23,"male":true},"v":null},"fsubmsg":{"name":"efg","age":23,"male":true},"fint32s":[1,2,3],"fitems":[{"name":"abc","age":12,"male":true},null,{"name":"efg","age":23}]}`
		complexWant = `090000000000c05e40150000f642187b207b287b307b38f60140f6014d7b000000517b000000000000005d7b000000617b00000000000000680172036f6b6b7a030102038201050a016b10018a010c0a017512070a0361626310178a01030a01769201070a0365666710179a0103010203a201070a03616263100ca20100a201070a036566671017`
	)
	type args struct {
		j   string
		msg *Message
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{name: "empty", args: args{j: `{}`, msg: getTestSimpleMessage()}, want: ""},
		{name: "simple", args: args{j: `{"name":"string","age":123,"male":true}`, msg: getTestSimpleMessage()}, want: "0a06737472696e67107b"},
		{name: "simple2", args: args{j: `{"name":"string","age":123,"male":true}`, msg: getTestSimpleMessage2()}, want: "1801"},
		{name: "complex", args: args{j: complexJson, msg: getTestComplexMessage()}, want: complexWant},
		{name: "eof", args: args{j: `{`, msg: getTestSimpleMessage()}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := transJsonObjectCase(tt.args.j, tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("transJsonObjectCase() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("transJsonObjectCase() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTranscodeToProto(t *testing.T) {
	type args struct {
		p   *proto.Encoder
		j   *JsonIter
		msg *Message
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "simple", args: args{p: proto.NewEncoder(nil), j: jsonlit.NewIter([]byte(`{}`)), msg: getTestSimpleMessage()}},
		{name: "not_object", args: args{p: proto.NewEncoder(nil), j: jsonlit.NewIter([]byte(`[]`)), msg: getTestSimpleMessage()}, wantErr: true},
		{name: "eof", args: args{p: proto.NewEncoder(nil), j: jsonlit.NewIter([]byte(``)), msg: getTestSimpleMessage()}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := TranscodeToProto(tt.args.p, tt.args.j, tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("TranscodeToProto() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
