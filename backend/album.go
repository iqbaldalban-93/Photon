package backend

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"app/internal/ctxrelay"
	"app/internal/uploadcore"
)

const (
	AlbumBatchSize = uploadcore.AlbumBatchSize
	AlbumLimit     = uploadcore.AlbumLimit
)

type AlbumStatus struct {
	AlbumName  string   `json:"AlbumName"`
	ItemsAdded int      `json:"ItemsAdded"`
	TotalItems int      `json:"TotalItems"`
	AlbumKeys  []string `json:"AlbumKeys"`
	IsComplete bool     `json:"IsComplete"`
	Paths      []string `json:"Paths"`
}

type AlbumError struct {
	AlbumName string `json:"AlbumName"`
	Error     string `json:"Error"`
	IsError   bool   `json:"IsError"`
}

type AlbumManager struct {
	api    *Api
	app    AppInterface
	cancel <-chan struct{}
	ctx    context.Context
}

type createdAlbum struct {
	key   string
	items int
}

var sessionAlbums = struct {
	sync.Mutex
	m map[string]createdAlbum
}{m: map[string]createdAlbum{}}

func sessionAlbumKey(account, name string) string { return account + "\x00" + name }

func lookupSessionAlbum(account, name string, adding int) (string, bool) {
	sessionAlbums.Lock()
	defer sessionAlbums.Unlock()
	c, ok := sessionAlbums.m[sessionAlbumKey(account, name)]
	if !ok || c.items+adding > AlbumLimit {
		return "", false
	}
	return c.key, true
}

func rememberSessionAlbum(account, name, key string, items int) {
	sessionAlbums.Lock()
	defer sessionAlbums.Unlock()
	k := sessionAlbumKey(account, name)
	c := sessionAlbums.m[k]
	if c.key != key {
		c = createdAlbum{key: key}
	}
	c.items += items
	sessionAlbums.m[k] = c
}

func NewAlbumManager(api *Api, app AppInterface, cancel <-chan struct{}) *AlbumManager {
	return &AlbumManager{
		api:    api,
		app:    app,
		cancel: cancel,
	}
}

func (m *AlbumManager) isCancelled() bool {
	if m.cancel == nil {
		return false
	}
	select {
	case <-m.cancel:
		return true
	default:
		return false
	}
}

func IsAlbumKey(input string) bool {
	return len(input) > 6 && input[:6] == "AF1Qip"
}

func albumDisplayName(key string) string {
	if len(key) > 10 {
		key = key[:10]
	}
	return fmt.Sprintf("Album (%s...)", key)
}

func (m *AlbumManager) AddToAlbum(mediaKeys []string, paths []string, albumNameOrKey string) ([]string, error) {
	if len(mediaKeys) == 0 {
		return nil, fmt.Errorf("no media keys provided")
	}

	albumNameOrKey = strings.TrimSpace(albumNameOrKey)
	if albumNameOrKey == "" {
		return nil, fmt.Errorf("album name or key cannot be empty")
	}

	mediaKeys, paths = dedupeMediaKeys(mediaKeys, paths)

	ctx, stop := ctxrelay.RelayCancel(m.cancel)
	defer stop()
	m.ctx = ctx

	if IsAlbumKey(albumNameOrKey) {
		return m.addToExistingAlbum(mediaKeys, paths, albumNameOrKey)
	}

	if key, ok := lookupSessionAlbum(m.api.account, albumNameOrKey, len(mediaKeys)); ok {
		keys, err := m.addToExistingAlbum(mediaKeys, paths, key)
		if err == nil {
			rememberSessionAlbum(m.api.account, albumNameOrKey, key, len(mediaKeys))
		}
		return keys, err
	}

	if key, ok := m.findExistingAlbumByName(albumNameOrKey, len(mediaKeys)); ok {
		keys, err := m.addToExistingAlbum(mediaKeys, paths, key)
		if err == nil {
			rememberSessionAlbum(m.api.account, albumNameOrKey, key, len(mediaKeys))
		}
		return keys, err
	}

	keys, err := m.createNewAlbum(mediaKeys, paths, albumNameOrKey)
	if err == nil && len(keys) == 1 {
		rememberSessionAlbum(m.api.account, albumNameOrKey, keys[0], len(mediaKeys))
	}
	return keys, err
}

func dedupeMediaKeys(keys, paths []string) ([]string, []string) {
	seen := make(map[string]struct{}, len(keys))
	outKeys := make([]string, 0, len(keys))
	outPaths := make([]string, 0, len(paths))
	for i, k := range keys {
		if _, dup := seen[k]; dup {
			continue
		}
		seen[k] = struct{}{}
		outKeys = append(outKeys, k)
		if i < len(paths) {
			outPaths = append(outPaths, paths[i])
		}
	}
	return outKeys, outPaths
}

func (m *AlbumManager) findExistingAlbumByName(name string, adding int) (string, bool) {
	albums, err := m.api.ListAlbums()
	if err != nil {
		m.app.GetLogger().Warn(fmt.Sprintf("could not look up existing albums, creating a new one: %v", err))
		return "", false
	}
	for _, a := range albums {
		if a.Key != "" && a.Name == name && int(a.ItemCount)+adding <= AlbumLimit {
			return a.Key, true
		}
	}
	return "", false
}

func (m *AlbumManager) emitAlbumStatus(event, albumName string, albumKeys []string, itemsAdded, totalItems int, complete bool, paths []string) {
	m.app.EmitEvent(event, AlbumStatus{
		AlbumName:  albumName,
		ItemsAdded: itemsAdded,
		TotalItems: totalItems,
		AlbumKeys:  albumKeys,
		IsComplete: complete,
		Paths:      paths,
	})
}

