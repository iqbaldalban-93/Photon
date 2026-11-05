package backend

import (
	"net/http"
	"time"

	"app/internal/httpx"
)

const maxConnsPerHost = 64

func NewHTTPClientWithProxy(proxyURLStr string) (*http.Client, error) {
	return httpx.NewClientWithProxy(proxyURLStr, httpx.ClientOptions{
		MaxConnsPerHost:       maxConnsPerHost,
		ResponseHeaderTimeout: 3 * time.Minute,
	})
}

type RetryConfig = httpx.RetryConfig

func DefaultRetryConfig() RetryConfig { return httpx.DefaultRetryConfig() }

func ShouldRetry(resp *http.Response, err error) bool { return httpx.ShouldRetry(resp, err) }

func ShouldRetryStatus(code int) bool { return httpx.ShouldRetryStatus(code) }

func CalculateBackoff(attempt int, config RetryConfig) time.Duration {
	return httpx.CalculateBackoff(attempt, config)
}

func ReadResponseBody(resp *http.Response) ([]byte, error) { return httpx.ReadResponseBody(resp) }

type HTTPStatusError = httpx.HTTPStatusError

func newStatusError(resp *http.Response) *HTTPStatusError { return httpx.NewStatusError(resp) }
