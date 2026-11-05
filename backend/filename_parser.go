package backend

import (
	"path/filepath"
	"regexp"
	"strconv"
	"time"
)

type filenamePattern struct {
	re       *regexp.Regexp
	hasTime  bool
	isUnixMs bool
}

var filenameTimestampPatterns = []filenamePattern{
	{regexp.MustCompile(`(\d{4})(\d{2})(\d{2})[_-](\d{2})(\d{2})(\d{2})\d*`), true, false},
	{regexp.MustCompile(`(\d{4})-(\d{1,2})-(\d{1,2})(?:[ _-](\d{2})(\d{2})(\d{2})\d*)?`), false, false},
	{regexp.MustCompile(`(?:^|[^0-9])(\d{4})(\d{2})(\d{2})(\d{2})(\d{2})(\d{2})(?:[^0-9]|$)`), true, false},
	{regexp.MustCompile(`(?:^|[^0-9])(1\d{12})(?:[^0-9]|$)`), true, true},
}

func parseTimestampFromFilename(filename string) (time.Time, bool) {
	base := filepath.Base(filename)

	for _, pat := range filenameTimestampPatterns {
		m := pat.re.FindStringSubmatch(base)
		if m == nil {
			continue
		}

		if pat.isUnixMs {
			ms, err := strconv.ParseInt(m[1], 10, 64)
			if err != nil {
				continue
			}
			t := time.UnixMilli(ms).In(time.Local)
			if t.Year() < 1990 || t.Year() > time.Now().Year()+1 {
				continue
			}
			return t, true
		}

		year, month, day := m[1], m[2], m[3]
		hour, minute, sec := "12", "00", "00"

		if pat.hasTime {
			hour, minute, sec = m[4], m[5], m[6]
		} else if len(m) >= 7 && m[4] != "" {
			hour, minute, sec = m[4], m[5], m[6]
		}

		pad2 := func(s string) string {
			if len(s) < 2 {
				return "0" + s
			}
			return s
		}

		t, err := time.ParseInLocation("20060102 150405",
			year+pad2(month)+pad2(day)+" "+pad2(hour)+minute+sec, time.Local)
		if err != nil {
			continue
		}
		if t.Year() < 1990 || t.Year() > time.Now().Year()+1 {
			continue
		}
		return t, true
	}

	return time.Time{}, false
}
