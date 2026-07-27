package jsonlit

import (
	"unicode/utf8"
)

const (
	rawMark       = '0'
	escapeTable   = "00000000btn0fr00000000000000000000\"000000000000/00000000000000000000000000000000000000000000\\"
	unescapeTable = "0000000000000000000000000000000000\"000000000000/00000000000000000000000000000000000000000000\\00000\x08000\x0C0000000\n000\r0\tu"
)

// hexVal 把一个十六进制字符转换为数值，非法字符返回 ok=false。
func hexVal(c byte) (rune, bool) {
	switch {
	case '0' <= c && c <= '9':
		return rune(c - '0'), true
	case 'a' <= c && c <= 'f':
		return rune(c - 'a' + 10), true
	case 'A' <= c && c <= 'F':
		return rune(c - 'A' + 10), true
	}
	return 0, false
}

const hexDigits = "0123456789abcdef"

func EscapeString[S Bytes](dst []byte, s S) []byte {
	begin := 0
	i := 0
	for i < len(s) {
		c := s[i]
		if int(c) < len(escapeTable) && escapeTable[c] != rawMark {
			if begin < i {
				dst = append(dst, s[begin:i]...)
			}
			dst = append(dst, '\\', escapeTable[c])
			i++
			begin = i
		} else if c < 0x20 {
			// 其它控制字符按 \uXXXX 转义，避免产出非法 JSON
			if begin < i {
				dst = append(dst, s[begin:i]...)
			}
			dst = append(dst, '\\', 'u', '0', '0', hexDigits[c>>4], hexDigits[c&0xf])
			i++
			begin = i
		} else {
			i++
		}
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
					d, ok := hexVal(c)
					if !ok {
						return nil, false
					}
					uc = uc<<4 | d
				}
				// 处理代理对：高位代理后紧跟低位代理时合并为补充平面字符
				if uc >= 0xD800 && uc <= 0xDBFF {
					if i+6 < len(s) && s[i+1] == '\\' && s[i+2] == 'u' {
						lo := rune(0)
						ok := true
						for k := 0; k < 4; k++ {
							d, dok := hexVal(s[i+3+k])
							if !dok {
								ok = false
								break
							}
							lo = lo<<4 | d
						}
						if ok && lo >= 0xDC00 && lo <= 0xDFFF {
							uc = 0x10000 + ((uc - 0xD800) << 10) + (lo - 0xDC00)
							i += 6
						}
						// 不是合法低位代理则保留 uc 为高位代理，
						// 由 EncodeRune 编码为替换字符。
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
