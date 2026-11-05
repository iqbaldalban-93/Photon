package backend

import (
	"fmt"
	"log/slog"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"app/internal/credentials"
	"app/libweb"
	"app/libweb/configx"
	"app/libweb/library"
)

const defaultLibraryWebPort = 8787

type LibraryWebStatus struct {
	Enabled bool     `json:"enabled"`
	Running bool     `json:"running"`
	Port    int      `json:"port"`
	LAN     bool     `json:"lan"`
	URLs    []string `json:"urls"`
	Error   string   `json:"error"`
}

var libraryWeb struct {
	mu   sync.Mutex
	inst *libweb.Instance
	err  string
	port int
	lan  bool
}

func photosBridge() *configx.Bridge {
	return &configx.Bridge{
		Get: func() configx.Photos {
			c := CurrentConfig()
			return configx.Photos{
				Credentials:                   c.Credentials,
				Selected:                      c.Selected,
				Proxy:                         c.Proxy,
				UseQuota:                      c.UseQuota,
				Saver:                         c.Saver,
				Recursive:                     c.Recursive,
				ForceUpload:                   c.ForceUpload,
				UploadThreads:                 c.UploadThreads,
				DeleteFromHost:                c.DeleteFromHost,
				DisableUnsupportedFilesFilter: c.DisableUnsupportedFilesFilter,
				SetDateFromFilename:           c.SetDateFromFilename,
				ExcludePattern:                c.ExcludePattern,
			}
		},
		Set: func(p configx.Photos) error {
			return setConfig(func(c *Config) {
				c.Credentials = p.Credentials
				c.Selected = p.Selected
				c.Proxy = p.Proxy
				c.UseQuota = p.UseQuota
				c.Saver = p.Saver
				c.Recursive = p.Recursive
				c.ForceUpload = p.ForceUpload
				if p.UploadThreads >= 1 {
					c.UploadThreads = min(p.UploadThreads, maxUploadThreads)
				}
				c.DeleteFromHost = p.DeleteFromHost
				c.DisableUnsupportedFilesFilter = p.DisableUnsupportedFilesFilter
				c.SetDateFromFilename = p.SetDateFromFilename
				c.ExcludePattern = p.ExcludePattern
			})
		},
	}
}

func libraryWebDirs() (configFile, dataDir string) {
	if ConfigPath == "" {
		determineConfigPath()
	}
	dir := filepath.Dir(ConfigPath)
	return filepath.Join(dir, "library-web.config"), filepath.Join(dir, "library-data")
}

func applyLibraryWebLocked() {
	if libraryWeb.inst != nil {
		libraryWeb.inst.Stop()
		libraryWeb.inst = nil
	}
	libraryWeb.err = ""

	cfg := CurrentConfig()
	libraryWeb.port, libraryWeb.lan = cfg.LibraryWebPort, cfg.LibraryWebLAN
	if !cfg.LibraryWebEnabled {
		return
	}

	host := "127.0.0.1"
	if cfg.LibraryWebLAN {
		host = "0.0.0.0"
	}
	configFile, dataDir := libraryWebDirs()
	inst, err := libweb.Start(slog.Default(), libweb.Options{
		ConfigPath: configFile,
		DataDir:    dataDir,
		Listen:     net.JoinHostPort(host, strconv.Itoa(cfg.LibraryWebPort)),
		Photos:     photosBridge(),
		Version:    "embedded",

		OnAccountApplied: notifyAccountChanged,
	})
	if err != nil {
		libraryWeb.err = fmt.Sprintf("Couldn't start on port %d: %v", cfg.LibraryWebPort, err)
		slog.Warn("library web failed to start", "port", cfg.LibraryWebPort, "error", err)
		return
	}
	libraryWeb.inst = inst
	slog.Info("library web started", "port", cfg.LibraryWebPort, "lan", cfg.LibraryWebLAN)
}

func libraryWebStatusLocked() LibraryWebStatus {
	cfg := CurrentConfig()
	running := libraryWeb.inst != nil
	st := LibraryWebStatus{
		Enabled: cfg.LibraryWebEnabled,
		Running: running,
		Port:    cfg.LibraryWebPort,
		LAN:     cfg.LibraryWebLAN,
		URLs:    []string{},
		Error:   libraryWeb.err,
	}
	if running {
		st.Port, st.LAN = libraryWeb.port, libraryWeb.lan
		st.URLs = libweb.URLs(libraryWeb.port, libraryWeb.lan)
	}
	return st
}

