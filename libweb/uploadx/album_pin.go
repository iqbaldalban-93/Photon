package uploadx

import (
	"fmt"
	"strings"
	"time"

	"app/internal/httpx"
	"app/internal/uploadcore"
	"app/libweb/gphotos"
)

// Album pinning: route uploaded media keys into a named or keyed album,
// batching adds and splitting oversized sets across numbered albums.

func (m *Manager) pinToAlbum(api *gphotos.Api, albumTarget string, mediaKeys []string) {
	const batchSize = 500
	var albumKey string
	var err error
	added := 0

	m.setStatus(func(s *Status) { s.Album = &AlbumProgress{AlbumName: albumTarget, TotalItems: len(mediaKeys)} })

	report := func(name string, complete bool, e error) {
		m.setStatus(func(s *Status) {
			s.Album = &AlbumProgress{AlbumName: name, ItemsAdded: added, TotalItems: len(mediaKeys), IsComplete: complete, Error: errStr(e)}
		})
	}

	fail := func(e error) {
		report(albumTarget, false, e)
	}

	isAlbumKey := len(albumTarget) > 6 && albumTarget[:6] == "AF1Qip"
	if isAlbumKey {
		albumKey = albumTarget
		for i := 0; i < len(mediaKeys); i += batchSize {
			end := min(i+batchSize, len(mediaKeys))
			if err = addWithRetry(api, albumKey, mediaKeys[i:end]); err != nil {
				fail(fmt.Errorf("failed to add media to album (added %d/%d): %w", added, len(mediaKeys), err))
				return
			}
			added += end - i
			report(albumTarget, false, nil)
		}
		report(albumTarget, true, nil)
		return
	}

	name := albumTarget
	var currentKey string
	for _, chunk := range uploadcore.PlanAlbumBatches(mediaKeys, albumTarget) {
		if chunk.IsFirstOfAlbum {
			name = chunk.Name
			currentKey, err = createAlbumWithRetry(api, name, chunk.Batch)
			if err != nil {
				fail(fmt.Errorf("failed to create album '%s': %w", name, err))
				return
			}
			albumKey = currentKey
		} else {
			err = addWithRetry(api, currentKey, chunk.Batch)
			if err != nil {
				fail(fmt.Errorf("failed to add media to album '%s': %w", name, err))
				return
			}
		}
		added += len(chunk.Batch)
		report(name, false, nil)
	}
	report(albumTarget, true, nil)
}

func withRetry(op func() error) error {
	rc := httpx.DefaultRetryConfig()
	var lastErr error
	for attempt := 0; attempt <= rc.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(httpx.CalculateBackoff(attempt-1, rc))
		}
		err := op()
		if err == nil {
			return nil
		}
		lastErr = err
		if !retryableAlbumErr(err) {
			return err
		}
	}
	return fmt.Errorf("failed after %d attempts: %w", rc.MaxRetries+1, lastErr)
}

func createAlbumWithRetry(api *gphotos.Api, name string, keys []string) (string, error) {
	var key string
	err := withRetry(func() error {
		var err error
		key, err = api.CreateAlbum(contextBackground(), name, keys)
		return err
	})
	if err != nil {
		return "", err
	}
	return key, nil
}

func addWithRetry(api *gphotos.Api, albumKey string, keys []string) error {
	return withRetry(func() error {
		return api.AddMediaToAlbum(contextBackground(), albumKey, keys)
	})
}

func retryableAlbumErr(err error) bool {
	s := err.Error()
	return strings.Contains(s, "status 5") || strings.Contains(s, "status 429") ||
		strings.Contains(s, "connection") || strings.Contains(s, "timeout")
}
