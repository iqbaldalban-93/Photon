package server

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"app/internal/credentials"
	"app/libweb/configx"
	"app/libweb/gphotos"
)

// Account management endpoints: list/add/remove/select Google accounts and
// upgrade credentials with an ADB-extracted token-binding alias.

func (s *Server) handleAccountGet(w http.ResponseWriter, r *http.Request) {
	cfg := configx.Get()
	type acct struct {
		Email     string `json:"email"`
		NeedsBind bool   `json:"needsTokenBinding"`
	}
	accounts := make([]acct, 0, len(cfg.Photos.Credentials))
	for _, c := range cfg.Photos.Credentials {
		email := parseEmail(c)
		if email == "" {
			continue
		}
		accounts = append(accounts, acct{Email: email, NeedsBind: authNeedsBinding(c)})
	}
	jsonWrite(w, http.StatusOK, map[string]any{
		"selected": cfg.Photos.Selected,
		"accounts": accounts,
	})
}

func (s *Server) handleAccountAdd(w http.ResponseWriter, r *http.Request) {
	var body struct {
		AuthString string `json:"authString"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, err)
		return
	}
	cred := strings.TrimSpace(body.AuthString)
	required := []string{"androidId", "app", "client_sig", "Email", "Token", "lang", "service"}
	var missing []string
	for _, f := range required {
		if !queryHas(cred, f) {
			missing = append(missing, f)
		}
	}
	if len(missing) > 0 {
		jsonError(w, http.StatusBadRequest, "auth string missing required fields: "+fmt.Sprint(missing))
		return
	}
	email := parseEmail(cred)

	err := configx.Update(func(c *configx.Config) {
		for _, existing := range c.Photos.Credentials {
			if credentials.Matches(parseEmail(existing), email) {
				var updated []string
				for _, e := range c.Photos.Credentials {
					if !credentials.Matches(parseEmail(e), email) {
						updated = append(updated, e)
					}
				}
				c.Photos.Credentials = updated
				break
			}
		}
		c.Photos.Credentials = append(c.Photos.Credentials, cred)
		c.Photos.Selected = email
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	if s.AccountsChanged != nil {
		s.AccountsChanged()
	}
	jsonWrite(w, http.StatusOK, map[string]string{"selected": email})
}

func queryHas(cred, key string) bool {
	for _, pair := range splitQuery(cred) {
		if strings.HasPrefix(pair, key+"=") && len(pair) > len(key)+1 {
			return true
		}
	}
	return false
}

func (s *Server) handleAccountRemove(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		jsonError(w, http.StatusBadRequest, "missing email")
		return
	}
	err := configx.Update(func(c *configx.Config) {
		var updated []string
		for _, e := range c.Photos.Credentials {
			if !credentials.Matches(parseEmail(e), email) {
				updated = append(updated, e)
			}
		}
		c.Photos.Credentials = updated
		if credentials.Matches(c.Photos.Selected, email) {
			c.Photos.Selected = ""
		}
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	if s.AccountsChanged != nil {
		s.AccountsChanged()
	}
	jsonWrite(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleAccountSelect(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
		Force bool   `json:"force"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, err)
		return
	}
	found := false
	for _, c := range configx.Get().Photos.Credentials {
		if credentials.Matches(parseEmail(c), body.Email) {
			found = true
			break
		}
	}
	if !found {
		jsonError(w, http.StatusNotFound, "no such account")
		return
	}
	if s.SwitchAccount != nil {
		if err := s.SwitchAccount(body.Email, body.Force); err != nil {
			if errors.Is(err, ErrUploadsRunning) {
				jsonError(w, http.StatusConflict, err.Error())
				return
			}
			writeErr(w, err)
			return
		}
		jsonWrite(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}
	if err := configx.Update(func(c *configx.Config) { c.Photos.Selected = body.Email }); err != nil {
		writeErr(w, err)
		return
	}
	jsonWrite(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleAccountTokenBinding(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, err)
		return
	}
	var target string
	for _, c := range configx.Get().Photos.Credentials {
		if credentials.Matches(parseEmail(c), body.Email) {
			target = c
			break
		}
	}
	if target == "" {
		jsonError(w, http.StatusNotFound, "no such account")
		return
	}
	alias, err := gphotos.ExtractTokenBindingAliasFromADB(body.Email)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	updated, err := gphotos.AttachTokenBindingAliasFromADB(target, alias)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := configx.Update(func(c *configx.Config) {
		for i, e := range c.Photos.Credentials {
			if credentials.Matches(parseEmail(e), body.Email) {
				c.Photos.Credentials[i] = updated
			}
		}
	}); err != nil {
		writeErr(w, err)
		return
	}
	jsonWrite(w, http.StatusOK, map[string]bool{"ok": true})
}
