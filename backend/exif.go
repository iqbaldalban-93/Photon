package backend

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

var exifHeaderPrefix = []byte("Exif\x00\x00")

func exifCaptureDate(path string) (time.Time, bool) {
	f, err := os.Open(path)
	if err != nil {
		return time.Time{}, false
	}
	defer func() { _ = f.Close() }()

	head := make([]byte, 12)
	if _, err := io.ReadFull(f, head); err != nil {
		return time.Time{}, false
	}

	switch {
	case head[0] == 0xFF && head[1] == 0xD8:
		blob, ok := readJPEGExifBlob(f)
		if !ok {
			return time.Time{}, false
		}
		return parseTIFFDates(bytes.NewReader(blob), int64(len(blob)))
	case (head[0] == 'I' && head[1] == 'I' && head[2] == 0x2A && head[3] == 0x00) ||
		(head[0] == 'M' && head[1] == 'M' && head[2] == 0x00 && head[3] == 0x2A):
		info, err := f.Stat()
		if err != nil {
			return time.Time{}, false
		}
		return parseTIFFDates(f, info.Size())
	}
	return time.Time{}, false
}

func readJPEGExifBlob(f *os.File) ([]byte, bool) {
	if _, err := f.Seek(2, io.SeekStart); err != nil {
		return nil, false
	}
	for range 128 {
		var marker [2]byte
		if _, err := io.ReadFull(f, marker[:]); err != nil || marker[0] != 0xFF {
			return nil, false
		}
		switch m := marker[1]; {
		case m == 0xD8:
			continue
		case m == 0x01 || (m >= 0xD0 && m <= 0xD7):
			continue
		case m == 0xD9 || m == 0xDA:
			return nil, false
		}
		var l [2]byte
		if _, err := io.ReadFull(f, l[:]); err != nil {
			return nil, false
		}
		segLen := int(binary.BigEndian.Uint16(l[:])) - 2
		if segLen <= 0 || segLen > 1<<20 {
			return nil, false
		}
		seg := make([]byte, segLen)
		if _, err := io.ReadFull(f, seg); err != nil {
			return nil, false
		}
		if marker[1] == 0xE1 && bytes.HasPrefix(seg, exifHeaderPrefix) {
			return seg[len(exifHeaderPrefix):], true
		}
	}
	return nil, false
}

func exifTypeSize(typ uint16) int64 {
	switch typ {
	case 1, 2, 6, 7:
		return 1
	case 3, 8:
		return 2
	case 4, 9:
		return 4
	case 5, 10:
		return 8
	}
	return 0
}

func parseTIFFDates(r io.ReaderAt, size int64) (time.Time, bool) {
	var h [8]byte
	if _, err := r.ReadAt(h[:], 0); err != nil {
		return time.Time{}, false
	}
	var bo binary.ByteOrder
	switch {
	case h[0] == 'I' && h[1] == 'I':
		bo = binary.LittleEndian
	case h[0] == 'M' && h[1] == 'M':
		bo = binary.BigEndian
	default:
		return time.Time{}, false
	}
	if bo.Uint16(h[2:4]) != 42 {
		return time.Time{}, false
	}

	const maxExifValueBytes = 64
	readIFD := func(off int64, wanted func(tag uint16) bool, visit func(tag, typ uint16, value []byte) bool) bool {
		if off <= 0 || off+2 > size {
			return false
		}
		var cnt [2]byte
		if _, err := r.ReadAt(cnt[:], off); err != nil {
			return false
		}
		n := int(bo.Uint16(cnt[:]))
		if n > 512 {
			return false
		}
		for i := range n {
			var e [12]byte
			eoff := off + 2 + int64(i)*12
			if eoff+12 > size {
				return false
			}
			if _, err := r.ReadAt(e[:], eoff); err != nil {
				return false
			}
			tag := bo.Uint16(e[0:2])
			if !wanted(tag) {
				continue
			}
			typ := bo.Uint16(e[2:4])
			count := bo.Uint32(e[4:8])
			typeSize := exifTypeSize(typ)
			if typeSize == 0 {
				continue
			}
			total := int64(count) * typeSize
			if total > maxExifValueBytes {
				continue
			}
			var value []byte
			if total <= 4 {
				value = e[8 : 8+total]
			} else {
				voff := int64(bo.Uint32(e[8:12]))
				if voff <= 0 || voff+total > size {
					continue
				}
				value = make([]byte, total)
				if _, err := r.ReadAt(value, voff); err != nil {
					continue
				}
			}
			if visit(tag, typ, value) {
				return true
			}
		}
		return false
	}

	var exifIFD int64
	readIFD(int64(bo.Uint32(h[4:8])), func(tag uint16) bool { return tag == 0x8769 }, func(tag, typ uint16, value []byte) bool {
		if tag == 0x8769 && typ == 4 && len(value) >= 4 {
			exifIFD = int64(bo.Uint32(value[:4]))
			return true
		}
		return false
	})

	var origVal, digiVal, origOff, digiOff []byte
	if exifIFD > 0 {
		readIFD(exifIFD, func(tag uint16) bool {
			return tag == 0x9003 || tag == 0x9004 || tag == 0x9011 || tag == 0x9012
		}, func(tag, typ uint16, value []byte) bool {
			if typ != 2 {
				return false
			}
			switch tag {
			case 0x9003:
				origVal = append([]byte(nil), value...)
			case 0x9004:
				digiVal = append([]byte(nil), value...)
			case 0x9011:
				origOff = append([]byte(nil), value...)
			case 0x9012:
				digiOff = append([]byte(nil), value...)
			}
			return false
		})
	}
	if t, ok := parseExifTimeOffset(origVal, origOff); ok {
		return t, true
	}
	if t, ok := parseExifTimeOffset(digiVal, digiOff); ok {
		return t, true
	}
	return time.Time{}, false
}

func parseExifTimeOffset(value, offset []byte) (time.Time, bool) {
	if len(value) == 0 {
		return time.Time{}, false
	}
	t, ok := parseExifTime(value)
	if !ok {
		return time.Time{}, false
	}
	if secs, ok := parseExifOffset(offset); ok {
		s := strings.TrimRight(string(value), "\x00 ")
		if tz, err := time.ParseInLocation("2006:01:02 15:04:05", s, time.FixedZone("", secs)); err == nil {
			return tz, true
		}
	}
	return t, true
}

func parseExifOffset(b []byte) (int, bool) {
	s := strings.TrimRight(string(b), "\x00 ")
	if len(s) != 6 || (s[0] != '+' && s[0] != '-') || s[3] != ':' {
		return 0, false
	}
	hh, err1 := strconv.Atoi(s[1:3])
	mm, err2 := strconv.Atoi(s[4:6])
	if err1 != nil || err2 != nil || hh > 14 || mm > 59 {
		return 0, false
	}
	secs := hh*3600 + mm*60
	if s[0] == '-' {
		secs = -secs
	}
	return secs, true
}

func parseExifTime(value []byte) (time.Time, bool) {
	s := strings.TrimRight(string(value), "\x00 ")
	t, err := time.ParseInLocation("2006:01:02 15:04:05", s, time.Local)
	if err != nil {
		return time.Time{}, false
	}
	if t.Year() < 1900 || t.Year() > time.Now().Year()+2 {
		return time.Time{}, false
	}
	return t, true
}
