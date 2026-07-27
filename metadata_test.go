package jsonpb

import (
	"testing"
)

func TestMessage_noIndex(t *testing.T) {
	m := &Message{
		Fields: []Field{
			{Name: "a", Tag: 1},
			{Name: "b", Tag: 10},
			{Name: "c", Tag: 11},
			{Name: "d", Tag: 20},
		},
	}
	type args struct {
		tag  uint32
		name string
	}
	tests := []struct {
		name string
		args args
	}{
		{name: "a", args: args{tag: 1, name: "a"}},
		{name: "b", args: args{tag: 10, name: "b"}},
		{name: "c", args: args{tag: 11, name: "c"}},
		{name: "d", args: args{tag: 20, name: "d"}},
		{name: "not_found", args: args{tag: 12, name: "e"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if byTag, byName := m.FieldByTag(tt.args.tag), m.FieldByName(tt.args.name); byTag != byName {
				t.Errorf("byTag = %p, byName = %p", byTag, byName)
			}
		})
	}
}

func TestMessage_sparseTagIndex(t *testing.T) {
	m := &Message{
		Fields: []Field{
			{Name: "a", Tag: 1},
			{Name: "b", Tag: 10},
			{Name: "c", Tag: 11},
			{Name: "d", Tag: 20},
		},
	}
	m.BakeTagIndex()

	type args struct {
		tag uint32
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{name: "a", args: args{tag: 1}, want: 0},
		{name: "b", args: args{tag: 10}, want: 1},
		{name: "c", args: args{tag: 11}, want: 2},
		{name: "d", args: args{tag: 20}, want: 3},
		{name: "not_found", args: args{tag: 12}, want: -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := m.FieldIndexByTag(tt.args.tag); got != tt.want {
				t.Errorf("Message.FieldIndexByTag() = %v, want %v", got, tt.want)
			}
		})
	}
}

// #10: 含 tag 0 的 dense 索引（len(tagIdx)==len(Fields)）不应误走二分路径。
func TestMessage_denseWithTag0(t *testing.T) {
	m := NewMessage("M", []Field{
		{Name: "c", Tag: 2},
		{Name: "a", Tag: 0},
		{Name: "b", Tag: 1},
	}, true, true)
	// maxTag=2, len(Fields)=3 => dense，且 len(tagIdx)=3 == len(Fields)
	if m.tagIdxSparse {
		t.Fatal("expected dense index")
	}
	cases := []struct {
		tag  uint32
		name string
	}{
		{0, "a"}, {1, "b"}, {2, "c"}, {3, ""},
	}
	for _, c := range cases {
		f := m.FieldByTag(c.tag)
		if c.name == "" {
			if f != nil {
				t.Errorf("tag %d: expected nil, got %s", c.tag, f.Name)
			}
		} else if f == nil || f.Name != c.name {
			t.Errorf("tag %d: expected %s, got %v", c.tag, c.name, f)
		}
	}
}

// #10: 重复 tag（非法元数据）不应导致 panic（旧实现会因二分查找访问 -1 槽而越界）。
func TestMessage_duplicateTagNoPanic(t *testing.T) {
	m := NewMessage("M", []Field{
		{Name: "a", Tag: 0},
		{Name: "b", Tag: 0},
		{Name: "c", Tag: 2},
	}, true, true)
	if f := m.FieldByTag(2); f == nil || f.Name != "c" {
		t.Errorf("tag 2: expected c, got %v", f)
	}
	if f := m.FieldByTag(1); f != nil {
		t.Errorf("tag 1: expected nil, got %s", f.Name)
	}
}
