package configx

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sync"
)

type Auth struct {
	Username      string `json:"username"`
	PasswordHash  string `json:"password_hash"`
	SessionSecret string `json:"session_secret"`
}

type Photos struct {
	Credentials                   []string `json:"credentials"`
	Selected                      string   `json:"selected"`
	Proxy                         string   `json:"proxy"`
	UseQuota                      bool     `json:"useQuota"`
	Saver                         bool     `json:"saver"`
	Recursive                     bool     `json:"recursive"`
	ForceUpload                   bool     `json:"forceUpload"`
	UploadThreads                 int      `json:"uploadThreads"`
	DeleteFromHost                bool     `json:"deleteFromHost"`
	DisableUnsupportedFilesFilter bool     `json:"disableUnsupportedFilesFilter"`
	SetDateFromFilename           bool     `json:"setDateFromFilename"`
	ExcludePattern                string   `json:"excludePattern"`
}

const MaxUploadThreads = 32

type Config struct {
	Listen         string `json:"listen"`
	DataDir        string `json:"dataDir"`
	ThumbCacheMB   int    `json:"thumbCacheMB"`
	Auth           Auth   `json:"auth"`
	Photos         Photos `json:"photos"`
	LibrarySyncSec int    `json:"librarySyncSec"`
}

func Default() Config {
	return Config{
		Listen:         "0.0.0.0:8787",
		DataDir:        "data",
		ThumbCacheMB:   300,
		LibrarySyncSec: 120,
		Photos:         Photos{UploadThreads: 3},
	}
}

var (
	mu   sync.RWMutex
	path string
	cfg  Config

	bridge *Bridge
)

type Bridge struct {
	Get func() Photos
	Set func(Photos) error
}

func SetBridge(b *Bridge) {
	mu.Lock()
	defer mu.Unlock()
	bridge = b
}

func withBridge(c Config) Config {
	if bridge == nil {
		return c
	}
	c.Photos = bridge.Get()
	return c
}

func Load(p string) (Config, error) {
	mu.Lock()
	defer mu.Unlock()
	path = p

	raw, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		cfg = Default()
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	cfg = Default()
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config %s: %w", p, err)
	}
	if cfg.Listen == "" {
		cfg.Listen = Default().Listen
	}
	if cfg.Photos.UploadThreads < 1 {
		cfg.Photos.UploadThreads = 3
	}
	if cfg.LibrarySyncSec == 0 {
		cfg.LibrarySyncSec = Default().LibrarySyncSec
	}
	if cfg.LibrarySyncSec > 0 && cfg.LibrarySyncSec < 15 {
		cfg.LibrarySyncSec = 15
	}
	if cfg.Photos.Selected != "" {
		if dec, err := url.QueryUnescape(cfg.Photos.Selected); err == nil && dec != cfg.Photos.Selected {
			cfg.Photos.Selected = dec
			_ = saveLocked()
		}
	}
	return cfg, nil
}

func Get() Config {
	mu.RLock()
	defer mu.RUnlock()
	c := withBridge(cfg)
	if len(c.Photos.Credentials) > 0 {
		c.Photos.Credentials = append([]string(nil), c.Photos.Credentials...)
	}
	return c
}

func Update(fn func(*Config)) error {
	mu.Lock()
	defer mu.Unlock()
	if bridge == nil {
		fn(&cfg)
		return saveLocked()
	}
	cfg.Photos = bridge.Get()
	fn(&cfg)
	photos := cfg.Photos
	cfg.Photos = Photos{}
	if err := bridge.Set(photos); err != nil {
		return err
	}
	return saveLocked()
}

func saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func Save() error {
	mu.Lock()
	defer mu.Unlock()
	return saveLocked()
}
