package jsonpb

func getTestSimpleMessage() *Message {
	return NewMessage("Simple", []Field{
		{Name: "name", Tag: 1, Kind: StringKind, Omit: OmitEmpty},
		{Name: "age", Tag: 2, Kind: Int32Kind, Omit: OmitEmpty},
		{Name: "male", Tag: 3, Kind: BoolKind, Omit: OmitAlways},
	}, true, true)
}

func getTestSimpleMessage2() *Message {
	return NewMessage("Simple", []Field{
		{Name: "name", Tag: 1, Kind: StringKind, Omit: OmitAlways},
		{Name: "age", Tag: 2, Kind: Int32Kind, Omit: OmitAlways},
		{Name: "male", Tag: 3, Kind: BoolKind},
	}, true, true)
}

func getTestMapEntry(keyKind Kind, valueKind Kind, valueRef *Message) *Message {
	return NewMessage("", []Field{
		0: {Tag: 1, Kind: keyKind},
		1: {Tag: 2, Kind: valueKind, Ref: valueRef},
	}, true, true)
}

func getTestComplexMessage() *Message {
	return NewMessage("Complex", []Field{
		{Name: "fdouble", Kind: DoubleKind, Tag: 1},
		{Name: "ffloat", Kind: FloatKind, Tag: 2},
		{Name: "fint32", Kind: Int32Kind, Tag: 3},
		{Name: "fint64", Kind: Int64Kind, Tag: 4},
		{Name: "fuint32", Kind: Uint32Kind, Tag: 5},
		{Name: "fuint64", Kind: Uint64Kind, Tag: 6},
		{Name: "fsint32", Kind: Sint32Kind, Tag: 7},
		{Name: "fsint64", Kind: Sint64Kind, Tag: 8},
		{Name: "ffixed32", Kind: Fixed32Kind, Tag: 9},
		{Name: "ffixed64", Kind: Fixed64Kind, Tag: 10},
		{Name: "fsfixed32", Kind: Sfixed32Kind, Tag: 11},
		{Name: "fsfixed64", Kind: Sfixed64Kind, Tag: 12},
		{Name: "fbool", Kind: BoolKind, Tag: 13},
		{Name: "fstring", Kind: StringKind, Tag: 14},
		{Name: "fbytes", Kind: BytesKind, Tag: 15},
		{Name: "fmap1", Kind: MapKind, Tag: 16, Ref: getTestMapEntry(StringKind, Int32Kind, nil)},
		{Name: "fmap2", Kind: MapKind, Tag: 17, Ref: getTestMapEntry(StringKind, MessageKind, getTestSimpleMessage())},
		{Name: "fsubmsg", Kind: MessageKind, Tag: 18, Ref: getTestSimpleMessage()},
		{Name: "fint32s", Kind: Int32Kind, Tag: 19, Repeated: true},
		{Name: "fitems", Kind: MessageKind, Tag: 20, Repeated: true, Ref: getTestSimpleMessage()},
	}, true, true)
}
