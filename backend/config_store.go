package backend

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"app/internal/fsx"
	"app/internal/proxyurl"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
)

// Global config store: a process-wide snapshot behind an atomic pointer,
// mutated through setConfig which persists after every change.

var (
	configMu sync.Mutex

	configPtr atomic.Pointer[Config]

	configLoaded atomic.Bool

	configReadErr error

	DefaultConfig = Config{
		UploadThreads:    3,
		NotifyOnComplete: true,
		AutoRetryFailed:  true,
		UseExifDate:      true,
		LibraryWebPort:   defaultLibraryWebPort,
		LibraryWebLAN:    false,
	}
)

func CurrentConfig() Config {
	cfg := configPtr.Load()
	if cfg == nil {
		return DefaultConfig
	}
	c := *cfg
	c.Credentials = slices.Clone(cfg.Credentials)
	c.SavedAlbums = slices.Clone(cfg.SavedAlbums)
	c.WatchedFolders = slices.Clone(cfg.WatchedFolders)
	return c
}

func setConfig(mutate func(*Config)) error {
	configMu.Lock()
	defer configMu.Unlock()

	if configReadErr != nil {
		return fmt.Errorf("config file could not be read, refusing to overwrite it: %w", configReadErr)
	}
	c := CurrentConfig()
	mutate(&c)
	if ConfigPath == "" {
		determineConfigPath()
	}
	configPtr.Store(&c)
	return writeConfigFile(c)
}

func saveSet(mutate func(*Config)) {
	if err := setConfig(mutate); err != nil {
		slog.Error("failed to persist config", "error", err)
	}
}

func (g *ConfigManager) SetProxy(proxy string) error {
	if _, err := proxyurl.Parse(proxy); err != nil {
		return err
	}
	proxy = strings.TrimSpace(proxy)
	return setConfig(func(c *Config) { c.Proxy = proxy })
}

func (g *ConfigManager) SetSelected(email string) {
	saveSet(func(c *Config) { c.Selected = email })
	ReconcileLibraryAccount()
}

func (g *ConfigManager) SetUseQuota(useQuota bool) {
	saveSet(func(c *Config) { c.UseQuota = useQuota })
}

func (g *ConfigManager) SetSaver(saver bool) {
	saveSet(func(c *Config) { c.Saver = saver })
}

func (g *ConfigManager) SetRecursive(recursive bool) {
	saveSet(func(c *Config) { c.Recursive = recursive })
}

func (g *ConfigManager) SetForceUpload(forceUpload bool) {
	saveSet(func(c *Config) { c.ForceUpload = forceUpload })
}

func (g *ConfigManager) SetDeleteFromHost(deleteFromHost bool) {
	saveSet(func(c *Config) { c.DeleteFromHost = deleteFromHost })
}

func (g *ConfigManager) SetDisableUnsupportedFilesFilter(disableUnsupportedFilesFilter bool) {
	saveSet(func(c *Config) { c.DisableUnsupportedFilesFilter = disableUnsupportedFilesFilter })
}

const maxUploadThreads = 32

func (g *ConfigManager) SetUploadThreads(uploadThreads int) {
	if uploadThreads < 1 {
		return
	}
	uploadThreads = min(uploadThreads, maxUploadThreads)
	saveSet(func(c *Config) { c.UploadThreads = uploadThreads })
}

func (g *ConfigManager) SetSetDateFromFilename(v bool) {
	saveSet(func(c *Config) { c.SetDateFromFilename = v })
}

func (g *ConfigManager) SetNotifyOnComplete(v bool) {
	saveSet(func(c *Config) { c.NotifyOnComplete = v })
}

func (g *ConfigManager) SetAutoRetryFailed(v bool) {
	saveSet(func(c *Config) { c.AutoRetryFailed = v })
}

func (g *ConfigManager) SetUseExifDate(v bool) {
	saveSet(func(c *Config) { c.UseExifDate = v })
}

func (g *ConfigManager) GetUploadHistoryCount() int64 {
	return countUploadHistory()
}

func (g *ConfigManager) ClearUploadHistory() error {
	return clearUploadHistory()
}

var OnCloseToTrayChanged func(enabled bool)

func (g *ConfigManager) SetCloseToTray(closeToTray bool) {
	saveSet(func(c *Config) { c.CloseToTray = closeToTray })
	if OnCloseToTrayChanged != nil {
		OnCloseToTrayChanged(closeToTray)
	}
}

