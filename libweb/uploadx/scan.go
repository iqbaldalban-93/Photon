package uploadx

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"app/internal/mediatypes"
	"app/libweb/configx"
)

// Filesystem preflight: directory scanning and the supported-format filter
// with same-file de-duplication (canonical path + inode identity).

func scanDirectory(path string, recursive bool, excludePattern string, cancelled func() bool, onScanned func()) ([]string, error) {
	var files []string
	if cancelled() {
		return nil, contextCanceled
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if cancelled() {
			return nil, contextCanceled
		}
		fullPath := filepath.Join(path, entry.Name())
		if entry.IsDir() {
			if excludePattern != "" && entry.Name() == excludePattern {
				continue
			}
			if recursive {
				subFiles, err := scanDirectory(fullPath, recursive, excludePattern, cancelled, onScanned)
				if err != nil {
					if errors.Is(err, contextCanceled) {
						return nil, err
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

func filterFiles(paths []string, cancelled func() bool, onScanned func()) ([]string, map[string]fileMeta, error) {
	var supported []string
	sizes := make(map[string]fileMeta)
	type seenFile struct {
		canonical string
		info      os.FileInfo
	}
	seen := make(map[string][]seenFile)
	cfg := configx.Get()

	appendFile := func(path string, info os.FileInfo) {
		if !cfg.Photos.DisableUnsupportedFilesFilter && !mediatypes.IsSupportedByGooglePhotos(path) {
			return
		}
		canonical, err := filepath.Abs(path)
		if err != nil {
			canonical = filepath.Clean(path)
		}
		if resolved, err := filepath.EvalSymlinks(canonical); err == nil {
			canonical = resolved
		}
		canonical = filepath.Clean(canonical)
		bucket := canonical
		if runtime.GOOS == "windows" {
			bucket = strings.ToLower(bucket)
		}
		if info == nil {
			info, _ = os.Stat(path)
		}
		for _, s := range seen[bucket] {
			if canonical == s.canonical || (info != nil && s.info != nil && os.SameFile(info, s.info)) {
				return
			}
		}
		seen[bucket] = append(seen[bucket], seenFile{canonical: canonical, info: info})
		supported = append(supported, path)
		if info != nil {
			sizes[path] = fileMeta{Size: info.Size(), ModTime: info.ModTime()}
		}
	}

	for _, path := range paths {
		if cancelled() {
			return nil, nil, contextCanceled
		}
		info, err := os.Stat(path)
		if err != nil {
			return nil, nil, fmt.Errorf("error accessing path %s: %w", path, err)
		}
		if info.IsDir() {
			files, err := scanDirectory(path, cfg.Photos.Recursive, cfg.Photos.ExcludePattern, cancelled, onScanned)
			if err != nil {
				return nil, nil, fmt.Errorf("error scanning directory %s: %w", path, err)
			}
			for _, f := range files {
				if cancelled() {
					return nil, nil, contextCanceled
				}
				appendFile(f, nil)
			}
		} else {
			appendFile(path, info)
		}
	}
	return supported, sizes, nil
}
