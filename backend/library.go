package backend

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

const libraryStateEndpoint = "https://photosdata-pa.googleapis.com/6439526531001121323/18047484249733410717"

type AlbumInfo struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	ItemCount int64  `json:"itemCount"`
}

func parseAlbumsFromLibraryState(data []byte) ([]AlbumInfo, error) {
	libState, err := firstMessageField(data, 1)
	if err != nil {
		return nil, err
	}

	var albums []AlbumInfo
	for _, entry := range messageFields(libState, 3) {
		var info AlbumInfo
		for _, f := range iterateFields(entry) {
			switch f.num {
			case 1:
				info.Key = string(f.data)
			case 2:
				for _, g := range iterateFields(f.data) {
					switch g.num {
					case 5:
						info.Name = string(g.data)
					case 7:
						if v, n := protowire.ConsumeVarint(g.data); n >= 0 {
							info.ItemCount = int64(v)
						}
					}
				}
			}
		}
		if info.Key != "" {
			albums = append(albums, info)
		}
	}
	return albums, nil
}

type rawField struct {
	num  protowire.Number
	typ  protowire.Type
	data []byte
}

func iterateFields(data []byte) []rawField {
	var fields []rawField
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fields
		}
		data = data[n:]
		switch typ {
		case protowire.VarintType:
			_, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fields
			}
			fields = append(fields, rawField{num: num, typ: typ, data: data[:n]})
			data = data[n:]
		case protowire.BytesType:
			payload, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fields
			}
			fields = append(fields, rawField{num: num, typ: typ, data: payload})
			data = data[n:]
		default:
			n := protowire.ConsumeFieldValue(num, typ, data)
			if n < 0 {
				return fields
			}
			data = data[n:]
		}
	}
	return fields
}

func firstMessageField(data []byte, num protowire.Number) ([]byte, error) {
	for _, f := range iterateFields(data) {
		if f.num == num && f.typ == protowire.BytesType {
			return f.data, nil
		}
	}
	return nil, fmt.Errorf("field %d not found", num)
}

func messageFields(data []byte, num protowire.Number) [][]byte {
	var out [][]byte
	for _, f := range iterateFields(data) {
		if f.num == num && f.typ == protowire.BytesType {
			out = append(out, f.data)
		}
	}
	return out
}
