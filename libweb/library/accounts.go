package library

import (
	"crypto/sha1"
	"encoding/hex"
	"net/url"
	"path/filepath"
	"strings"
)

const NoAccountDir = "_none"

func AccountKey(email string) string {
	email = strings.TrimSpace(email)
	if strings.Contains(email, "%") {
		if dec, err := url.PathUnescape(email); err == nil {
			email = dec
		}
	}

	email = strings.ReplaceAll(email, " ", "+")
	return strings.ToLower(email)
}

func AccountDir(dataDir, email string) string {
	key := AccountKey(email)
	if key == "" {
		return filepath.Join(dataDir, "accounts", NoAccountDir)
	}
	var b strings.Builder
	for _, r := range strings.ReplaceAll(key, "@", "_at_") {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '.', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	slug := b.String()
	if len(slug) > 48 {
		slug = slug[:48]
	}
	sum := sha1.Sum([]byte(key))
	return filepath.Join(dataDir, "accounts", slug+"-"+hex.EncodeToString(sum[:3]))
}
