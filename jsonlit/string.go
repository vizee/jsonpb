package jsonlit

import (
	"unicode/utf8"
)

const (
	rawMark       = '0'
	escapeTable   = "00000000btn0fr00000000000000000000\"000000000000/00000000000000000000000000000000000000000000\\"
	unescapeTable = "0000000000000000000000000000000000\"000000000000/00000000000000000000000000000000000000000000\\00000\x08000\x0C0000000\n000\r0\tu"
)

func EscapeString[S Bytes](dst []byte, s S) []byte {
	begin := 0
	i := 0
	for i < len(s) {
		c := s[i]
		if int(c) >= len(escapeTable) || escapeTable[c] == rawMark {
			i++
			continue
		}
		if begin < i {
			dst = append(dst, s[begin:i]...)
		}
		dst = append(dst, '\\', escapeTable[c])
		i++
		begin = i
	}
	if begin < len(s) {
		dst = append(dst, s[begin:]...)
	}
	return dst
}

func UnescapeString[S Bytes](dst []byte, s S) ([]byte, bool) {
	i := 0
	for i < len(s) {
		c := s[i]
		if c == '\\' {
			i++
			if i >= len(s) {
				return nil, false
			}
			c = s[i]
			if int(c) >= len(unescapeTable) || unescapeTable[c] == rawMark {
				return nil, false
			}
			if c == 'u' {
				if i+4 >= len(s) {
					return nil, false
				}
				uc := rune(0)
				for k := 0; k < 4; k++ {
					i++
					c = s[i]
					if isdigit(c) {
						uc = uc<<4 | rune(c-'0')
					} else if 'A' <= c && c <= 'F' {
						uc = uc<<4 | rune(c-'A'+10)
					} else if 'a' <= c && c <= 'f' {
						uc = uc<<4 | rune(c-'a'+10)
					} else {
						return nil, false
					}
				}
				var u8 [6]byte
				n := utf8.EncodeRune(u8[:], uc)
				dst = append(dst, u8[:n]...)
			} else {
				dst = append(dst, unescapeTable[c])
			}
		} else {
			dst = append(dst, c)
		}
		i++
	}
	return dst, true
}
