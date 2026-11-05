package server

import (
	"net"
	"net/http"
	"net/url"
	"strings"

	"app/internal/credentials"
	"app/libweb/authx"
	"app/libweb/configx"
)

func (s *Server) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	cfg := configx.Get()
	jsonWrite(w, http.StatusOK, map[string]any{
		"setupNeeded": authx.NeedsSetup(),
		"loggedIn":    authx.Authorized(r),
		"username":    cfg.Auth.Username,
		"throttled":   authx.LoginThrottled(clientKey(r)),
	})
}

var SetupLocalOnly bool

func redactPhotos(p configx.Photos) configx.Photos {
	p.Credentials = nil
	return p
}

func isLocalHostName(hostport string) bool {
	host := hostport
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	return net.ParseIP(host) != nil
}

func clientKey(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func isLoopbackRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (s *Server) handleAuthSetup(w http.ResponseWriter, r *http.Request) {
	if !authx.NeedsSetup() {
		jsonError(w, http.StatusForbidden, "already configured")
		return
	}
	if SetupLocalOnly && !isLocalHostName(r.Host) {
		jsonError(w, http.StatusForbidden, "first-time setup must be opened as http://localhost on the computer running Photon")
		return
	}
	if SetupLocalOnly && !isLoopbackRequest(r) {
		jsonError(w, http.StatusForbidden, "first-time setup must be done in a browser on the computer running Photon (open http://localhost there)")
		return
	}
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, err)
		return
	}
	if err := authx.SetupUser(body.Username, body.Password); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	cookie, err := authx.SessionCookie(body.Username)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	http.SetCookie(w, cookie)
	jsonWrite(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	if authx.NeedsSetup() {
		jsonError(w, http.StatusServiceUnavailable, "setup required")
		return
	}
	if authx.LoginThrottled(clientKey(r)) {
		jsonError(w, http.StatusTooManyRequests, "too many attempts; wait a moment")
		return
	}
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, err)
		return
	}
	ok := authx.Login(body.Username, body.Password)
	authx.RecordLoginResult(clientKey(r), ok)
	if !ok {
		jsonError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	cookie, err := authx.SessionCookie(body.Username)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	http.SetCookie(w, cookie)
	jsonWrite(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, authx.ClearCookie())
	jsonWrite(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	cfg := configx.Get()
	accounts := credentialEmails(cfg.Photos.Credentials)
	jsonWrite(w, http.StatusOK, map[string]any{
		"version":           s.Version,
		"username":          cfg.Auth.Username,
		"account":           cfg.Photos.Selected,
		"epoch":             s.epoch(),
		"accounts":          accounts,
		"needsTokenBinding": needsTokenBinding(cfg.Photos.Credentials, cfg.Photos.Selected),
		"settings":          redactPhotos(cfg.Photos),
		"autosyncSec":       cfg.LibrarySyncSec,
		"counts":            s.index.Counts(),
		"stats":             s.index.Stats(),
		"sync":              s.sync.Status(),
		"upload":            s.uploads.Status(),
		"resumable":         s.uploads.PeekResumable(),
	})
}

func (s *Server) epoch() uint64 {
	if s.AccountEpoch == nil {
		return 0
	}
	return s.AccountEpoch()
}

func credentialEmails(credentials []string) []string {
	out := make([]string, 0, len(credentials))
	for _, c := range credentials {
		if email := parseEmail(c); email != "" {
			out = append(out, email)
		}
	}
	return out
}

func parseEmail(cred string) string {
	if params, err := url.ParseQuery(cred); err == nil {
		return params.Get("Email")
	}
	for _, pair := range splitQuery(cred) {
		if strings.HasPrefix(pair, "Email=") {
			v := strings.TrimPrefix(pair, "Email=")
			if dec, err := url.QueryUnescape(v); err == nil {
				return dec
			}
			return v
		}
	}
	return ""
}

func splitQuery(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '&' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

func needsTokenBinding(creds []string, selected string) bool {
	for _, c := range creds {
		if credentials.Matches(parseEmail(c), selected) {
			return authNeedsBinding(c)
		}
	}
	return false
}

func authNeedsBinding(cred string) bool {
	has := func(k string) bool {
		for _, pair := range splitQuery(cred) {
			if len(pair) > len(k)+1 && pair[:len(k)+1] == k+"=" && pair[len(k)+1:] != "" {
				return true
			}
		}
		return false
	}
	if has("token_binding_alias") {
		return false
	}
	return has("assertion_jwt") || has("check_tb_upgrade_eligible")
}
