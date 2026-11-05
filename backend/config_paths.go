package backend

import (
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

// Config file location: portable data dir next to the executable (with
// legacy-layout migration), falling back to the per-user config directory.

var (
	pathMu sync.Mutex

	ConfigPath string
)

func determineConfigPath() {
	pathMu.Lock()
	defer pathMu.Unlock()
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		dataDir := filepath.Join(exeDir, portableDataDirName)
		portableConfigPath := filepath.Join(dataDir, "Photon.config")
		oldPortableConfigPath := filepath.Join(exeDir, "Photon.config")
		migrateLegacyFile(filepath.Join(exeDir, legacyConfigName), oldPortableConfigPath)

		if _, err := os.Stat(portableConfigPath); err != nil {
			if _, oldErr := os.Stat(oldPortableConfigPath); oldErr == nil {
				migratePortableData(exeDir, dataDir)
				if _, err := os.Stat(portableConfigPath); err != nil {
					ConfigPath = oldPortableConfigPath
					return
				}
			}
		}

		if _, err := os.Stat(portableConfigPath); err == nil {
			ConfigPath = portableConfigPath
			return
		}
	}

	userConfigDir := filepath.Join(getUserConfigDir(), "Photon")
	migrateLegacyConfigDir(filepath.Join(getUserConfigDir(), legacyAppName), userConfigDir, "Photon.config")
	ConfigPath = filepath.Join(userConfigDir, "Photon.config")
}

func getUserConfigDir() string {
	dirname, err := os.UserConfigDir()
	if err != nil {
		slog.Error("cannot determine user config directory", "error", err)
		dirname = "."
	}
	return dirname
}
