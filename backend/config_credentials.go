package backend

import (
	"fmt"
	"net/url"
	"strings"

	"app/internal/adbtoken"
	"app/internal/credentials"
)

// Google-account credential management: add/remove the credential query
// strings, token-binding upgrades via ADB, and redaction for the frontend.

func (g *ConfigManager) AddCredentials(newAuthString string) error {
	requiredFields := []string{
		"androidId",
		"app",
		"client_sig",
		"Email",
		"Token",
		"lang",
		"service",
	}

	params, ok := credentials.Values(newAuthString)
	if !ok {
		return fmt.Errorf("invalid auth string format")
	}

	var missingFields []string
	for _, field := range requiredFields {
		if params.Get(field) == "" {
			missingFields = append(missingFields, field)
		}
	}
	if len(missingFields) > 0 {
		return fmt.Errorf("auth string missing required fields: %v", missingFields)
	}

	email := params.Get("Email")
	if email == "" {
		return fmt.Errorf("email cannot be empty")
	}

	var dupErr error
	err := setConfig(func(c *Config) {
		for _, cred := range c.Credentials {
			if credentials.Matches(credentials.Email(cred), email) {
				dupErr = fmt.Errorf("auth string with email %s already exists", email)
				return
			}
		}
		c.Credentials = append(c.Credentials, newAuthString)
		c.Selected = email
	})
	if err != nil {
		return err
	}
	if dupErr == nil {
		ReconcileLibraryAccount()
	}
	return dupErr
}

func (g *ConfigManager) CredentialNeedsTokenBinding(authString string) bool {
	params, ok := credentials.Values(authString)
	if !ok {
		return false
	}
	return credentialNeedsTokenBinding(params)
}

func (g *ConfigManager) AddTokenBindingAliasFromADB(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return fmt.Errorf("email cannot be empty")
	}

	alias, err := extractTokenBindingAliasFromADB(email)
	if err != nil {
		return err
	}

	return setConfig(func(c *Config) {
		for i, cred := range c.Credentials {
			params, ok := credentials.Values(cred)
			if !ok || !credentials.Matches(params.Get("Email"), email) {
				continue
			}
			params.Set("token_binding_alias", alias)
			c.Credentials[i] = params.Encode()
			return
		}
	})
}

func (g *ConfigManager) RemoveCredentials(email string) error {
	if email == "" {
		return fmt.Errorf("email cannot be empty")
	}

	found := false
	err := setConfig(func(c *Config) {
		updated := make([]string, 0, len(c.Credentials))
		for _, cred := range c.Credentials {
			if params, ok := credentials.Values(cred); ok && credentials.Matches(params.Get("Email"), email) {
				found = true
				continue
			}
			updated = append(updated, cred)
		}
		if !found {
			return
		}
		c.Credentials = updated
		if c.Selected == email {
			c.Selected = ""
		}
	})
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("no credentials found for email %s", email)
	}
	ReconcileLibraryAccount()
	return nil
}

func credentialNeedsTokenBinding(params url.Values) bool {
	if params.Get("token_binding_alias") != "" {
		return false
	}
	return params.Get("assertion_jwt") != "" ||
		params.Get("check_tb_upgrade_eligible") != ""
}

func extractTokenBindingAliasFromADB(email string) (string, error) {
	return adbtoken.ExtractAlias(email)
}

func redactCredentials(creds []string) []string {
	if len(creds) == 0 {
		return creds
	}
	out := make([]string, 0, len(creds))
	for _, c := range creds {
		params, ok := credentials.Values(c)
		if !ok || params.Get("Email") == "" {
			continue
		}
		r := url.Values{}
		r.Set("Email", params.Get("Email"))
		for _, k := range []string{"token_binding_alias", "assertion_jwt", "check_tb_upgrade_eligible"} {
			if params.Get(k) != "" {
				r.Set(k, "1")
			}
		}
		out = append(out, r.Encode())
	}
	return out
}
