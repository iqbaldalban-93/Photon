package gphotos

import (
	"path/filepath"
	"regexp"
	"strconv"
	"time"
)

var filenameTimestampPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?:^|[^0-9])(20\d{2})[-_.]?(0[1-9]|1[0-2])[-_.]?(0[1-9]|[12]\d|3[01])[-_.T ]?([01]\d|2[0-3])?[:]?:?([0-5]\d)?[:]?:?([0-5]\d)?`),
	regexp.MustCompile(`(?:^|[^0-9])(20\d{2})(0[1-9]|1[0-2])(0[1-9]|[12]\d|3[01])[_-]?([01]\d|2[0-3])([0-5]\d)([0-5]\d)`),
}

func ParseTimestampFromFilename(path string) (t time.Time, ok bool) {
	name := filepath.Base(path)
	for _, re := range filenameTimestampPatterns {
		m := re.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		year, _ := strconv.Atoi(m[1])
		month, _ := strconv.Atoi(m[2])
		day, _ := strconv.Atoi(m[3])
		hour := 12
		minute, second := 0, 0
		if len(m) > 4 && m[4] != "" {
			hour, _ = strconv.Atoi(m[4])
		}
		if len(m) > 5 && m[5] != "" {
			minute, _ = strconv.Atoi(m[5])
		}
		if len(m) > 6 && m[6] != "" {
			second, _ = strconv.Atoi(m[6])
		}
		candidate := time.Date(year, time.Month(month), day, hour, minute, second, 0, time.Local)
		if candidate.After(time.Now().AddDate(0, 0, 1)) || candidate.Before(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)) {
			continue
		}
		return candidate, true
	}
	return time.Time{}, false
}
