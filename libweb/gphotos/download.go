package gphotos

import (
	"context"
	"fmt"
	"net/http"

	"app/internal/httpx"
	"app/internal/protox"
)

type DownloadInfo struct {
	FileName       string `json:"fileName"`
	EditedURL      string `json:"editedUrl,omitempty"`
	OriginalURL    string `json:"originalUrl,omitempty"`
	VideoURL       string `json:"videoUrl,omitempty"`
	ContentVersion int64  `json:"contentVersion,omitempty"`
	CreatedAt      int64  `json:"createdAt,omitempty"`
	SizeBytes      int64  `json:"sizeBytes,omitempty"`
}

func (a *Api) GetDownloadURLs(ctx context.Context, mediaKey string) (*DownloadInfo, error) {
	body := map[string]any{
		"1": map[string]any{"1": map[string]any{"1": mediaKey}},
		"2": map[string]any{
			"1": map[string]any{"7": map[string]any{"2": map[string]any{}}},
			"5": map[string]any{"2": map[string]any{}, "3": map[string]any{}, "5": map[string]any{"1": map[string]any{}, "3": 0}},
		},
	}
	respBody, err := a.DoProto(ctx, prepareDownloadEndpoint, protox.FromMap(body), true)
	if err != nil {
		return nil, err
	}

	root, err := protox.Parse(respBody)
	if err != nil {
		return nil, fmt.Errorf("parse prepare-download response: %w", err)
	}
	f1 := root.Sub(1)
	if f1 == nil {
		return nil, fmt.Errorf("prepare-download response missing payload")
	}

	info := &DownloadInfo{}
	if f12 := f1.Sub(2); f12 != nil {
		info.FileName = f12.Str(4)
		info.CreatedAt = int64(f12.Uint(7))
		info.SizeBytes = int64(f12.Uint(10))
	}
	if f15 := f1.Sub(5); f15 != nil {
		if f152 := f15.Sub(2); f152 != nil {
			info.EditedURL = f152.Str(5)
			info.OriginalURL = f152.Str(6)
		}
		if f153 := f15.Sub(3); f153 != nil {
			info.VideoURL = f153.Str(5)
			info.ContentVersion = int64(f153.Uint(1))
		}
	}
	if info.OriginalURL == "" && info.EditedURL == "" && info.VideoURL == "" {
		return nil, fmt.Errorf("prepare-download returned no usable URL")
	}
	if info.FileName == "" {
		info.FileName = mediaKey
	}
	return info, nil
}

func (a *Api) GetStreamManifest(ctx context.Context, mediaKey, protocol string, contentVersion int64) (string, error) {
	if protocol == "" {
		protocol = "hls"
	}
	url := "https://lh3.googleusercontent.com/p/" + mediaKey + "%3Dmm," + protocol + "-vm"
	if contentVersion > 0 {
		url += fmt.Sprintf("-iv%d", contentVersion)
	}

	bearer, err := a.BearerToken()
	if err != nil {
		return "", fmt.Errorf("failed to get bearer token: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("accept-encoding", "gzip")
	req.Header.Set("User-Agent", a.UserAgent())
	req.Header.Set("Authorization", "Bearer "+bearer)

	resp, err := a.HTTP().Do(req)
	if err != nil {
		return "", fmt.Errorf("manifest request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := httpx.ReadResponseBody(resp)
		return "", fmt.Errorf("manifest request failed with status %d: %s", resp.StatusCode, truncateErrBody(b))
	}
	body, err := httpx.ReadResponseBody(resp)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func truncateErrBody(b []byte) string {
	s := string(b)
	if len(s) > 300 {
		s = s[:300] + "..."
	}
	return s
}
