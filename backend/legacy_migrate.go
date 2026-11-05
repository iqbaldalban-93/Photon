package backend

import (
	"log/slog"
	"os"
	"path/filepath"
)

const (
	legacyAppName    = "gotohp"
	legacyConfigName = legacyAppName + ".config"
)

func migrateLegacyFile(oldPath, newPath string) {
	if _, err := os.Stat(newPath); err == nil {
		return
	}
	if _, err := os.Stat(oldPath); err != nil {
		return
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		slog.Warn("could not migrate old config file", "from", oldPath, "to", newPath, "error", err)
		return
	}
	slog.Info("migrated old config file", "from", oldPath, "to", newPath)
}

func migrateLegacyConfigDir(oldDir, newDir, newConfigName string) {
	if _, err := os.Stat(newDir); err != nil {
		if _, oldErr := os.Stat(oldDir); oldErr == nil {
			if err := os.Rename(oldDir, newDir); err != nil {
				slog.Warn("could not migrate old data folder", "from", oldDir, "to", newDir, "error", err)
				return
			}
			slog.Info("migrated old data folder", "from", oldDir, "to", newDir)
		}
	} else if _, err := os.Stat(filepath.Join(newDir, newConfigName)); err != nil {
		mergeLegacyDir(oldDir, newDir)
	}
	migrateLegacyFile(filepath.Join(newDir, legacyConfigName), filepath.Join(newDir, newConfigName))
}

func mergeLegacyDir(oldDir, newDir string) {
	entries, err := os.ReadDir(oldDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		dst := filepath.Join(newDir, e.Name())
		if _, err := os.Stat(dst); err == nil {
			continue
		}
		if err := os.Rename(filepath.Join(oldDir, e.Name()), dst); err != nil {
			slog.Warn("could not migrate old data file", "name", e.Name(), "error", err)
		}
	}
	_ = os.Remove(oldDir)
}

const portableDataDirName = "data"

var portableDataFiles = []string{
	"history.db",
	"history.db-wal",
	"history.db-shm",
	"library-web.config",
	"resume_batch.json",
	"library-data",
}

func migratePortableData(exeDir, dataDir string) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		slog.Warn("could not create data folder", "path", dataDir, "error", err)
		return
	}
	migrateLegacyFile(filepath.Join(exeDir, "Photon.config"), filepath.Join(dataDir, "Photon.config"))
	if _, err := os.Stat(filepath.Join(dataDir, "Photon.config")); err != nil {
		return
	}
	for _, name := range portableDataFiles {
		migrateLegacyFile(filepath.Join(exeDir, name), filepath.Join(dataDir, name))
	}
}
