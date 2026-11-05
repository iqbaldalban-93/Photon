package proxy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"sync"
	"syscall"
	"time"
)

var ErrBlockedHost = errors.New("blocked: URL host is not a public address")

func ValidateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("blocked: only http/https URLs are allowed")
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("blocked: URL has no host")
	}
	ips, err := LookupHost(host)
	if err != nil {
		return fmt.Errorf("blocked: cannot resolve %q: %w", host, err)
	}
	for _, ip := range ips {
		if !isPublicIP(ip) {
			return fmt.Errorf("%w (%s resolved to %s)", ErrBlockedHost, host, ip)
		}
	}
	return nil
}

var (
	dnsMu    sync.Mutex
	dnsCache = map[string][]netip.Addr{}
	dnsAt    = map[string]time.Time{}
)

const dnsCacheTTL = 5 * time.Minute

func LookupHost(host string) ([]netip.Addr, error) {
	if addr, err := netip.ParseAddr(host); err == nil {
		return []netip.Addr{addr.Unmap()}, nil
	}

	dnsMu.Lock()
	if at, ok := dnsAt[host]; ok && time.Since(at) < dnsCacheTTL {
		ips := dnsCache[host]
		dnsMu.Unlock()
		return ips, nil
	}
	dnsMu.Unlock()

	addrs, err := net.DefaultResolver.LookupIPAddr(context.Background(), host)
	if err != nil {
		return nil, err
	}
	ips := make([]netip.Addr, 0, len(addrs))
	for _, ia := range addrs {
		if a, ok := netip.AddrFromSlice(ia.IP); ok {
			ips = append(ips, a.Unmap())
		}
	}
	dnsMu.Lock()
	if len(dnsCache) > 500 {
		dnsCache = map[string][]netip.Addr{}
		dnsAt = map[string]time.Time{}
	}
	dnsCache[host] = ips
	dnsAt[host] = time.Now()
	dnsMu.Unlock()
	return ips, nil
}

func isPublicIP(ip netip.Addr) bool {
	if !ip.IsValid() || ip.IsUnspecified() {
		return false
	}
	if ip.Is4In6() {
		ip = ip.Unmap()
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
		return false
	}
	if ip.Is4() {
		return ipv4Public(ip)
	}
	if ip.Is6() {
		return ipv6Public(ip)
	}
	return false
}

func ipv4Public(ip netip.Addr) bool {
	b := ip.As4()
	switch {
	case b[0] == 0:
		return false
	case b[0] == 10:
		return false
	case b[0] == 100 && b[1] >= 64 && b[1] <= 127:
		return false
	case b[0] == 127:
		return false
	case b[0] == 169 && b[1] == 254:
		return false
	case b[0] == 172 && b[1] >= 16 && b[1] <= 31:
		return false
	case b[0] == 192 && b[1] == 0 && (b[2] == 0 || b[2] == 2):
		return false
	case b[0] == 192 && b[1] == 88 && b[2] == 99:
		return false
	case b[0] == 192 && b[1] == 168:
		return false
	case b[0] == 198 && (b[1] == 18 || b[1] == 19):
		return false
	case b[0] == 198 && b[1] == 51 && b[2] == 100:
		return false
	case b[0] == 203 && b[1] == 0 && b[2] == 113:
		return false
	case b[0] >= 224:
		return false
	}
	return true
}

func ipv6Public(ip netip.Addr) bool {
	b := ip.As16()
	if b[0] == 0x20 && b[1] == 0x01 && b[2] == 0x0d && b[3] == 0xb8 {
		return false
	}
	if b[0] == 0x01 && b[1] == 0x00 {
		return false
	}
	if b[0] == 0 && !(b[1] == 0x64 && b[2] == 0xff && b[3] == 0x9b) {
		return false
	}
	if b[0] == 0x00 && b[1] == 0x64 && b[2] == 0xff && b[3] == 0x9b {
		return ipv4Public(netip.AddrFrom4([4]byte{b[12], b[13], b[14], b[15]}))
	}
	if b[0] == 0x20 && b[1] == 0x02 {
		return ipv4Public(netip.AddrFrom4([4]byte{b[2], b[3], b[4], b[5]}))
	}
	if b[0] == 0x20 && b[1] == 0x01 && b[2] == 0x00 && b[3] == 0x00 {
		return false
	}
	return true
}

func NewMediaClient() *http.Client {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		Control: func(network, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			addr, err := netip.ParseAddr(host)
			if err != nil {
				return err
			}
			if !isPublicIP(addr) {
				return ErrBlockedHost
			}
			return nil
		},
	}
	transport := &http.Transport{
		Proxy:                 nil,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          64,
		MaxIdleConnsPerHost:   8,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   15 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &http.Client{
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("too many redirects")
			}
			if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
				return fmt.Errorf("blocked: redirect to non-http scheme")
			}
			return nil
		},
	}
}
