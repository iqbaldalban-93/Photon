package gauth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"app/internal/httpx"
	"app/internal/tokenbinding"
)

const (
	authEndpoint       = "https://android.googleapis.com/auth"
	authRequestTimeout = 60 * time.Second
	expirySkew         = 60
)

type TokenSource struct {
	AuthData string
	Client   *http.Client

	mu    sync.Mutex
	cache map[string]string
}

func NewTokenSource(authData string, client *http.Client) *TokenSource {
	return &TokenSource{
		AuthData: authData,
		Client:   client,
		cache: map[string]string{
			"Expiry": "0",
			"Auth":   "",
		},
	}
}

func (t *TokenSource) BearerToken() (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	expiryStr := t.cache["Expiry"]
	expiry, err := strconv.ParseInt(expiryStr, 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid expiry time: %w", err)
	}

	if expiry <= time.Now().Unix()+expirySkew {
		resp, err := t.getAuthToken()
		if err != nil {
			return "", fmt.Errorf("failed to get auth token: %w", err)
		}
		t.cache = resp
	}

	if token, ok := t.cache["Auth"]; ok && token != "" {
		return token, nil
	}

	return "", errors.New("auth response does not contain bearer token")
}

func (t *TokenSource) Invalidate() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cache != nil {
		t.cache["Expiry"] = "0"
	}
}

func (t *TokenSource) getAuthToken() (map[string]string, error) {
	authDataValues, err := url.ParseQuery(t.AuthData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse auth data: %w", err)
	}

	authRequestData := url.Values{}
	for key, values := range authDataValues {
		authRequestData[key] = append([]string(nil), values...)
	}
	authRequestData.Set("app", "com.google.android.apps.photos")
	authRequestData.Set("callerPkg", "com.google.android.apps.photos")
	authRequestData.Del("it_caveat_types")
	authRequestData.Del("assertion_jwt")

	var tokenBinding *tokenbinding.Session
	if alias := authRequestData.Get("token_binding_alias"); alias != "" {
		var assertionJWT string
		tokenBinding, assertionJWT, err = tokenbinding.NewSession(alias)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare token binding assertion: %w", err)
		}
		authRequestData.Set("assertion_jwt", assertionJWT)
	}
	authRequestData.Del("token_binding_alias")

	headers := map[string]string{
		"Accept-Encoding": "gzip",
		"app":             "com.google.android.apps.photos",
		"Connection":      "Keep-Alive",
		"Content-Type":    "application/x-www-form-urlencoded",
		"device":          authRequestData.Get("androidId"),
		"User-Agent":      "GoogleAuth/1.4 (Pixel XL PQ2A.190205.001); gzip",
	}

	authCtx, cancelAuth := context.WithTimeout(context.Background(), authRequestTimeout)
	defer cancelAuth()
	req, err := http.NewRequestWithContext(
		authCtx,
		"POST",
		authEndpoint,
		strings.NewReader(authRequestData.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := t.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("auth request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, httpx.NewStatusError(resp)
	}

	bodyBytes, err := httpx.ReadResponseBody(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	parsedAuthResponse := ParseAuthResponse(string(bodyBytes))
	if err := tokenbinding.DecryptEncryptedResponse(parsedAuthResponse, tokenBinding); err != nil {
		return nil, err
	}

	if parsedAuthResponse["Auth"] == "" {
		return nil, errors.New("auth response missing Auth token")
	}
	if parsedAuthResponse["Expiry"] == "" {
		return nil, errors.New("auth response missing Expiry")
	}

	return parsedAuthResponse, nil
}

func ParseAuthResponse(body string) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if parts := strings.SplitN(line, "=", 2); len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}
