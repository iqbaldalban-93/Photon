package httpx

import (
	"compress/gzip"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"app/internal/proxyurl"
)

const MaxResponseBytes = int64(64 << 20)

type ClientOptions struct {
	MaxConnsPerHost       int
	ResponseHeaderTimeout time.Duration
}

func NewClientWithProxy(proxyURLStr string, opts ClientOptions) (*http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()

	maxConns := opts.MaxConnsPerHost
	if maxConns <= 0 {
		maxConns = 64
	}
	transport.MaxIdleConns = 100
	transport.MaxIdleConnsPerHost = maxConns
	transport.MaxConnsPerHost = maxConns
	transport.IdleConnTimeout = 90 * time.Second
	if opts.ResponseHeaderTimeout > 0 {
		transport.ResponseHeaderTimeout = opts.ResponseHeaderTimeout
	}

	proxyURL, err := proxyurl.Parse(proxyURLStr)
	if err != nil {
		return nil, err
	}
	if proxyURL != nil {
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	return &http.Client{Transport: transport, Timeout: 0}, nil
}

type RetryConfig struct {
	MaxRetries   int
	InitialDelay time.Duration
	MaxDelay     time.Duration
}

func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:   3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
	}
}

func ShouldRetry(resp *http.Response, err error) bool {
	if err != nil {
		return true
	}
	if resp == nil {
		return true
	}
	return ShouldRetryStatus(resp.StatusCode)
}

func ShouldRetryStatus(code int) bool {
	return code >= 500 || code == http.StatusTooManyRequests || code == http.StatusRequestTimeout
}

func CalculateBackoff(attempt int, config RetryConfig) time.Duration {
	delay := config.MaxDelay
	if attempt >= 0 && attempt < 30 {
		if d := config.InitialDelay * time.Duration(1<<uint(attempt)); d > 0 && d < delay {
			delay = d
		}
	}
	if tenth := int64(delay / 10); tenth > 0 {
		delay += time.Duration(rand.Int63n(tenth))
	}
	return delay
}

func ReadResponseBody(resp *http.Response) ([]byte, error) {
	var reader io.Reader = resp.Body
	if strings.Contains(strings.ToLower(resp.Header.Get("Content-Encoding")), "gzip") {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer func() { _ = gzipReader.Close() }()
		reader = gzipReader
	}
	data, err := io.ReadAll(io.LimitReader(reader, MaxResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > MaxResponseBytes {
		return nil, fmt.Errorf("response body exceeds %d bytes", MaxResponseBytes)
	}
	return data, nil
}

func ErrorBody(body []byte) string {
	s := strings.Join(strings.Fields(string(body)), " ")
	const limit = 300
	if len(s) > limit {
		s = s[:limit] + "..."
	}
	return s
}

type HTTPStatusError struct {
	Code int
	Body string
}

func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("request failed with status %d: %s", e.Code, e.Body)
}

func NewStatusError(resp *http.Response) *HTTPStatusError {
	body, _ := ReadResponseBody(resp)
	return &HTTPStatusError{Code: resp.StatusCode, Body: ErrorBody(body)}
}
