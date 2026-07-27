package jsonpb

import (
	"math"
	"testing"
)

// #2: 重复 string 字段被其它字段分隔时应拼接，而非丢弃后续出现。
func TestRegress_RepeatedStringSplit(t *testing.T) {
	// f19="aaa", f3=5, f19="bbb"
	pb := "9a0103616161" + "1805" + "9a0103626262"
	msg := NewMessage("M", []Field{
		{Name: "f3", Kind: Int32Kind, Tag: 3},
		{Name: "f19", Kind: StringKind, Tag: 19, Repeated: true},
	}, true, true)
	got, err := transProtoMessageCase(pb, msg)
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"f3":5,"f19":["aaa","bbb"]}`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

// #2: 拆分的 packed 数组应拼接。
func TestRegress_PackedSplit(t *testing.T) {
	// f19 packed [1,2], f3=5, f19 packed [3,4]
	pb := "9a01020102" + "1805" + "9a01020304"
	msg := NewMessage("M", []Field{
		{Name: "f3", Kind: Int32Kind, Tag: 3},
		{Name: "f19", Kind: Int32Kind, Tag: 19, Repeated: true},
	}, true, true)
	got, err := transProtoMessageCase(pb, msg)
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"f3":5,"f19":[1,2,3,4]}`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

// #2: 非连续的 map entry 应拼接，且不吸收其它字段。
func TestRegress_MapNonConsecutive(t *testing.T) {
	// m={a:1}, x=5, m={b:2}  (tag16 = 0x82 0x01)
	pb := "8201050a01611001" + "1805" + "8201050a01621002"
	msg := NewMessage("M", []Field{
		{Name: "m", Kind: MapKind, Tag: 16, Ref: getTestMapEntry(StringKind, Int32Kind, nil)},
		{Name: "x", Kind: Int32Kind, Tag: 3},
	}, true, true)
	got, err := transProtoMessageCase(pb, msg)
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"m":{"a":1,"b":2},"x":5}`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

// #3: unpacked repeated 标量应被接受。
func TestRegress_UnpackedRepeated(t *testing.T) {
	// f3 int32 unpacked [1,2,3]
	pb := "18011802" + "1803"
	msg := NewMessage("M", []Field{
		{Name: "f3", Kind: Int32Kind, Tag: 3, Repeated: true},
	}, true, true)
	got, err := transProtoMessageCase(pb, msg)
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"f3":[1,2,3]}`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

// #3: packed 与 unpacked 混合应拼接。
func TestRegress_MixedPackedUnpacked(t *testing.T) {
	// f19 packed [1,2], f3=5, f19 unpacked 3, f19 unpacked 4
	pb := "9a01020102" + "1805" + "980103" + "980104"
	msg := NewMessage("M", []Field{
		{Name: "f3", Kind: Int32Kind, Tag: 3},
		{Name: "f19", Kind: Int32Kind, Tag: 19, Repeated: true},
	}, true, true)
	got, err := transProtoMessageCase(pb, msg)
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"f3":5,"f19":[1,2,3,4]}`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

// #7: 非重复字段重复出现时 last-one-wins。
func TestRegress_LastOneWins(t *testing.T) {
	// f3=5 then f3=9
	pb := "1805" + "1809"
	msg := NewMessage("M", []Field{
		{Name: "f3", Kind: Int32Kind, Tag: 3},
	}, true, true)
	got, err := transProtoMessageCase(pb, msg)
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"f3":9}`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

// #9: NaN/±Inf 按 protobuf JSON 规范输出。
func TestRegress_FloatSpecial(t *testing.T) {
	cases := []struct {
		kind Kind
		x    uint64
		want string
	}{
		{DoubleKind, math.Float64bits(math.NaN()), "NaN"},
		{DoubleKind, math.Float64bits(math.Inf(1)), "Infinity"},
		{DoubleKind, math.Float64bits(math.Inf(-1)), "-Infinity"},
		{FloatKind, uint64(math.Float32bits(float32(math.NaN()))), "NaN"},
	}
	for _, c := range cases {
		var j JsonBuilder
		transProtoSimpleValue(&j, c.kind, c.x)
		if got := j.String(); got != c.want {
			t.Errorf("kind=%v x=%d: got %s, want %s", c.kind, c.x, got, c.want)
		}
	}
}
