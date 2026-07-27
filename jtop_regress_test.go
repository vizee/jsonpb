package jsonpb

import (
	"encoding/hex"
	"testing"

	"github.com/vizee/jsonpb/jsonlit"
	"github.com/vizee/jsonpb/proto"
)

// #11: 浮点零值的多种形式应被省略；整数 "0.0" 仍应报错（保持类型严格）。
func TestRegress_NumericZeroForms(t *testing.T) {
	cases := []struct {
		name    string
		kind    Kind
		s       string
		want    string
		wantErr bool
	}{
		{name: "double_0.0", kind: DoubleKind, s: "0.0", want: ""},
		{name: "double_0e0", kind: DoubleKind, s: "0e0", want: ""},
		{name: "double_-0", kind: DoubleKind, s: "-0", want: ""},
		{name: "double_0.00", kind: DoubleKind, s: "0.00", want: ""},
		{name: "double_1", kind: DoubleKind, s: "1", want: "09000000000000f03f"}, // 1.0
		{name: "int32_0", kind: Int32Kind, s: "0", want: ""},
		{name: "int32_0.0_err", kind: Int32Kind, s: "0.0", wantErr: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := transJsonNumericCase(1, c.kind, []byte(c.s))
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, c.wantErr)
			}
			if !c.wantErr && got != c.want {
				t.Errorf("got %s, want %s", got, c.want)
			}
		})
	}
}

// #4: 默认 key + 默认 value 的 map entry 不应被丢弃。
func TestRegress_MapDefaultEntryKept(t *testing.T) {
	cases := []struct {
		name  string
		j     string
		tag   uint32
		entry *Message
		want  string
	}{
		{
			name:  "empty_string_key_zero_value",
			j:     `{"":0}`,
			tag:   1,
			entry: getTestMapEntry(StringKind, Int32Kind, nil),
			// entry {key:""} => 0a00 ; EmitBytes(tag1, [0a00]) => 0a020a00
			want: "0a020a00",
		},
		{
			name:  "zero_int_key_zero_value",
			j:     `{"0":0}`,
			tag:   1,
			entry: getTestMapEntry(Int32Kind, Int32Kind, nil),
			// entry {key:0} => 0800 ; EmitBytes(tag1, [0800]) => 0a020800
			want: "0a020800",
		},
		{
			name:  "empty_string_key_empty_string_value",
			j:     `{"":""}`,
			tag:   1,
			entry: getTestMapEntry(StringKind, StringKind, nil),
			// entry {key:""} => 0a00 ; EmitBytes(tag1, [0a00]) => 0a020a00
			want: "0a020a00",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var buf proto.Encoder
			it := jsonlit.NewIter([]byte(c.j))
			it.Next()
			if err := transJsonToMap(&buf, it, c.tag, c.entry); err != nil {
				t.Fatal(err)
			}
			if got := hex.EncodeToString(buf.Bytes()); got != c.want {
				t.Errorf("got %s, want %s", got, c.want)
			}
		})
	}
}
