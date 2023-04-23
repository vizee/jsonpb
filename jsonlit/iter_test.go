package jsonlit

import (
	"testing"
)

func TestIter(t *testing.T) {
	const input = "{\"animals\":{\"dog\":[{\"name\":\"Rufus\",\"age\":15,\"is_male\":true},{\"name\":\"Marty\",\"age\":null,\"is_male\":false}]}}"
	it := NewIter(input)
	for !it.EOF() {
		k, s := it.Next()
		if k == Invalid {
			t.Fatal(k, string(s))
		}
		t.Log(string(s))
	}
	eof, _ := it.Next()
	if eof != EOF {
		t.Fatal(eof)
	}
}
