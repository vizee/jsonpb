package jsonpb

import (
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/vizee/jsonpb/proto"
	"google.golang.org/protobuf/encoding/protowire"
)

func decodeBytes(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

func readProtoValueCase(s string, wire protowire.Type) (protoValue, int) {
	return readProtoValue(proto.NewDecoder(decodeBytes(s)), wire)
}

func Test_readProtoValueCase(t *testing.T) {
	type args struct {
		s    string
		wire protowire.Type
	}
	tests := []struct {
		name  string
		args  args
		want  protoValue
		want1 int
	}{
		{name: "varint", args: args{s: "7b", wire: protowire.VarintType}, want: protoValue{x: 123}},
		{name: "fixed32", args: args{s: "7b000000", wire: protowire.Fixed32Type}, want: protoValue{x: 123}},
		{name: "fixed64", args: args{s: "7b00000000000000", wire: protowire.Fixed64Type}, want: protoValue{x: 123}},
		{name: "bytes", args: args{s: "036f6b6b", wire: protowire.BytesType}, want: protoValue{s: []byte("okk")}},
		{name: "bad_wire", args: args{s: "", wire: protowire.StartGroupType}, want: protoValue{}, want1: -100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := readProtoValueCase(tt.args.s, tt.args.wire)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("readProtoValueCase() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("readProtoValueCase() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func transProtoBytesCase(s string) string {
	var j JsonBuilder
	transProtoBytes(&j, decodeBytes(s))
	return j.String()
}

func Test_transProtoBytesCase(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "empty", args: args{s: ""}, want: `""`},
		{name: "hello", args: args{s: "68656c6c6f"}, want: `"aGVsbG8="`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := transProtoBytesCase(tt.args.s)
			if got != tt.want {
				t.Errorf("transProtoBytesCase() = %v, want %v", got, tt.want)
			}
		})
	}
}

func transProtoStringCase(s string) string {
	var j JsonBuilder
	transProtoString(&j, decodeBytes(s))
	return j.String()
}

func Test_transProtoStringCase(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "empty", args: args{s: ""}, want: `""`},
		{name: "hello", args: args{s: "68656c6c6f"}, want: `"hello"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := transProtoStringCase(tt.args.s)
			if got != tt.want {
				t.Errorf("transProtoStringCase() = %v, want %v", got, tt.want)
			}
		})
	}
}

func transProtoSimpleValueCase(kind Kind, s string) string {
	pv, _ := readProtoValue(proto.NewDecoder(decodeBytes(s)), getFieldWireType(kind, false))
	var j JsonBuilder
	transProtoSimpleValue(&j, kind, pv.x)
	return j.String()
}

func Test_transProtoSimpleValueCase(t *testing.T) {
	type args struct {
		kind Kind
		s    string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "double", args: args{kind: DoubleKind, s: "ae47e17a14aef33f"}, want: "1.23"},
		{name: "float", args: args{kind: FloatKind, s: "a4709d3f"}, want: "1.23"},
		{name: "int32", args: args{kind: Int32Kind, s: "7b"}, want: "123"},
		{name: "int64", args: args{kind: Int64Kind, s: "7b"}, want: "123"},
		{name: "uint32", args: args{kind: Uint32Kind, s: "7b"}, want: "123"},
		{name: "uint64", args: args{kind: Uint64Kind, s: "7b"}, want: "123"},
		{name: "sint32", args: args{kind: Sint32Kind, s: "f501"}, want: "-123"},
		{name: "sint64", args: args{kind: Sint64Kind, s: "f501"}, want: "-123"},
		{name: "fixed32", args: args{kind: Fixed32Kind, s: "7b000000"}, want: "123"},
		{name: "fixed64", args: args{kind: Fixed64Kind, s: "7b00000000000000"}, want: "123"},
		{name: "sfixed32", args: args{kind: Sfixed32Kind, s: "85ffffff"}, want: "-123"},
		{name: "sfixed64", args: args{kind: Sfixed64Kind, s: "85ffffffffffffff"}, want: "-123"},
		{name: "bool_true", args: args{kind: BoolKind, s: "01"}, want: "true"},
		{name: "bool_false", args: args{kind: BoolKind, s: "00"}, want: "false"},
		{name: "unexpected_kind", args: args{kind: StringKind, s: "00"}, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := transProtoSimpleValueCase(tt.args.kind, tt.args.s); got != tt.want {
				t.Errorf("transProtoSimpleValueCase() = %v, want %v", got, tt.want)
			}
		})
	}
}

func transProtoRepeatedBytesCase(p string, field *Field, s string) (string, error) {
	var j JsonBuilder
	err := transProtoRepeatedBytes(&j, proto.NewDecoder(decodeBytes(p)), field, decodeBytes(s))
	if err != nil {
		return "", err
	}
	return j.String(), nil
}

func Test_transProtoRepeatedBytesCase(t *testing.T) {
	type args struct {
		p     string
		field *Field
		s     string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{name: "one_string", args: args{p: "", field: &Field{Tag: 1, Kind: StringKind}, s: "616263"}, want: `["abc"]`},
		{name: "more_strings", args: args{p: "0a0568656c6c6f0a05776f726c64", field: &Field{Tag: 1, Kind: StringKind}, s: "616263"}, want: `["abc","hello","world"]`},
		{name: "more_bytes", args: args{p: "0a0568656c6c6f0a05776f726c64", field: &Field{Tag: 1, Kind: BytesKind}, s: "616263"}, want: `["YWJj","aGVsbG8=","d29ybGQ="]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := transProtoRepeatedBytesCase(tt.args.p, tt.args.field, tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("transProtoRepeatedBytesCase() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("transProtoRepeatedBytesCase() = %v, want %v", got, tt.want)
			}
		})
	}
}

