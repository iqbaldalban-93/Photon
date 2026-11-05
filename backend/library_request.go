package backend

import "google.golang.org/protobuf/encoding/protowire"

func pbTag(field int, wire protowire.Type) []byte {
	return protowire.AppendTag(nil, protowire.Number(field), wire)
}

func pbVarint(field int, v uint64) []byte {
	out := pbTag(field, protowire.VarintType)
	return protowire.AppendVarint(out, v)
}

func pbBlob(field int, data []byte) []byte {
	out := pbTag(field, protowire.BytesType)
	return protowire.AppendBytes(out, data)
}

func pbMsg(field int, parts ...[]byte) []byte {
	return pbBlob(field, pbConcat(parts...))
}

func pbConcat(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

func pbRepeatedVarints(field int, vals ...uint64) []byte {
	var out []byte
	for _, v := range vals {
		out = append(out, pbVarint(field, v)...)
	}
	return out
}

func buildLibraryStateRequest() []byte {
	entityDecls := pbMsg(1,
		pbBlob(1, nil), pbBlob(3, nil), pbBlob(4, nil),
		pbMsg(5, pbBlob(1, nil), pbBlob(2, nil), pbBlob(3, nil), pbBlob(4, nil), pbBlob(5, nil), pbBlob(7, nil)),
		pbBlob(6, nil),
		pbMsg(7, pbBlob(2, nil)),
		pbBlob(15, nil), pbBlob(16, nil), pbBlob(17, nil), pbBlob(19, nil), pbBlob(20, nil),
		pbMsg(21, pbMsg(5, pbBlob(3, nil)), pbBlob(6, nil)),
		pbBlob(25, nil),
		pbMsg(30, pbBlob(2, nil)),
		pbBlob(31, nil), pbBlob(32, nil),
		pbMsg(33, pbBlob(1, nil)),
		pbBlob(34, nil), pbBlob(36, nil), pbBlob(37, nil), pbBlob(38, nil), pbBlob(39, nil), pbBlob(40, nil), pbBlob(41, nil),
	)

	typeRanges := pbMsg(5,
		pbMsg(2,
			pbMsg(2,
				pbMsg(3, pbBlob(2, nil)),
				pbMsg(4, pbBlob(2, nil), pbBlob(4, nil)),
			),
			pbMsg(4, pbMsg(2, pbVarint(2, 1))),
			pbMsg(5, pbBlob(2, nil)),
			pbVarint(6, 1),
		),
		pbMsg(3,
			pbMsg(2, pbBlob(3, nil), pbBlob(4, nil)),
			pbMsg(3, pbBlob(2, nil), pbMsg(3, pbVarint(2, 1), pbBlob(3, nil))),
			pbBlob(4, nil),
			pbMsg(5, pbMsg(2, pbVarint(2, 1))),
			pbBlob(7, nil),
		),
		pbMsg(4, pbMsg(2, pbMsg(2, nil))),
		pbMsg(5,
			pbMsg(1,
				pbMsg(2,
					pbBlob(3, nil), pbBlob(4, nil),
				),
				pbMsg(3, pbBlob(2, nil), pbMsg(3, pbVarint(2, 1), pbBlob(3, nil))),
			),
			pbVarint(3, 1),
		),
	)

	people := pbMsg(9,
		pbBlob(2, nil),
		pbMsg(3, pbBlob(1, nil), pbBlob(2, nil)),
		pbMsg(4,
			pbMsg(1,
				pbMsg(3,
					pbMsg(1,
						pbMsg(1, pbMsg(5, pbBlob(1, nil)), pbBlob(6, nil), pbBlob(7, nil)),
						pbBlob(2, nil),
						pbMsg(3,
							pbMsg(1, pbMsg(5, pbBlob(1, nil)), pbBlob(6, nil), pbBlob(7, nil)),
							pbBlob(2, nil),
						),
					),
				),
				pbMsg(4, pbMsg(1, pbBlob(2, nil))),
			),
		),
	)

	libraryDecl := pbMsg(1,
		entityDecls,
		typeRanges,
		pbBlob(8, nil),
		people,
		pbMsg(11, pbBlob(2, nil), pbBlob(3, nil), pbMsg(4, pbMsg(2, pbVarint(1, 1), pbVarint(2, 2)))),
		pbBlob(12, nil),
		pbMsg(14, pbBlob(2, nil), pbBlob(3, nil), pbMsg(4, pbMsg(2, pbVarint(1, 1), pbVarint(2, 2)))),
		pbMsg(15, pbBlob(1, nil), pbBlob(4, nil)),
		pbMsg(17, pbBlob(1, nil), pbBlob(4, nil)),
		pbMsg(19, pbBlob(2, nil), pbBlob(3, nil), pbMsg(4, pbMsg(2, pbVarint(1, 1), pbVarint(2, 2)))),
		pbMsg(21, pbBlob(1, nil)),
		pbBlob(22, nil), pbBlob(23, nil), pbBlob(24, nil),
	)

	mediaDecl := pbMsg(2,
		pbMsg(1,
			pbBlob(2, nil), pbBlob(3, nil), pbBlob(4, nil), pbBlob(5, nil),
			pbMsg(6, pbBlob(1, nil), pbBlob(2, nil), pbBlob(3, nil), pbBlob(4, nil), pbBlob(5, nil), pbBlob(7, nil)),
			pbBlob(7, nil), pbBlob(8, nil), pbBlob(10, nil), pbBlob(12, nil),
			pbMsg(13, pbBlob(2, nil), pbBlob(3, nil)),
			pbMsg(15, pbBlob(1, nil)),
			pbBlob(18, nil),
		),
		pbMsg(4, pbBlob(1, nil)),
		pbBlob(9, nil),
		pbMsg(11, pbMsg(1, pbBlob(1, nil), pbBlob(4, nil), pbBlob(5, nil), pbBlob(6, nil), pbBlob(9, nil))),
		pbMsg(14,
			pbMsg(1,
				pbMsg(1,
					pbBlob(1, nil),
					pbMsg(2, pbMsg(2, pbMsg(1, pbMsg(1, nil)), pbBlob(3, nil))),
					pbMsg(3,
						pbMsg(4, pbMsg(1, pbBlob(1, nil)), pbBlob(3, nil)),
						pbMsg(5, pbMsg(1, pbBlob(1, nil)), pbBlob(3, nil)),
					),
				),
				pbBlob(2, nil),
			),
		),
		pbBlob(17, nil),
		pbMsg(18, pbBlob(1, nil), pbMsg(2, pbBlob(1, nil))),
		pbMsg(20, pbMsg(2, pbBlob(1, nil), pbBlob(2, nil))),
		pbBlob(22, nil), pbBlob(23, nil), pbBlob(24, nil),
	)

	albumDecl := pbMsg(3,
		pbBlob(2, nil),
		pbMsg(3,
			pbBlob(2, nil), pbBlob(3, nil), pbBlob(7, nil), pbBlob(8, nil),
			pbMsg(14, pbBlob(1, nil)),
			pbBlob(16, nil),
			pbMsg(17, pbBlob(2, nil)),
			pbBlob(18, nil), pbBlob(19, nil), pbBlob(20, nil), pbBlob(21, nil), pbBlob(22, nil), pbBlob(23, nil),
			pbMsg(27, pbBlob(1, nil), pbMsg(2, pbBlob(1, nil))),
			pbBlob(29, nil), pbBlob(30, nil), pbBlob(31, nil), pbBlob(32, nil), pbBlob(34, nil),
			pbBlob(37, nil), pbBlob(38, nil), pbBlob(39, nil), pbBlob(41, nil),
			pbMsg(43, pbBlob(1, nil)),
			pbMsg(45, pbMsg(1, pbBlob(1, nil))),
			pbMsg(46, pbBlob(1, nil), pbBlob(2, nil), pbBlob(3, nil)),
			pbBlob(47, nil),
		),
		pbMsg(4, pbBlob(2, nil), pbMsg(3, pbBlob(1, nil)), pbBlob(4, nil), pbMsg(5, pbBlob(1, nil))),
		pbBlob(7, nil), pbBlob(12, nil), pbBlob(13, nil),
		pbMsg(14,
			pbBlob(1, nil),
			pbMsg(2, pbBlob(1, nil), pbMsg(2, pbBlob(1, nil)), pbBlob(3, nil), pbMsg(4, pbBlob(1, nil))),
			pbMsg(3, pbBlob(1, nil), pbMsg(2, pbBlob(1, nil)), pbBlob(3, nil), pbBlob(4, nil)),
		),
		pbBlob(15, nil),
		pbMsg(16, pbBlob(1, nil)),
		pbBlob(18, nil),
		pbMsg(19, pbMsg(4, pbBlob(2, nil)), pbMsg(6, pbBlob(2, nil), pbBlob(3, nil)), pbMsg(7, pbBlob(2, nil), pbBlob(3, nil)), pbBlob(8, nil), pbBlob(9, nil)),
		pbBlob(20, nil), pbBlob(22, nil), pbBlob(24, nil), pbBlob(25, nil), pbBlob(26, nil),
	)

	syncOptions := pbMsg(9,
		pbMsg(1, pbMsg(2, pbBlob(1, nil), pbBlob(2, nil))),
		pbMsg(2, pbMsg(3, pbVarint(2, 1))),
		pbMsg(3, pbBlob(2, nil)),
		pbBlob(4, nil),
		pbMsg(7, pbBlob(1, nil)),
		pbMsg(8, pbVarint(1, 2), pbBlob(2, []byte("\x01\x02\x03\x05\x06\x07"))),
		pbBlob(9, nil),
		pbMsg(11, pbBlob(1, nil)),
	)

	var top1 []byte
	top1 = append(top1, libraryDecl...)
	top1 = append(top1, mediaDecl...)
	top1 = append(top1, albumDecl...)
	top1 = append(top1, pbBlob(6, []byte(""))...)
	top1 = append(top1, pbVarint(7, 2)...)
	top1 = append(top1, syncOptions...)
	top1 = append(top1, pbRepeatedVarints(11, 1, 2, 6)...)
	top1 = append(top1, pbMsg(12,
		pbMsg(2, pbBlob(1, nil), pbBlob(2, nil)),
		pbMsg(3, pbBlob(1, nil)),
		pbBlob(4, nil),
	)...)
	top1 = append(top1, pbBlob(13, nil)...)
	top1 = append(top1, pbMsg(15, pbMsg(3, pbVarint(1, 1)))...)
	top1 = append(top1, pbMsg(18,
		pbBlob(169945741,
			pbMsg(1,
				pbMsg(1,
					pbRepeatedVarints(4, 2, 1, 6, 8, 10, 15, 18, 13, 17, 19, 14, 20),
					pbVarint(5, 6), pbVarint(6, 2), pbVarint(7, 1), pbVarint(8, 2),
					pbVarint(11, 3), pbVarint(12, 1), pbVarint(13, 3), pbVarint(15, 1),
					pbVarint(16, 1), pbVarint(17, 1), pbVarint(18, 2),
				),
			),
		),
	)...)
	top1 = append(top1, pbMsg(19,
		pbMsg(1, pbBlob(1, nil), pbBlob(2, nil)),
		pbMsg(2, pbRepeatedVarints(1, 1, 2, 4, 6, 5, 7)),
		pbMsg(3, pbBlob(1, nil), pbBlob(2, nil)),
		pbMsg(5, pbBlob(1, nil), pbBlob(2, nil)),
		pbMsg(6, pbBlob(1, nil)),
		pbMsg(7, pbBlob(1, nil), pbBlob(2, nil)),
		pbMsg(8, pbBlob(1, nil)),
	)...)
	top1 = append(top1, pbMsg(20,
		pbVarint(1, 1),
		pbBlob(2, []byte("")),
		pbMsg(3,
			pbBlob(1, []byte("type.googleapis.com/photos.printing.client.PrintingPromotionSyncOptions")),
			pbMsg(2,
				pbMsg(1,
					pbRepeatedVarints(4, 2, 1, 6, 8, 10, 15, 18, 13, 17, 19, 14, 20),
					pbVarint(5, 6), pbVarint(6, 2), pbVarint(7, 1), pbVarint(8, 2),
					pbVarint(11, 3), pbVarint(12, 1), pbVarint(13, 3), pbVarint(15, 1),
					pbVarint(16, 1), pbVarint(17, 1), pbVarint(18, 2),
				),
			),
		),
	)...)
	top1 = append(top1, pbMsg(21,
		pbMsg(2, pbMsg(2, pbBlob(4, nil)), pbBlob(4, nil), pbBlob(5, nil)),
		pbMsg(3, pbMsg(2, pbVarint(1, 1)), pbMsg(4, pbBlob(2, nil))),
		pbMsg(5, pbBlob(1, nil)),
		pbMsg(6, pbBlob(1, nil), pbMsg(2, pbBlob(1, nil))),
		pbMsg(7,
			pbVarint(1, 2),
			pbBlob(2, []byte("\x01\x07\x08\t\n\r\x0e\x0f\x11\x13\x14\x16\x17-./01:\x06\x18267;>?@A89<GBED")),
			pbBlob(3, []byte("\x01")),
		),
		pbMsg(8,
			pbMsg(3,
				pbMsg(1,
					pbMsg(1,
						pbMsg(2, pbVarint(1, 1)),
						pbMsg(4, pbBlob(2, nil)),
					),
				),
				pbBlob(3, nil),
			),
			pbMsg(4, pbBlob(1, nil)),
			pbMsg(5,
				pbMsg(1,
					pbMsg(2, pbVarint(1, 1)),
					pbMsg(4, pbBlob(2, nil)),
				),
			),
		),
		pbMsg(9, pbBlob(1, nil)),
		pbMsg(10, pbMsg(1, pbBlob(1, nil)), pbBlob(3, nil), pbBlob(5, nil), pbMsg(6, pbBlob(1, nil)), pbBlob(7, nil), pbBlob(9, nil), pbBlob(10, nil)),
		pbBlob(11, nil), pbBlob(12, nil), pbBlob(13, nil), pbBlob(14, nil),
		pbMsg(16, pbBlob(1, nil)),
	)...)
	top1 = append(top1, pbMsg(22, pbVarint(1, 1), pbBlob(2, []byte("107818234414673686888")))...)
	top1 = append(top1, pbMsg(25, pbMsg(1, pbMsg(1, pbMsg(1, pbMsg(1, nil)))), pbBlob(2, nil))...)
	top1 = append(top1, pbBlob(26, nil)...)

	top2 := pbMsg(2,
		pbMsg(1,
			pbMsg(1,
				pbMsg(1, pbMsg(1, nil)),
				pbBlob(2, nil),
			),
		),
		pbBlob(2, nil),
	)

	return pbConcat(
		pbMsg(1, top1),
		top2,
	)
}
