package jsonlit

import (
	"encoding/hex"
	"testing"
)

func TestRegress_NextStringBackslash(t *testing.T) {
	cases := []struct {
		in   string
		want Kind
		tok  string
	}{
		{`"\\"`, String, `"\\"`},   // value: single backslash
		{`"a\\"`, String, `"a\\"`}, // value: a\
		{`"C:\\dir\\"`, String, `"C:\\dir\\"`},
		{`"\\"\\b"`, String, `"\\"`}, // two adjacent strings; first must close at index 3
		{`"\""`, String, `"\""`},     // escaped quote
		{`"\n\t"`, String, `"\n\t"`},
		{`"plain"`, String, `"plain"`},
	}
	for _, c := range cases {
		it := NewIter([]byte(c.in))
		k, s := it.Next()
		if k != c.want || string(s) != c.tok {
			t.Errorf("input=%q: got kind=%v tok=%q, want kind=%v tok=%q", c.in, k, string(s), c.want, c.tok)
		}
	}
}

func TestRegress_EscapeControl(t *testing.T) {
	// EscapeString does not add surrounding quotes; caller does.
	cases := []struct {
		in   string
		want string
	}{
		{"a\x00b", `a\u0000b`},
		{"\x0b", `\u000b`},
		{"\x1f", `\u001f`},
		{"\x07", `\u0007`},
		{"\b\t\n\f\r", `\b\t\n\f\r`}, // short escapes preserved
		{`a\b"c`, `a\\b\"c`},
	}
	for _, c := range cases {
		got := string(EscapeString(nil, c.in))
		if got != c.want {
			t.Errorf("EscapeString(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestRegress_SurrogatePair(t *testing.T) {
	out, ok := UnescapeString(nil, `\uD83D\uDE00`)
	if !ok || hex.EncodeToString(out) != "f09f9880" {
		t.Errorf("surrogate pair: ok=%v hex=%s, want f09f9880", ok, hex.EncodeToString(out))
	}
	// BMP char still works
	out, ok = UnescapeString(nil, `\u4f60`)
	if !ok || string(out) != "你" {
		t.Errorf("bmp: ok=%v got %q", ok, string(out))
	}
	// lone high surrogate -> not an error, encodes as replacement
	_, ok = UnescapeString(nil, `\uD83D`)
	if !ok {
		t.Errorf("lone surrogate should not error")
	}
}