func (g *ConfigManager) SaveAlbum(name, key, account string) error {
	name = strings.TrimSpace(name)
	key = strings.TrimSpace(key)
	account = strings.TrimSpace(account)
	if account == "" {
		return fmt.Errorf("no account selected")
	}
	if name == "" {
		return fmt.Errorf("album name cannot be empty")
	}
	if idx := strings.Index(key, "AF1Qip"); idx >= 0 {
		key = key[idx:]
	}
	if !IsAlbumKey(key) {
		return fmt.Errorf("invalid album key: %q", key)
	}

	cfg := CurrentConfig()
	for _, saved := range cfg.SavedAlbums {
		if saved.Key == key {
			return setConfig(func(c *Config) {
				for i := range c.SavedAlbums {
					if c.SavedAlbums[i].Key == key {
						c.SavedAlbums[i].Account = account
						c.SavedAlbums[i].Name = name
						return
					}
				}
			})
		}
	}
	return setConfig(func(c *Config) {
		c.SavedAlbums = append(c.SavedAlbums, SavedAlbum{Account: account, Name: name, Key: key})
	})
}

func (g *ConfigManager) RemoveSavedAlbum(key string) error {
	key = strings.TrimSpace(key)
	return setConfig(func(c *Config) {
		updated := make([]SavedAlbum, 0, len(c.SavedAlbums))
		for _, saved := range c.SavedAlbums {
			if saved.Key != key {
				updated = append(updated, saved)
			}
		}
		c.SavedAlbums = updated
	})
}

func (g *ConfigManager) GetConfig() Config {
	if !configLoaded.Load() {
		_ = LoadConfig()
	}
	cfg := CurrentConfig()
	cfg.Credentials = redactCredentials(cfg.Credentials)
	return cfg
}

func LoadConfig() error {
	determineConfigPath()

	var data []byte
	var readErr error
	for attempt := 0; attempt < 3; attempt++ {
		data, readErr = os.ReadFile(ConfigPath)
		if readErr == nil || errors.Is(readErr, os.ErrNotExist) {
			readErr = nil
			break
		}
		time.Sleep(150 * time.Millisecond)
	}

	configMu.Lock()
	defer configMu.Unlock()

	var cfg Config
	switch {
	case readErr != nil:
		slog.Error("config file exists but cannot be read; using defaults without saving", "path", ConfigPath, "error", readErr)
		cfg = DefaultConfig
	case len(data) == 0:
		cfg = DefaultConfig
	default:
		cfg = loadAppConfig()
	}
	configReadErr = readErr
	configPtr.Store(&cfg)
	configLoaded.Store(true)
	return readErr
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	return fsx.WriteFileAtomic(path, data, perm)
}

func writeConfigFile(cfg Config) error {
	k := koanf.New(".")
	if err := k.Load(structs.Provider(cfg, "koanf"), nil); err != nil {
		return err
	}
	b, err := k.Marshal(yaml.Parser())
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(ConfigPath), 0o755); err != nil {
		return err
	}
	return writeFileAtomic(ConfigPath, b, 0o600)
}

func loadAppConfig() Config {
	c := DefaultConfig
	k := koanf.New(".")
	if err := k.Load(file.Provider(ConfigPath), yaml.Parser()); err != nil {
		quarantineCorruptConfig(err)
		return DefaultConfig
	}
	if err := k.Unmarshal("", &c); err != nil {
		quarantineCorruptConfig(err)
		return DefaultConfig
	}

	if c.UploadThreads < 1 {
		c.UploadThreads = DefaultConfig.UploadThreads
	}
	c.UploadThreads = min(c.UploadThreads, maxUploadThreads)
	if c.LibraryWebPort < 1 || c.LibraryWebPort > 65535 {
		c.LibraryWebPort = DefaultConfig.LibraryWebPort
	}

	return c
}

func quarantineCorruptConfig(cause error) {
	slog.Error("app config is corrupt; using defaults and preserving the original", "path", ConfigPath, "error", cause)
	backup := ConfigPath + ".corrupt-" + time.Now().Format("20060102-150405")
	if err := os.Rename(ConfigPath, backup); err != nil {
		slog.Error("could not preserve corrupt config file", "error", err)
		return
	}
	slog.Info("corrupt config file preserved for recovery", "backup", backup)
}
