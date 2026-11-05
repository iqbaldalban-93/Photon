package backend

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

func contentMatchesExtension(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".ts" && ext != ".mod" && ext != ".tod" {
		return true
	}
	f, err := os.Open(path)
	if err != nil {
		return true
	}
	defer f.Close()
	buf := make([]byte, 197)
	n, _ := io.ReadFull(f, buf)
	buf = buf[:n]
	switch ext {
	case ".ts":
		if n >= 1 && buf[0] == 0x47 && (n < 189 || buf[188] == 0x47) {
			return true
		}
		return n >= 197 && buf[4] == 0x47 && buf[196] == 0x47
	default:
		return n >= 4 && buf[0] == 0 && buf[1] == 0 && buf[2] == 1
	}
}
