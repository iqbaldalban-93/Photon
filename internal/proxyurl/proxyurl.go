package proxyurl

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

func Parse(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy URL: %w", err)
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https", "socks5":
	default:
		return nil, fmt.Errorf("proxy URL must use http, https, or socks5")
	}
	if u.Hostname() == "" || u.Opaque != "" || u.Fragment != "" {
		return nil, fmt.Errorf("proxy URL must include a hostname and cannot contain a fragment")
	}
	if port := u.Port(); port != "" {
		p, err := strconv.Atoi(port)
		if err != nil || p < 1 || p > 65535 {
			return nil, fmt.Errorf("proxy URL has an invalid port")
		}
	}
	return u, nil
}
