package jsonpb

import "sort"

type Kind uint8

const (
	DoubleKind Kind = iota
	FloatKind
	Int32Kind
	Int64Kind
	Uint32Kind
	Uint64Kind
	Sint32Kind
	Sint64Kind
	Fixed32Kind
	Fixed64Kind
	Sfixed32Kind
	Sfixed64Kind
	BoolKind
	StringKind
	BytesKind
	MapKind
	MessageKind
)

func IsNumericKind(k Kind) bool {
	return DoubleKind <= k && k <= Sfixed64Kind
}

type Message struct {
	Name   string
	Fields []Field

	tagIdx  []int
	nameIdx map[string]int
}

func NewMessage(name string, fields []Field, indexTag bool, indexName bool) *Message {
	msg := &Message{
		Name:   name,
		Fields: fields,
	}
	if indexTag {
		msg.BakeTagIndex()
	}
	if indexName {
		msg.BakeNameIndex()
	}
	return msg
}

func (m *Message) BakeTagIndex() {
	fields := m.Fields
	maxTag := uint32(0)
	for i := range fields {
		if fields[i].Tag > maxTag {
			maxTag = fields[i].Tag
		}
	}
	var tagIdx []int
	if int(maxTag) < len(fields)+len(fields)/4+3 {
		tagIdx = make([]int, maxTag+1)
		for i := range tagIdx {
			tagIdx[i] = -1
		}
		for i := range fields {
			tagIdx[fields[i].Tag] = i
		}
	} else {
		// sparse-index
		tagIdx = make([]int, len(fields))
		for i := range tagIdx {
			tagIdx[i] = i
		}
		sort.Slice(tagIdx, func(i, j int) bool {
			return fields[tagIdx[i]].Tag < fields[tagIdx[j]].Tag
		})
	}
	m.tagIdx = tagIdx
}

func (m *Message) FieldIndexByTag(tag uint32) int {
	if len(m.tagIdx) == len(m.Fields) {
		l, r := 0, len(m.tagIdx)-1
		for l <= r {
			mid := (l + r) / 2
			i := m.tagIdx[mid]
			x := m.Fields[i].Tag
			if x == tag {
				return i
			} else if x > tag {
				r = mid - 1
			} else {
				l = mid + 1
			}
		}
	} else if int(tag) < len(m.tagIdx) {
		idx := m.tagIdx[tag]
		if idx >= 0 {
			return idx
		}
	} else {
		for i := 0; i < len(m.Fields); i++ {
			if m.Fields[i].Tag == tag {
				return i
			}
		}
	}
	return -1
}

func (m *Message) FieldByTag(tag uint32) *Field {
	idx := m.FieldIndexByTag(tag)
	if idx >= 0 {
		return &m.Fields[idx]
	}
	return nil
}

func (m *Message) BakeNameIndex() {
	names := make(map[string]int, len(m.Fields))
	for i := range m.Fields {
		names[m.Fields[i].Name] = i
	}
	m.nameIdx = names
}

func (m *Message) FieldByName(name string) *Field {
	if m.nameIdx != nil {
		idx, ok := m.nameIdx[name]
		if ok {
			return &m.Fields[idx]
		}
	} else {
		for i := 0; i < len(m.Fields); i++ {
			if m.Fields[i].Name == name {
				return &m.Fields[i]
			}
		}
	}
	return nil
}

type Field struct {
	Name      string
	Kind      Kind
	Ref       *Message
	Tag       uint32
	Repeated  bool
	OmitEmpty bool
}
