package jsonlit

type Bytes interface {
	~string | []byte
}

type Kind int

const (
	Invalid Kind = iota
	Null
	Bool
	Number
	String
	Opener
	Object
	ObjectClose
	Array
	ArrayClose
	Comma
	Colon
	EOF
)

type Iter[S Bytes] struct {
	s S
	p int
}

func NewIter[S Bytes](s S) *Iter[S] {
	return &Iter[S]{
		s: s,
	}
}

func (it *Iter[S]) Reset(data S) {
	it.s = data
	it.p = 0
}

func (it *Iter[S]) EOF() bool {
	return it.p >= len(it.s)
}

func (it *Iter[S]) nextString() (Kind, S) {
	b := it.p
	p := it.p + 1
	for p < len(it.s) {
		if it.s[p] == '"' && it.s[p-1] != '\\' {
			it.p = p + 1
			return String, it.s[b:it.p]
		}
		p++
	}
	it.p = p
	return Invalid, it.s[b:]
}

func (it *Iter[S]) nextNumber() (Kind, S) {
	b := it.p
	p := it.p + 1
	for p < len(it.s) {
		c := it.s[p]
		if !isdigit(c) && c != '.' && c != '-' && c != 'e' && c != 'E' {
			break
		}
		p++
	}
	it.p = p
	return Number, it.s[b:p]
}

func (it *Iter[S]) consume(kind Kind) (Kind, S) {
	p := it.p
	it.p++
	return kind, it.s[p : p+1]
}

func (it *Iter[S]) expect(expected string, kind Kind) (Kind, S) {
	p := it.p
	e := p + len(expected)
	if e > len(it.s) {
		e = len(it.s)
		kind = Invalid
	} else if string(it.s[p:e]) != expected {
		// 如果 S 是 string，那么 `string(it.s[p:e])` 没有任何开销，
		// 如果 S 是 []byte，根据已知 go 语言优化，也不会产生开销。
		// See: https://www.go101.org/article/string.html#conversion-optimizations
		kind = Invalid
	}
	it.p = e
	return kind, it.s[p:e]
}

func (it *Iter[S]) Next() (Kind, S) {
	p := it.p
	for p < len(it.s) && iswhitespace(it.s[p]) {
		p++
	}
	if p >= len(it.s) {
		return EOF, it.s[len(it.s):]
	}
	it.p = p

	c := it.s[p]
	switch c {
	case 'n':
		return it.expect("null", Null)
	case 't':
		return it.expect("true", Bool)
	case 'f':
		return it.expect("false", Bool)
	case '"':
		return it.nextString()
	case '{':
		return it.consume(Object)
	case '}':
		return it.consume(ObjectClose)
	case '[':
		return it.consume(Array)
	case ']':
		return it.consume(ArrayClose)
	case ',':
		return it.consume(Comma)
	case ':':
		return it.consume(Colon)
	default:
		if isdigit(c) || c == '-' {
			return it.nextNumber()
		}
	}
	return it.consume(Invalid)
}

func iswhitespace(c byte) bool {
	return c == ' ' || c == '\n' || c == '\r' || c == '\t'
}

func isdigit(c byte) bool {
	return '0' <= c && c <= '9'
}
