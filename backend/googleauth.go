package backend

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"app/internal/credentials"
	"app/internal/gauth"
	"app/internal/proxyurl"
)

const (
	embeddedSetupAuthEndpoint = "https://android.clients.google.com/auth"
	googleAuthEmailHint       = "oauth-token@example.com"
	googlePlayServicesSig     = "38918a453d07199354f8b19af05ec6562ced5788"
	googlePhotosPackage       = "com.google.android.apps.photos"
	googlePhotosSig           = "24bb24c05e47e0aefa68a58a766179d9b613a600"
	googlePhotosService       = "oauth2:openid https://www.googleapis.com/auth/mobileapps.native https://www.googleapis.com/auth/photos.native"
)

type googleAccountAuthDependencies struct {
	client             *http.Client
	authEndpoint       string
	generateAndroidID  func() (string, error)
	validateCredential func(context.Context, string) error
}

type googleAuthExchange struct {
	Email       string
	MasterToken string
}

func (g *ConfigManager) AddGoogleAccount(oauthToken string) (string, error) {
	proxy := CurrentConfig().Proxy

	client, err := newGoogleAuthHTTPClient(proxy)
	if err != nil {
		return "", fmt.Errorf("prepare Google authentication: %w", err)
	}

	deps := googleAccountAuthDependencies{
		client:            client,
		authEndpoint:      embeddedSetupAuthEndpoint,
		generateAndroidID: generateAndroidID,
		validateCredential: func(ctx context.Context, credential string) error {
			return validateGooglePhotosCredential(ctx, credential, proxy)
		},
	}
	return g.addGoogleAccount(context.Background(), oauthToken, deps)
}

func (g *ConfigManager) addGoogleAccount(
	ctx context.Context,
	oauthToken string,
	deps googleAccountAuthDependencies,
) (string, error) {
	oauthToken, err := normalizeOAuthToken(oauthToken)
	if err != nil {
		return "", err
	}

	androidID, err := deps.generateAndroidID()
	if err != nil {
		return "", fmt.Errorf("generate Android ID: %w", err)
	}

	exchange, err := exchangeEmbeddedSetupToken(ctx, deps.client, deps.authEndpoint, oauthToken, androidID)
	if err != nil {
		return "", err
	}

	credential := buildGooglePhotosCredential(exchange.Email, exchange.MasterToken, androidID)
	if err := deps.validateCredential(ctx, credential); err != nil {
		return "", fmt.Errorf("Google Photos rejected the new credential: %w", err)
	}

	if err := upsertCredential(credential); err != nil {
		return "", fmt.Errorf("save Google account: %w", err)
	}
	return exchange.Email, nil
}

func newGoogleAuthHTTPClient(proxy string) (*http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	} else {
		transport.TLSClientConfig = transport.TLSClientConfig.Clone()
	}
	transport.TLSClientConfig.InsecureSkipVerify = false
	transport.ForceAttemptHTTP2 = true
	transport.TLSNextProto = nil

	proxyURL, err := proxyurl.Parse(proxy)
	if err != nil {
		return nil, err
	}
	if proxyURL != nil {
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	return &http.Client{
		Transport:     transport,
		Timeout:       30 * time.Second,
		CheckRedirect: rejectGoogleAuthRedirect,
	}, nil
}

func rejectGoogleAuthRedirect(_ *http.Request, _ []*http.Request) error {
	return http.ErrUseLastResponse
}

