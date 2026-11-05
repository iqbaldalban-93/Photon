// Package credentials provides helpers for the Google-account credential
// query strings (Email=...&Token=...&lang=...) stored in the app config.
package credentials

import (
	"net/url"
	"strings"
)

// Values parses a credentials query string. ok is false when parsing fails.
func Values(cred string) (url.Values, bool) {
	params, err := url.ParseQuery(cred)
	return params, err == nil
}

// Email returns the Email field of a credentials query string, or "" when the
// string cannot be parsed or has no Email.
func Email(cred string) string {
	params, ok := Values(cred)
	if !ok {
		return ""
	}
	return params.Get("Email")
}

// Matches reports whether two account email values refer to the same account:
// percent-escapes are decoded where possible and comparison is case-insensitive.
func Matches(a, b string) bool {
	da, erra := url.QueryUnescape(a)
	db, errb := url.QueryUnescape(b)
	if erra != nil {
		da = a
	}
	if errb != nil {
		db = b
	}
	return strings.EqualFold(da, db)
}

// Select returns the first credentials string from list whose Email matches
// email, or "" when none does.
func Select(list []string, email string) string {
	for _, c := range list {
		if Matches(Email(c), email) {
			return c
		}
	}
	return ""
}