func StartLibraryWebIfEnabled() {
	libraryWeb.mu.Lock()
	defer libraryWeb.mu.Unlock()
	if CurrentConfig().LibraryWebEnabled {
		applyLibraryWebLocked()
	}
}

func StopLibraryWeb() {
	libraryWeb.mu.Lock()
	defer libraryWeb.mu.Unlock()
	if libraryWeb.inst != nil {
		libraryWeb.inst.Stop()
		libraryWeb.inst = nil
	}
}

func (g *ConfigManager) GetLibraryWebStatus() LibraryWebStatus {
	libraryWeb.mu.Lock()
	defer libraryWeb.mu.Unlock()
	return libraryWebStatusLocked()
}

func (g *ConfigManager) SetLibraryWebEnabled(enabled bool) LibraryWebStatus {
	libraryWeb.mu.Lock()
	defer libraryWeb.mu.Unlock()
	saveSet(func(c *Config) { c.LibraryWebEnabled = enabled })
	applyLibraryWebLocked()
	return libraryWebStatusLocked()
}

func (g *ConfigManager) SetLibraryWebPort(port int) (LibraryWebStatus, error) {
	if port < 1024 || port > 65535 {
		return LibraryWebStatus{}, fmt.Errorf("port must be between 1024 and 65535")
	}
	libraryWeb.mu.Lock()
	defer libraryWeb.mu.Unlock()
	saveSet(func(c *Config) { c.LibraryWebPort = port })
	applyLibraryWebLocked()
	return libraryWebStatusLocked(), nil
}

func (g *ConfigManager) SetLibraryWebLAN(lan bool) LibraryWebStatus {
	libraryWeb.mu.Lock()
	defer libraryWeb.mu.Unlock()
	saveSet(func(c *Config) { c.LibraryWebLAN = lan })
	applyLibraryWebLocked()
	return libraryWebStatusLocked()
}

func libraryInstance() *libweb.Instance {
	libraryWeb.mu.Lock()
	defer libraryWeb.mu.Unlock()
	return libraryWeb.inst
}

func LibraryWebURL() string {
	libraryWeb.mu.Lock()
	defer libraryWeb.mu.Unlock()
	if libraryWeb.inst == nil {
		return ""
	}
	return "http://localhost:" + strconv.Itoa(libraryWeb.port)
}

func TriggerLibrarySync() bool {
	if inst := libraryInstance(); inst != nil {
		return inst.TriggerSync()
	}
	return false
}

func SetWebUploadsPaused(paused bool) {
	if inst := libraryInstance(); inst != nil {
		if paused {
			inst.PauseWebUploads()
		} else {
			inst.ResumeWebUploads()
		}
	}
}

func CancelWebUploads() {
	if inst := libraryInstance(); inst != nil {
		inst.CancelWebUploads()
	}
}

var ErrWebUploadsRunning = libweb.ErrUploadsRunning

var OnAccountChanged func(email string)

func notifyAccountChanged(email string) {
	if f := OnAccountChanged; f != nil {
		f(email)
	}
}

func credentialEmail(cred string) string {
	return credentials.Email(cred)
}

func AccountEmails(cfg Config) []string {
	out := make([]string, 0, len(cfg.Credentials))
	for _, c := range cfg.Credentials {
		if e := credentialEmail(c); e != "" {
			out = append(out, e)
		}
	}
	return out
}

func sameAccount(a, b string) bool {
	return library.AccountKey(a) == library.AccountKey(b)
}

func SwitchAccount(email string, force bool) error {
	cfg := CurrentConfig()
	known := ""
	for _, e := range AccountEmails(cfg) {
		if sameAccount(e, email) {
			known = e
			break
		}
	}
	if known == "" {
		return fmt.Errorf("no account %q", strings.TrimSpace(email))
	}
	if inst := libraryInstance(); inst != nil {

		return inst.SwitchAccount(known, force)
	}
	if sameAccount(cfg.Selected, known) {
		return nil
	}
	if err := setConfig(func(c *Config) { c.Selected = known }); err != nil {
		return err
	}
	notifyAccountChanged(known)
	return nil
}

func ReconcileLibraryAccount() {
	if inst := libraryInstance(); inst != nil {
		go func() {
			if err := inst.ApplyAccount(); err != nil {
				slog.Warn("library could not follow the selected account", "error", err)
			}
		}()
		return
	}
	notifyAccountChanged(CurrentConfig().Selected)
}
