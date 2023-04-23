package jsonlit

import "testing"

func TestEscapeString(t *testing.T) {
	if s := EscapeString(nil, "\t"); string(s) != `\t` {
		t.Fatal(string(s))
	}
	if s := EscapeString(nil, "123\tabc"); string(s) != `123\tabc` {
		t.Fatal(string(s))
	}
}

func TestUnescapeString(t *testing.T) {
	if s, ok := UnescapeString(nil, `\t`); ok && string(s) != "\t" {
		t.Fatal(string(s))
	}
	if s, ok := UnescapeString(nil, `123\tabc`); ok && string(s) != "123\tabc" {
		t.Fatal(string(s))
	}
	if s, ok := UnescapeString(nil, `\u4f60\u597d`); ok && string(s) != "你好" {
		t.Fatal(string(s))
	}
}