func exchangeEmbeddedSetupToken(
	ctx context.Context,
	client *http.Client,
	endpoint string,
	oauthToken string,
	androidID string,
) (googleAuthExchange, error) {
	form := url.Values{
		"accountType":                  {"HOSTED_OR_GOOGLE"},
		"Email":                        {googleAuthEmailHint},
		"has_permission":               {"1"},
		"add_account":                  {"1"},
		"ACCESS_TOKEN":                 {"1"},
		"Token":                        {oauthToken},
		"service":                      {"ac2dm"},
		"source":                       {"android"},
		"androidId":                    {androidID},
		"device_country":               {"us"},
		"operatorCountry":              {"us"},
		"lang":                         {"en"},
		"sdk_version":                  {"17"},
		"google_play_services_version": {"240913000"},
		"client_sig":                   {googlePlayServicesSig},
		"callerSig":                    {googlePlayServicesSig},
		"droidguard_results":           {"dummy123"},
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return googleAuthExchange{}, fmt.Errorf("prepare Google token exchange: %w", err)
	}
	request.Header.Set("Accept-Encoding", "identity")
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("User-Agent", "GoogleAuth/1.4")

	response, err := client.Do(request)
	if err != nil {
		return googleAuthExchange{}, fmt.Errorf("contact Google authentication: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(response.Body, 64*1024))
	if err != nil {
		return googleAuthExchange{}, fmt.Errorf("read Google authentication response: %w", err)
	}
	values := parseGoogleAuthResponse(string(body))
	if authError := values["Error"]; authError != "" {
		return googleAuthExchange{}, googleAuthError(authError)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return googleAuthExchange{}, fmt.Errorf("Google authentication returned HTTP %d", response.StatusCode)
	}

	masterToken := values["Token"]
	if masterToken == "" {
		return googleAuthExchange{}, errors.New("Google authentication response did not contain a master token")
	}
	email, err := normalizeGoogleEmail(values["Email"])
	if err != nil {
		return googleAuthExchange{}, errors.New("Google authentication response did not contain a valid account email")
	}
	return googleAuthExchange{Email: email, MasterToken: masterToken}, nil
}

func parseGoogleAuthResponse(body string) map[string]string {
	return gauth.ParseAuthResponse(body)
}

func googleAuthError(code string) error {
	switch code {
	case "BadAuthentication":
		return errors.New("Google rejected the oauth_token; obtain a fresh cookie and try again")
	case "NeedsBrowser":
		return errors.New("Google requires a fresh Embedded Setup sign-in")
	case "MissingDroidguard":
		return errors.New("Google rejected the device verification data")
	default:
		return fmt.Errorf("Google authentication failed with %s", code)
	}
}

func normalizeGoogleEmail(value string) (string, error) {
	value = strings.TrimSpace(value)
	address, err := mail.ParseAddress(value)
	if err != nil || address.Address != value {
		return "", errors.New("enter a valid Google account email address")
	}
	return address.Address, nil
}

func normalizeOAuthToken(value string) (string, error) {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "oauth_token=")
	if len(value) < 16 || len(value) > 8192 || strings.ContainsAny(value, "\r\n") {
		return "", errors.New("enter the oauth_token cookie value from Google Embedded Setup")
	}
	return value, nil
}

func generateAndroidID() (string, error) {
	raw := make([]byte, 8)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func buildGooglePhotosCredential(email, masterToken, androidID string) string {
	return url.Values{
		"androidId":                    {androidID},
		"app":                          {googlePhotosPackage},
		"callerPkg":                    {googlePhotosPackage},
		"callerSig":                    {googlePhotosSig},
		"client_sig":                   {googlePhotosSig},
		"device_country":               {"us"},
		"Email":                        {email},
		"google_play_services_version": {"240913000"},
		"lang":                         {"en_US"},
		"oauth2_foreground":            {"1"},
		"operatorCountry":              {"us"},
		"sdk_version":                  {"33"},
		"service":                      {googlePhotosService},
		"source":                       {"android"},
		"Token":                        {masterToken},
	}.Encode()
}

func validateGooglePhotosCredential(ctx context.Context, credential, proxy string) error {
	api, err := newAPIFromCredential(credential, proxy)
	if err != nil {
		return err
	}
	api.HTTPClient().CheckRedirect = rejectGoogleAuthRedirect
	if _, err := api.BearerToken(); err != nil {
		return err
	}
	mediaKey, err := api.FindRemoteMediaByHash(ctx, make([]byte, 20))
	if err != nil {
		return err
	}
	if mediaKey != "" {
		return errors.New("unexpected media match for validation hash")
	}
	return nil
}

func upsertCredential(credential string) error {
	values, ok := credentials.Values(credential)
	if !ok {
		return fmt.Errorf("parse generated credential failed")
	}
	email := values.Get("Email")

	if err := setConfig(func(c *Config) {
		replaced := false
		for i, existing := range c.Credentials {
			if ev, ok := credentials.Values(existing); ok && credentials.Matches(ev.Get("Email"), email) {
				c.Credentials[i] = credential
				replaced = true
				break
			}
		}
		if !replaced {
			c.Credentials = append(c.Credentials, credential)
		}
		c.Selected = email
	}); err != nil {
		return err
	}
	ReconcileLibraryAccount()
	return nil
}
