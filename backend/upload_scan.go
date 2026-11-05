package backend

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"app/internal/mediatypes"
)

// Filesystem preflight: directory scanning and the Google-Photos support
// filter that decides which paths become upload work.

func looksLikeSupportedMedia(path string) bool {
	return mediatypes.IsSupportedByGooglePhotos(path) && contentMatchesExtension(path)
}

func scanDirectoryForFiles(path string, recursive bool, excludePattern string, cancelled func() bool, onScanned func(), onSkip func(dir string, err error)) ([]string, error) {
	var files []string

	if cancelled != nil && cancelled() {
		return nil, context.Canceled
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if cancelled != nil && cancelled() {
			return nil, context.Canceled
		}
		fullPath := filepath.Join(path, entry.Name())
		if entry.IsDir() {
			if excludePattern != "" && entry.Name() == excludePattern {
				continue
			}
			if recursive {
				subFiles, err := scanDirectoryForFiles(fullPath, recursive, excludePattern, cancelled, onScanned, onSkip)
				if err != nil {
					if errors.Is(err, context.Canceled) {
						return nil, err
					}
					if onSkip != nil {
						onSkip(fullPath, err)
					}
					continue
				}
				files = append(files, subFiles...)
			}
		} else {
			files = append(files, fullPath)
			if onScanned != nil {
				onScanned()
			}
		}
	}

	return files, nil
}

func filterGooglePhotosFilesWithCancel(cfg Config, paths []string, cancelled func() bool, onScanned func(), onSkip func(dir string, err error)) ([]string, map[string]fileMeta, error) {
	var supportedFiles []string
	sizes := make(map[string]fileMeta)
	seenFiles := make(map[string]bool)
	appendFile := func(path string, info os.FileInfo) {
		if strings.HasPrefix(filepath.Base(path), "._") {
			return
		}
		if !cfg.DisableUnsupportedFilesFilter && !looksLikeSupportedMedia(path) {
			return
		}
		key := canonicalUploadPath(path)
		if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
			key = strings.ToLower(key)
		}
		if seenFiles[key] {
			return
		}
		seenFiles[key] = true
		if info == nil {
			info, _ = os.Stat(path)
		}
		supportedFiles = append(supportedFiles, path)
		if info != nil {
			sizes[path] = fileMeta{Size: info.Size(), ModTime: info.ModTime()}
		}
	}

	for _, path := range paths {
		if cancelled != nil && cancelled() {
			return nil, nil, context.Canceled
		}
		fileInfo, err := os.Stat(path)
		if err != nil {
			return nil, nil, fmt.Errorf("error accessing path %s: %w", path, err)
		}

		if fileInfo.IsDir() {
			files, err := scanDirectoryForFiles(path, cfg.Recursive, cfg.ExcludePattern, cancelled, onScanned, onSkip)
			if err != nil {
				return nil, nil, fmt.Errorf("error scanning directory %s: %w", path, err)
			}

			for _, file := range files {
				if cancelled != nil && cancelled() {
					return nil, nil, context.Canceled
				}
				appendFile(file, nil)
			}
		} else {
			appendFile(path, fileInfo)
		}
	}

	return supportedFiles, sizes, nil
}

func canonicalUploadPath(path string) string {
	canonicalPath, err := filepath.Abs(path)
	if err != nil {
		canonicalPath = filepath.Clean(path)
	}
	if resolved, err := filepath.EvalSymlinks(canonicalPath); err == nil {
		canonicalPath = resolved
	}
	return filepath.Clean(canonicalPath)
}

func resolveUploadTimestamp(filePath string, mtime int64, cfg Config) int64 {
	ts := mtime
	exifFound := false
	if cfg.UseExifDate {
		if t, ok := exifCaptureDate(filePath); ok {
			ts = t.Unix()
			exifFound = true
		}
	}
	if !exifFound && cfg.SetDateFromFilename {
		if t, ok := parseTimestampFromFilename(filePath); ok {
			ts = t.Unix()
		}
	}
	return ts
}
