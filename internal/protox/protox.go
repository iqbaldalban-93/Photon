package protox

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
)

// Minimal protobuf wire encode/decode helpers (no protoc dependency).

type Field struct {
	Num  int
	Wire int
	Val  uint64
	Bts  []byte
}

type Builder struct{ fields []Field }

func NewBuilder() *Builder { return &Builder{} }

func (b *Builder) Varint(num int, v uint64) *Builder {
	b.fields = append(b.fields, Field{Num: num, Wire: 0, Val: v})
	return b
}

func (b *Builder) Int(num int, v int64) *Builder {
	return b.Varint(num, uint64(v))
}

func (b *Builder) Bool(num int, v bool) *Builder {
	if v {
		return b.Varint(num, 1)
	}
	return b.Varint(num, 0)
}

func (b *Builder) Bytes(num int, v []byte) *Builder {
	b.fields = append(b.fields, Field{Num: num, Wire: 2, Bts: v})
	return b
}

func (b *Builder) Str(num int, v string) *Builder {
	return b.Bytes(num, []byte(v))
}

func (b *Builder) Msg(num int, fn func(*Builder)) *Builder {
	sub := NewBuilder()
	fn(sub)
	return b.Bytes(num, sub.Bytes_())
}

func (b *Builder) Each(num int, items []any, fn func(*Builder, any)) *Builder {
	for _, it := range items {
		b.Msg(num, func(sub *Builder) { fn(sub, it) })
	}
	return b
}

func (b *Builder) Bytes_() []byte {
	sort.SliceStable(b.fields, func(i, j int) bool { return b.fields[i].Num < b.fields[j].Num })
	out := make([]byte, 0, 64)
	for _, f := range b.fields {
		out = appendTag(out, f.Num, f.Wire)
		switch f.Wire {
		case 0:
			out = appendVarint(out, f.Val)
		case 2:
			out = appendVarint(out, uint64(len(f.Bts)))
			out = append(out, f.Bts...)
		}
	}
	return out
}

func appendVarint(out []byte, v uint64) []byte {
	for v >= 0x80 {
		out = append(out, byte(v)|0x80)
		v >>= 7
	}
	return append(out, byte(v))
}

func appendTag(out []byte, num, wire int) []byte {
	return appendVarint(out, uint64(num)<<3|uint64(wire))
}

type Val struct {
	Varint uint64
	Bytes  []byte
	Wire   int
}

type Msg map[int][]Val

var ErrTruncated = errors.New("protox: truncated message")

func Parse(buf []byte) (Msg, error) {
	m := Msg{}
	for len(buf) > 0 {
		tag, n := binary.Uvarint(buf)
		if n <= 0 {
			return nil, ErrTruncated
		}
		buf = buf[n:]
		num, wire := int(tag>>3), int(tag&7)
		switch wire {
		case 0:
			v, n := binary.Uvarint(buf)
			if n <= 0 {
				return nil, ErrTruncated
			}
			buf = buf[n:]
			m[num] = append(m[num], Val{Varint: v, Wire: 0})
		case 1:
			if len(buf) < 8 {
				return nil, ErrTruncated
			}
			v := binary.LittleEndian.Uint64(buf)
			buf = buf[8:]
			m[num] = append(m[num], Val{Varint: v, Wire: 1})
		case 2:
			l, n := binary.Uvarint(buf)
			if n <= 0 || l > uint64(len(buf[n:])) {
				return nil, ErrTruncated
			}
			buf = buf[n:]
			m[num] = append(m[num], Val{Bytes: buf[:l], Wire: 2})
			buf = buf[l:]
		case 5:
			if len(buf) < 4 {
				return nil, ErrTruncated
			}
			v := binary.LittleEndian.Uint32(buf)
			buf = buf[4:]
			m[num] = append(m[num], Val{Varint: uint64(v), Wire: 5})
		default:
			return nil, fmt.Errorf("protox: unsupported wire type %d (field %d)", wire, num)
		}
	}
	return m, nil
}

func (m Msg) First(num int) (Val, bool) {
	vals := m[num]
	if len(vals) == 0 {
		return Val{}, false
	}
	return vals[0], true
}

func (m Msg) Uint(num int) uint64 {
	if v, ok := m.First(num); ok && v.Wire == 0 {
		return v.Varint
	}
	return 0
}

func (m Msg) Str(num int) string {
	if v, ok := m.First(num); ok && v.Wire == 2 {
		return string(v.Bytes)
	}
	return ""
}

func (m Msg) Raw(num int) []byte {
	if v, ok := m.First(num); ok && v.Wire == 2 {
		return v.Bytes
	}
	return nil
}

func (m Msg) Sub(num int) Msg {
	v, ok := m.First(num)
	if !ok || v.Wire != 2 {
		return nil
	}
	sub, err := Parse(v.Bytes)
	if err != nil {
		return nil
	}
	return sub
}

func (m Msg) Msgs(num int) []Msg {
	vals := m[num]
	out := make([]Msg, 0, len(vals))
	for _, v := range vals {
		if v.Wire != 2 {
			continue
		}
		if sub, err := Parse(v.Bytes); err == nil {
			out = append(out, sub)
		}
	}
	return out
}

func Fixed32ToGeo(v uint64) float64 {
	n := int32(uint32(v))
	return float64(n) / 1e7
}
