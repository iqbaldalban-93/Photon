// Package gphotos implements the Google Photos Android-API client shared by
// the Wails desktop layer and the browser-library layer. One transport, one
// set of endpoint builders, parameterized by caller (proxy, connection pool,
// commit retry policy).
package gphotos

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"app/internal/credentials"
	"app/internal/gauth"
	"app/internal/httpx"
)

const (
	// Endpoint constants shared by both layers.
	PhotosCreateMediaItemsEndpoint = "https://photosdata-pa.googleapis.com/6439526531001121323/16538846908252377752"
	HashCheckEndpoint              = "https://photosdata-pa.googleapis.com/6439526531001121323/5084965799730810217"
	CreateAlbumEndpoint            = "https://photosdata-pa.googleapis.com/6439526531001121323/8386163679468898444"
	AddMediaToAlbumEndpoint        = "https://photosdata-pa.googleapis.com/6439526531001121323/484917746253879292"
	DeleteAlbumEndpoint            = "https://photosdata-pa.googleapis.com/6439526531001121323/11165707358190966680"
	RenameAlbumEndpoint            = "https://photosdata-pa.googleapis.com/6439526531001121323/16466587394238175348"

	defaultModel           = "Pixel XL"
	defaultMake            = "Google"
	clientVersionCodeInt64 = int64(49029607)
	androidAPIVersionInt64 = int64(28)

	// RequestTimeout bounds every proto RPC.
	RequestTimeout = 2 * time.Minute
)

// CommitDevice describes the device identity sent with commit/album requests.
// The storage-quality flags map onto the model Google expects.
type CommitDevice struct {
	Model   string
	Quality int64
}

// DeviceFromFlags maps storage-quality flags onto the commit device.
func DeviceFromFlags(saver, useQuota bool) CommitDevice {
	model := defaultModel
	quality := int64(3)
	if saver {
		model = "Pixel 2"
		quality = 1
	}
	if useQuota {
		model = "Pixel 8"
	}
	return CommitDevice{Model: model, Quality: quality}
}

type Options struct {
	Credentials string
	Proxy       string
	// MaxConnsPerHost tunes the connection pool; 0 defaults to 10.
	MaxConnsPerHost int
	// CommitRetry overrides the commit retry policy; nil uses the default.
	CommitRetry *httpx.RetryConfig
}

type Client struct {
	authData  string
	language  string
	userAgent string
	model     string
	make      string

	client   *http.Client
	tokens   *gauth.TokenSource
	uploader *gauth.Uploader

	commitRetry *httpx.RetryConfig
}

func NewClient(opts Options) (*Client, error) {
	params, ok := credentials.Values(opts.Credentials)
	if !ok {
		return nil, fmt.Errorf("failed to parse credentials")
	}

	client, err := httpx.NewClientWithProxy(opts.Proxy, httpx.ClientOptions{
		MaxConnsPerHost:       opts.MaxConnsPerHost,
		ResponseHeaderTimeout: 3 * time.Minute,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	language := params.Get("lang")
	if language == "" {
		language = "en_US"
	}
	userAgent := fmt.Sprintf(
		"com.google.android.apps.photos/%d (Linux; U; Android 9; %s; %s; Build/PQ2A.190205.001; Cronet/127.0.6510.5) (gzip)",
		clientVersionCodeInt64, language, defaultModel,
	)
	tokens := gauth.NewTokenSource(strings.TrimSpace(opts.Credentials), client)

	return &Client{
		authData:  strings.TrimSpace(opts.Credentials),
		language:  language,
		userAgent: userAgent,
		model:     defaultModel,
		make:      defaultMake,
		client:    client,
		tokens:    tokens,
		uploader: &gauth.Uploader{
			Client:    client,
			Tokens:    tokens,
			Language:  language,
			UserAgent: userAgent,
		},
		commitRetry: opts.CommitRetry,
	}, nil
}

// Email returns the account email bound to this client's credentials.
func (c *Client) Email() string {
	return credentials.Email(c.authData)
}

// AuthData returns the credential query string the client was built from.
func (c *Client) AuthData() string { return c.authData }

// BearerToken returns a (cached, refreshable) OAuth bearer token.
func (c *Client) BearerToken() (string, error) {
	return c.tokens.BearerToken()
}

// InvalidateToken drops the cached bearer token so the next call re-auths.
func (c *Client) InvalidateToken() {
	c.tokens.Invalidate()
}

// HTTP exposes the underlying HTTP client for direct (non-proto) requests.
func (c *Client) HTTP() *http.Client { return c.client }

// UserAgent returns the client's User-Agent string.
func (c *Client) UserAgent() string { return c.userAgent }

func (c *Client) protoHeaders(bearer string, withExt bool) map[string]string {
	h := map[string]string{
		"Accept-Encoding": "gzip",
		"Accept-Language": c.language,
		"Content-Type":    "application/x-protobuf",
		"User-Agent":      c.userAgent,
		"Authorization":   "Bearer " + bearer,
	}
	if withExt {
		h["x-goog-ext-173412678-bin"] = "CgcIAhClARgC"
		h["x-goog-ext-174067345-bin"] = "CgIIAg=="
	}
	return h
}

// DoProto POSTs a pre-serialized protobuf request and returns the response
// body. Non-2xx responses invalidate the token on 401 and return a typed
// *httpx.HTTPStatusError.
func (c *Client) DoProto(ctx context.Context, endpoint string, body []byte, withExt bool) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()
	bearer, err := c.BearerToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get bearer token: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	for k, v := range c.protoHeaders(bearer, withExt) {
		req.Header.Set(k, v)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusUnauthorized {
			c.InvalidateToken()
		}
		return nil, httpx.NewStatusError(resp)
	}
	return httpx.ReadResponseBody(resp)
}

// DoProtoHeader is DoProto for the upload-token flow, which needs a response
// header instead of (only) a body.
func (c *Client) DoProtoHeader(ctx context.Context, endpoint string, body []byte, withExt bool, extraHeaders map[string]string) ([]byte, http.Header, error) {
	ctx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()
	bearer, err := c.BearerToken()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get bearer token: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}
	for k, v := range c.protoHeaders(bearer, withExt) {
		req.Header.Set(k, v)
	}
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusUnauthorized {
			c.InvalidateToken()
		}
		return nil, nil, httpx.NewStatusError(resp)
	}
	header := resp.Header
	bodyBytes, err := httpx.ReadResponseBody(resp)
	return bodyBytes, header, err
}
