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