func transProtoPackedArrayCase(p string, field *Field) (string, error) {
	var j JsonBuilder
	err := transProtoPackedArray(&j, decodeBytes(p), field)
	if err != nil {
		return "", err
	}
	return j.String(), nil
}

func Test_transProtoPackedArrayCase(t *testing.T) {
	type args struct {
		p     string
		field *Field
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{name: "empty", args: args{p: "", field: &Field{Tag: 1, Kind: Int32Kind}}, want: `[]`},
		{name: "int32s", args: args{p: "7bc8039506", field: &Field{Tag: 1, Kind: Int32Kind}}, want: `[123,456,789]`},
		{name: "doubles", args: args{p: "ae47e17a14aef33f3d0ad7a3703d12408fc2f5285c8f1f40", field: &Field{Tag: 1, Kind: DoubleKind}}, want: `[1.23,4.56,7.89]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := transProtoPackedArrayCase(tt.args.p, tt.args.field)
			if (err != nil) != tt.wantErr {
				t.Errorf("transProtoPackedArrayCase() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("transProtoPackedArrayCase() = %v, want %v", got, tt.want)
			}
		})
	}
}

func transProtoMapCase(p string, tag uint32, entry *Message, s string) (string, error) {
	var j JsonBuilder
	err := transProtoMap(&j, proto.NewDecoder(decodeBytes(p)), tag, entry, decodeBytes(s))
	if err != nil {
		return "", err
	}
	return j.String(), nil
}

func Test_transProtoMapCase(t *testing.T) {
	type args struct {
		p     string
		tag   uint32
		entry *Message
		s     string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{name: "empty", args: args{p: "", tag: 1, entry: getTestMapEntry(StringKind, Int32Kind, nil), s: ""}, want: `{"":0}`},
		{name: "simple", args: args{p: "8201050a01621002", tag: 16, entry: getTestMapEntry(StringKind, Int32Kind, nil), s: "0a01611001"}, want: `{"a":1,"b":2}`},
		{name: "stop", args: args{p: "8201050a01621002", tag: 17, entry: getTestMapEntry(StringKind, Int32Kind, nil), s: "0a01611001"}, want: `{"a":1}`},
		{name: "int_key", args: args{p: "0a0608c803120162", tag: 1, entry: getTestMapEntry(Int32Kind, StringKind, nil), s: "087b120161"}, want: `{"123":"a","456":"b"}`},
		{name: "bytes_value", args: args{p: "", tag: 1, entry: getTestMapEntry(StringKind, BytesKind, nil), s: "0a0568656c6c6f1205776f726c64"}, want: `{"hello":"d29ybGQ="}`},
		{name: "message_value", args: args{p: "", tag: 1, entry: getTestMapEntry(StringKind, MessageKind, getTestSimpleMessage()), s: "0a0361626312090a03626f6210171801"}, want: `{"abc":{"name":"bob","age":23}}`},
		{name: "default_key", args: args{p: "", tag: 1, entry: getTestMapEntry(StringKind, Int32Kind, nil), s: "107b"}, want: `{"":123}`},
		{name: "default_int32_value", args: args{p: "", tag: 1, entry: getTestMapEntry(StringKind, Int32Kind, nil), s: "0a0161"}, want: `{"a":0}`},
		{name: "default_string_value", args: args{p: "", tag: 1, entry: getTestMapEntry(StringKind, StringKind, nil), s: "0a0161"}, want: `{"a":""}`},
		{name: "default_message_value", args: args{p: "", tag: 1, entry: getTestMapEntry(StringKind, MessageKind, getTestSimpleMessage()), s: "0a0161"}, want: `{"a":{}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := transProtoMapCase(tt.args.p, tt.args.tag, tt.args.entry, tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("transProtoMapCase() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("transProtoMapCase() = %v, want %v", got, tt.want)
			}
		})
	}
}

func transProtoMessageCase(p string, msg *Message) (string, error) {
	var j JsonBuilder
	err := transProtoMessage(&j, proto.NewDecoder(decodeBytes(p)), msg)
	if err != nil {
		return "", err
	}
	return j.String(), nil
}

func Test_transProtoMessageCase(t *testing.T) {
	const (
		complexProto       = `090000000000c05e40150000f642187b207b287b307b38f60140f6014d7b000000517b000000000000005d7b000000617b00000000000000680172036f6b6b7a030102038201050a016b10018a010e0a017512090a03616263101718018a01050a017612009201090a03656667101718019a0103010203a201090a03616263100c1801a20100a201070a036566671017`
		complexWant        = `{"fdouble":123,"ffloat":123,"fint32":123,"fint64":123,"fuint32":123,"fuint64":123,"fsint32":123,"fsint64":123,"ffixed32":123,"ffixed64":123,"fsfixed32":123,"fsfixed64":123,"fbool":true,"fstring":"okk","fbytes":"AQID","fmap1":{"k":1},"fmap2":{"u":{"name":"abc","age":23},"v":{}},"fsubmsg":{"name":"efg","age":23},"fint32s":[1,2,3],"fitems":[{"name":"abc","age":12},{},{"name":"efg","age":23}]}`
		complexDefaultWant = `{"fdouble":0,"ffloat":0,"fint32":0,"fint64":0,"fuint32":0,"fuint64":0,"fsint32":0,"fsint64":0,"ffixed32":0,"ffixed64":0,"fsfixed32":0,"fsfixed64":0,"fbool":false,"fstring":"","fbytes":"","fmap1":{},"fmap2":{},"fsubmsg":{},"fint32s":[],"fitems":[]}`
	)

	type args struct {
		p   string
		msg *Message
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{name: "empty", args: args{p: "", msg: getTestSimpleMessage()}, want: `{}`},
		{name: "simple", args: args{p: "0a03626f6210171801", msg: getTestSimpleMessage()}, want: `{"name":"bob","age":23}`},
		{name: "simple2", args: args{p: "0a03626f6210171801", msg: getTestSimpleMessage2()}, want: `{"male":true}`},
		{name: "emitted", args: args{p: "0a03626f6210170a03626f621801", msg: getTestSimpleMessage()}, want: `{"name":"bob","age":23}`},
		{name: "complex", args: args{p: complexProto, msg: getTestComplexMessage()}, want: complexWant},
		{name: "default", args: args{p: "", msg: getTestComplexMessage()}, want: complexDefaultWant},
		{name: "eof", args: args{p: "0a", msg: getTestComplexMessage()}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := transProtoMessageCase(tt.args.p, tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("transProtoMessageCase() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("transProtoMessageCase() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTranscodeToJson(t *testing.T) {
	type args struct {
		j   *JsonBuilder
		p   *proto.Decoder
		msg *Message
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "simple", args: args{j: &JsonBuilder{}, p: proto.NewDecoder(decodeBytes("")), msg: getTestSimpleMessage()}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := TranscodeToJson(tt.args.j, tt.args.p, tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("TranscodeToJson() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
