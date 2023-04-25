package jsonpb

import "unsafe"

func asString(s []byte) string {
	return unsafe.String(unsafe.SliceData(s), len(s))
}
