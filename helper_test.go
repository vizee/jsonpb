package jsonpb

import (
	"testing"
)

func Test_asString(t *testing.T) {
	type args struct {
		s []byte
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "empty", args: args{s: []byte("")}, want: ""},
		{name: "abc", args: args{s: []byte("abc")}, want: "abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := asString(tt.args.s); got != tt.want {
				t.Errorf("asString() = %v, want %v", got, tt.want)
			}
		})
	}
}
