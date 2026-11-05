package gauth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"app/internal/httpx"
	"app/internal/progress"
)

type UploadProgressCallback func(bytesUploaded, bytesTotal int64, attempt int)

type Uploader struct {
	Client    *http.Client
	Tokens    *TokenSource
	Language  string
	UserAgent string
}

func (u *Uploader) UploadFile(ctx context.Context, filePath string, uploadToken string, onProgress UploadProgressCallback) ([]byte, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("error getting file info: %w", err)
	}
	fileSize := fileInfo.Size()

	uploadURL := "https://photos.googleapis.com/data/upload/uploadmedia/interactive?upload_id=" + url.QueryEscape(uploadToken)
	retryConfig := httpx.DefaultRetryConfig()

	var lastErr error
	for attempt := 0; attempt <= retryConfig.MaxRetries; attempt++ {
		attemptNum := attempt + 1

		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		if attempt > 0 {
			delay := httpx.CalculateBackoff(attempt-1, retryConfig)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		if onProgress != nil {
			onProgress(0, fileSize, attemptNum)
		}

		file, err := os.Open(filePath)
		if err != nil {
			return nil, fmt.Errorf("error opening file: %w", err)
		}

		var reader io.Reader = file
		if onProgress != nil {
			reader = progress.New(file, fileSize, func(bytesRead, total int64) {
				onProgress(bytesRead, total, attemptNum)
			})
		}

		raw, retryable, err := u.doUploadRequest(ctx, uploadURL, reader, fileSize)
		closeErr := file.Close()
		if err == nil && closeErr != nil {
			return nil, fmt.Errorf("error closing file: %w", closeErr)
		}

		if err == nil {
			return raw, nil
		}

		lastErr = err

		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		if !retryable {
			return nil, fmt.Errorf("upload failed: %w", lastErr)
		}
	}

	return nil, fmt.Errorf("upload failed after %d attempts: %w", retryConfig.MaxRetries+1, lastErr)
}

func (u *Uploader) doUploadRequest(ctx context.Context, uploadURL string, reader io.Reader, fileSize int64) (raw []byte, retryable bool, err error) {
	req, err := http.NewRequestWithContext(ctx, "PUT", uploadURL, reader)
	if err != nil {
		return nil, false, fmt.Errorf("error creating request: %w", err)
	}

	req.ContentLength = fileSize

	bearerToken, err := u.Tokens.BearerToken()
	if err != nil {
		return nil, true, fmt.Errorf("failed to get bearer token: %w", err)
	}

	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Accept-Language", u.Language)
	req.Header.Set("User-Agent", u.UserAgent)
	req.Header.Set("Authorization", "Bearer "+bearerToken)

	resp, err := u.Client.Do(req)
	if err != nil {
		return nil, true, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusUnauthorized {
			u.Tokens.Invalidate()
		}
		return nil, httpx.ShouldRetryStatus(resp.StatusCode), httpx.NewStatusError(resp)
	}

	bodyBytes, err := httpx.ReadResponseBody(resp)
	if err != nil {
		return nil, false, fmt.Errorf("failed to read response body: %w", err)
	}
	return bodyBytes, false, nil
}