func (m *AlbumManager) addToExistingAlbum(mediaKeys []string, paths []string, albumKey string) ([]string, error) {
	totalItems := len(mediaKeys)
	itemsAdded := 0
	displayName := albumDisplayName(albumKey)

	for i := 0; i < len(mediaKeys); i += AlbumBatchSize {
		if m.isCancelled() {
			return []string{albumKey}, fmt.Errorf("album add cancelled (added %d/%d items)", itemsAdded, totalItems)
		}

		end := min(i+AlbumBatchSize, len(mediaKeys))
		batch := mediaKeys[i:end]
		batchPaths := pathsSlice(paths, i, end)

		if err := m.addMediaWithRetry(albumKey, batch); err != nil {
			return []string{albumKey}, fmt.Errorf("failed to add media to album (added %d/%d items): %w", itemsAdded, totalItems, err)
		}

		itemsAdded += len(batch)

		m.emitAlbumStatus(EventAlbumProgress, displayName, []string{albumKey}, itemsAdded, totalItems, false, batchPaths)
	}

	m.emitAlbumStatus(EventAlbumComplete, displayName, []string{albumKey}, itemsAdded, totalItems, true, nil)

	return []string{albumKey}, nil
}

func pathsSlice(paths []string, start, end int) []string {
	if start < 0 || end > len(paths) || start > end {
		return nil
	}
	return paths[start:end]
}

func (m *AlbumManager) withRetry(isRetryable func(error) bool, op func() error) error {
	retryConfig := DefaultRetryConfig()
	var lastErr error

	for attempt := 0; attempt <= retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			if m.isCancelled() {
				return fmt.Errorf("cancelled during retry")
			}
			if !m.sleep(CalculateBackoff(attempt-1, retryConfig)) {
				return fmt.Errorf("cancelled during retry")
			}
		}

		err := op()
		if err == nil {
			return nil
		}
		lastErr = err

		if !isRetryable(err) {
			return err
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", retryConfig.MaxRetries+1, lastErr)
}

func (m *AlbumManager) addMediaWithRetry(albumKey string, mediaKeys []string) error {
	return m.withRetry(isRetryableAlbumError, func() error {
		return m.api.AddMediaToAlbum(m.ctx, albumKey, mediaKeys)
	})
}

func (m *AlbumManager) sleep(d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-m.ctx.Done():
		return false
	}
}

func isRetryableAlbumError(err error) bool {
	if errors.Is(err, context.Canceled) {
		return false
	}
	var se *HTTPStatusError
	if errors.As(err, &se) {
		return ShouldRetryStatus(se.Code) || se.Code == 401
	}
	// Transport failures surface as *url.Error (which implements net.Error)
	// wrapped by the API client; any of those is worth retrying.
	var ne net.Error
	return errors.As(err, &ne)
}

func isSafeToRetryCreate(err error) bool {
	var se *HTTPStatusError
	if errors.As(err, &se) {
		return se.Code == 429 || se.Code == 503 || se.Code == 401
	}
	return false
}

func (m *AlbumManager) createNewAlbum(mediaKeys []string, paths []string, albumName string) ([]string, error) {
	var albumKeys []string
	totalItems := len(mediaKeys)
	itemsAdded := 0

	if len(mediaKeys) > uploadcore.AlbumLimit {
		m.app.GetLogger().Warn(fmt.Sprintf("%d items exceed the album limit of %d. They will be split into multiple albums.", len(mediaKeys), uploadcore.AlbumLimit))
	}

	var currentAlbumKey string
	offset := 0
	for _, chunk := range uploadcore.PlanAlbumBatches(mediaKeys, albumName) {
		if m.isCancelled() {
			return albumKeys, fmt.Errorf("album creation cancelled (added %d/%d items)", itemsAdded, totalItems)
		}

		end := offset + len(chunk.Batch)
		batchPaths := pathsSlice(paths, offset, end)

		var err error
		if chunk.IsFirstOfAlbum {
			currentAlbumKey, err = m.createAlbumWithRetry(chunk.Name, chunk.Batch)
			if err != nil {
				return albumKeys, fmt.Errorf("failed to create album '%s' (added %d/%d items): %w", chunk.Name, itemsAdded, totalItems, err)
			}
			albumKeys = append(albumKeys, currentAlbumKey)
		} else {
			err = m.addMediaWithRetry(currentAlbumKey, chunk.Batch)
			if err != nil {
				return albumKeys, fmt.Errorf("failed to add media to album '%s' (added %d/%d items): %w", chunk.Name, itemsAdded, totalItems, err)
			}
		}

		itemsAdded += len(chunk.Batch)
		offset = end

		m.emitAlbumStatus(EventAlbumProgress, chunk.Name, albumKeys, itemsAdded, totalItems, false, batchPaths)
	}

	m.emitAlbumStatus(EventAlbumComplete, albumName, albumKeys, itemsAdded, totalItems, true, nil)

	return albumKeys, nil
}

func (m *AlbumManager) createAlbumWithRetry(albumName string, mediaKeys []string) (string, error) {
	var albumKey string
	err := m.withRetry(isSafeToRetryCreate, func() error {
		var err error
		albumKey, err = m.api.CreateAlbum(m.ctx, albumName, mediaKeys)
		return err
	})
	if err != nil {
		return "", err
	}
	return albumKey, nil
}
