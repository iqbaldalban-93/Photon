package gphotos

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"app/internal/gauth"
	"app/internal/httpx"
	"app/internal/protox"
)

// Upload endpoints: upload token acquisition, hash-check dedupe lookup,
// resumable file upload, and the retried commit call.

const uploadTokenEndpoint = "https://photos.googleapis.com/data/upload/uploadmedia/interactive"

func (c *Client) GetUploadToken(ctx context.Context, shaHashB64 string, fileSize int64) (string, error) {
	body := protox.NewBuilder().
		Int(1, 2).Int(2, 2).Int(3, 1).Int(4, 3).Int(7, fileSize).
		Bytes_()

	_, header, err := c.DoProtoHeader(ctx, uploadTokenEndpoint, body, false, map[string]string{
		"X-Goog-Hash":             "sha1=" + shaHashB64,
		"X-Upload-Content-Length": strconv.FormatInt(fileSize, 10),
	})
	if err != nil {
		return "", err
	}

	uploadToken := header.Get("X-GUploader-UploadID")
	if uploadToken == "" {
		return "", errors.New("response missing X-GUploader-UploadID header")
	}
	return uploadToken, nil
}

func (c *Client) FindRemoteMediaByHash(ctx context.Context, shaHash []byte) (string, error) {
	b := protox.NewBuilder()
	b.Msg(1, func(m *protox.Builder) {
		m.Msg(1, func(h *protox.Builder) { h.Bytes(1, shaHash) })
		m.Msg(2, func(*protox.Builder) {})
	})
	body := b.Bytes_()

	respBody, err := c.DoProto(ctx, HashCheckEndpoint, body, false)
	if err != nil {
		return "", err
	}

	msg, err := protox.Parse(respBody)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal protobuf: %w", err)
	}
	if f1 := msg.Sub(1); f1 != nil {
		if f2 := f1.Sub(2); f2 != nil {
			if f2b := f2.Sub(2); f2b != nil {
				return f2b.Str(1), nil
			}
		}
	}
	return "", nil
}

type UploadProgressCallback = gauth.UploadProgressCallback

func (c *Client) UploadFileWithProgress(ctx context.Context, filePath string, uploadToken string, onProgress UploadProgressCallback) (ScottyFinalizeToken, error) {
	raw, err := c.uploader.UploadFile(ctx, filePath, uploadToken, onProgress)
	if err != nil {
		return ScottyFinalizeToken{}, err
	}
	token, err := ParseScottyFinalizeToken(raw)
	if err != nil {
		return ScottyFinalizeToken{}, fmt.Errorf("invalid upload finalize response: %w", err)
	}
	return token, nil
}

// CommitUpload commits an upload as the default device (Pixel XL, original
// quality). Prefer CommitUploadAs when storage-quality flags are in effect.
func (c *Client) CommitUpload(ctx context.Context, finalizeToken ScottyFinalizeToken, fileName string, sha1Hash []byte, uploadTimestamp int64) (string, error) {
	return c.CommitUploadAs(ctx, finalizeToken, fileName, sha1Hash, uploadTimestamp, DeviceFromFlags(false, false))
}

// CommitUploadAs commits an upload under a specific device identity, retrying
// on retryable transport/status failures.
func (c *Client) CommitUploadAs(ctx context.Context, finalizeToken ScottyFinalizeToken, fileName string, sha1Hash []byte, uploadTimestamp int64, device CommitDevice) (string, error) {
	if uploadTimestamp == 0 {
		uploadTimestamp = time.Now().Unix()
	}

	body := commitRequestBody(finalizeToken.Raw, fileName, sha1Hash, uploadTimestamp, device)
	return c.commitSerialized(ctx, body)
}

func commitRequestBody(tokenRaw []byte, fileName string, sha1Hash []byte, uploadTimestamp int64, device CommitDevice) []byte {
	unknownInt := int64(46000000)
	b := protox.NewBuilder()
	b.Msg(1, func(f1 *protox.Builder) {
		f1.Bytes(1, tokenRaw)
		f1.Str(2, fileName)
		f1.Bytes(3, sha1Hash)
		f1.Msg(4, func(f4 *protox.Builder) {
			f4.Int(1, uploadTimestamp)
			f4.Int(2, unknownInt)
		})
		f1.Int(7, device.Quality)
		f1.Int(10, 1)
	})
	b.Msg(2, func(f2 *protox.Builder) {
		f2.Str(3, device.Model)
		f2.Str(4, defaultMake)
		f2.Int(5, androidAPIVersionInt64)
	})
	b.Bytes(3, []byte{1, 3})
	return b.Bytes_()
}

func (c *Client) commitSerialized(ctx context.Context, serializedData []byte) (string, error) {
	retryConfig := httpx.DefaultRetryConfig()
	if c.commitRetry != nil {
		retryConfig = *c.commitRetry
	}
	var lastErr error
	for attempt := 0; attempt <= retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			delay := httpx.CalculateBackoff(attempt-1, retryConfig)
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(delay):
			}
		}
		mediaKey, retryable, err := c.doCommitRequest(ctx, serializedData)
		if err == nil {
			return mediaKey, nil
		}
		lastErr = err
		if !retryable {
			return "", fmt.Errorf("commit failed after %d attempt(s): %w", attempt+1, lastErr)
		}
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
	}
	return "", fmt.Errorf("commit failed after %d attempts: %w", retryConfig.MaxRetries+1, lastErr)
}

func (c *Client) doCommitRequest(ctx context.Context, serializedData []byte) (mediaKey string, retryable bool, err error) {
	bodyBytes, err := c.DoProto(ctx, PhotosCreateMediaItemsEndpoint, serializedData, true)
	if err != nil {
		var se *httpx.HTTPStatusError
		if errors.As(err, &se) {
			// Unauthorized and retryable status errors are worth retrying;
			// 401 invalidates the cached token inside DoProto already.
			retryable = httpx.ShouldRetryStatus(se.Code) || se.Code == 401
		} else {
			retryable = true
		}
		return "", retryable, err
	}

	mediaKey, err = parseCreateMediaItemsResponse(bodyBytes)
	if err != nil {
		return "", false, fmt.Errorf("failed to parse accepted response: %w", err)
	}
	return mediaKey, false, nil
}

func parseCreateMediaItemsResponse(responseBytes []byte) (string, error) {
	msg, err := protox.Parse(responseBytes)
	if err != nil {
		return "", fmt.Errorf("unmarshal create-media response: %w", err)
	}
	for _, item := range msg.Msgs(1) {
		if result := item.Sub(3); result != nil {
			if key := result.Str(1); key != "" {
				return key, nil
			}
		}
	}
	return "", fmt.Errorf("upload rejected by API: media key is empty or missing")
}
