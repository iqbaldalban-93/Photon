package libweb

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"app/libweb/configx"
	"app/libweb/events"
	"app/libweb/gphotos"
	"app/libweb/library"
	"app/libweb/proxy"
	"app/libweb/server"
	"app/libweb/uploadx"
)

//go:embed all:web
var webFS embed.FS

const mediaUserAgent = "com.google.android.apps.photos/49029607 (Linux; U; Android 9; Pixel XL; Build/PQ2A.190205.001) (gzip)"

var ErrUploadsRunning = server.ErrUploadsRunning

type Options struct {
	ConfigPath string
	DataDir    string
	Listen     string
	Photos     *configx.Bridge
	Version    string

	OnAccountApplied func(email string)
}

type Instance struct {
	httpServer *http.Server
	index      *library.Index
	syncMgr    *library.SyncManager
	uploads    *uploadx.Manager
	thumbs     *proxy.ThumbnailCache
	hub        *events.Hub
	log        *slog.Logger
	dataDir    string
	stopTick   chan struct{}
	stopOnce   sync.Once

	switchMu  sync.Mutex
	active    string
	epoch     atomic.Uint64
	onApplied func(email string)
}

func Start(log *slog.Logger, opt Options) (*Instance, error) {
	if opt.DataDir == "" {
		return nil, errors.New("libweb: data directory is required")
	}

	configx.SetBridge(nil)
	if _, err := configx.Load(opt.ConfigPath); err != nil {
		return nil, fmt.Errorf("load library settings: %w", err)
	}
	if configx.Get().DataDir != opt.DataDir {
		if err := configx.Update(func(c *configx.Config) { c.DataDir = opt.DataDir }); err != nil {
			return nil, fmt.Errorf("save library settings: %w", err)
		}
	}
	configx.SetBridge(opt.Photos)
	server.SetupLocalOnly = true

	if err := os.MkdirAll(opt.DataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}

	ln, err := net.Listen("tcp", opt.Listen)
	if err != nil {
		return nil, err
	}

	selected := configx.Get().Photos.Selected
	migrateLegacyLibrary(log, opt.DataDir, selected)
	acctDir := library.AccountDir(opt.DataDir, selected)

	index, err := library.NewIndexAt(filepath.Join(acctDir, "library.json"))
	if err != nil {
		_ = ln.Close()
		return nil, fmt.Errorf("open library index: %w", err)
	}
	watched, err := library.NewWatchedStore(opt.DataDir, func() string { return configx.Get().Photos.Selected })
	if err != nil {
		_ = ln.Close()
		return nil, fmt.Errorf("open watched videos store: %w", err)
	}
	index.SetWatchedStore(watched)
	syncMgr := library.NewSyncManager(index, log)

	uploads := uploadx.NewManager(acctDir, func() {
		if configx.Get().Photos.Selected != "" {
			syncMgr.Trigger(false)
		}
	})

	bearer := func() (string, error) {
		api, err := gphotos.Shared()
		if err != nil {
			return "", err
		}
		return api.BearerToken()
	}

	cfg := configx.Get()
	thumbMB := cfg.ThumbCacheMB
	if thumbMB <= 0 {
		thumbMB = 300
	}
	srv, err := server.New(
		log, webFS, index, syncMgr, uploads,
		proxy.NewMediaClient(), bearer, mediaUserAgent,
		filepath.Join(acctDir, "thumbcache"), thumbMB,
	)
	if err != nil {
		_ = ln.Close()
		return nil, fmt.Errorf("build server: %w", err)
	}
	srv.Version = opt.Version

	inst := &Instance{
		httpServer: &http.Server{
			Handler:           srv.Handler(),
			ReadHeaderTimeout: 30 * time.Second,
			IdleTimeout:       2 * time.Minute,
			MaxHeaderBytes:    64 << 10,
		},
		index:     index,
		syncMgr:   syncMgr,
		uploads:   uploads,
		thumbs:    srv.Thumbs(),
		hub:       events.NewHub(),
		log:       log,
		dataDir:   opt.DataDir,
		stopTick:  make(chan struct{}),
		active:    library.AccountKey(selected),
		onApplied: opt.OnAccountApplied,
	}
	srv.Events = inst.hub
	srv.AccountEpoch = inst.epoch.Load
	srv.SwitchAccount = inst.SwitchAccount
	srv.AccountsChanged = func() { go func() { _ = inst.ApplyAccount() }() }

	go inst.autoSyncLoop()
	go inst.publishLoop()

	go func() {
		if err := inst.httpServer.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("library server stopped unexpectedly", "err", err)
		}
	}()
	return inst, nil
}

func (i *Instance) SyncStatus() library.SyncStatus { return i.syncMgr.Status() }

func (i *Instance) TriggerSync() bool { return i.syncMgr.Trigger(false) }

func (i *Instance) LibraryStats() map[string]any { return i.index.Stats() }

func (i *Instance) WebUploadProgress() (running, paused bool, total, done, failed int) {
	return i.uploads.Progress()
}

func (i *Instance) WebUploadSpeed() float64 { return i.uploads.Speed() }

func (i *Instance) PauseWebUploads()  { i.uploads.Pause() }
func (i *Instance) ResumeWebUploads() { i.uploads.Resume() }
func (i *Instance) CancelWebUploads() { i.uploads.Cancel() }

func (i *Instance) Stop() {
	if i == nil {
		return
	}
	i.stopOnce.Do(func() {
		close(i.stopTick)
		i.syncMgr.Cancel()
		i.uploads.Cancel()

		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		defer cancel()
		if err := i.httpServer.Shutdown(ctx); err != nil {
			_ = i.httpServer.Close()
		}
		_ = i.index.Save()
	})
}

func (i *Instance) autoSyncLoop() {
	var last time.Time
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-i.stopTick:
			return
		case <-t.C:
		}

		if err := i.ApplyAccount(); err != nil {
			i.log.Warn("library account switch failed, will retry", "err", err)
			continue
		}
		if configx.Get().Photos.Selected == "" {
			continue
		}
		sec := configx.Get().LibrarySyncSec
		if sec == 0 {
			sec = 120
		}
		if sec < 0 {
			continue
		}
		if !last.IsZero() && time.Since(last) < time.Duration(sec)*time.Second {
			continue
		}
		if i.syncMgr.Trigger(false) {
			last = time.Now()
			i.log.Debug("auto library sync triggered", "intervalSec", sec)
		}
	}
}

func (i *Instance) publishLoop() {
	t := time.NewTicker(600 * time.Millisecond)
	defer t.Stop()
	var last string
	for {
		select {
		case <-i.stopTick:
			return
		case <-t.C:
		}
		if i.hub.Len() == 0 {
			last = ""
			continue
		}
		payload := map[string]any{
			"sync":    i.syncMgr.Status(),
			"version": i.index.Version(),
			"epoch":   i.epoch.Load(),
		}
		b, err := json.Marshal(payload)
		if err != nil || string(b) == last {
			continue
		}
		last = string(b)
		i.hub.Publish("status", payload)
	}
}

func (i *Instance) SwitchAccount(email string, force bool) error {
	if library.AccountKey(email) != library.AccountKey(configx.Get().Photos.Selected) && !force {
		if running, _, _, _, _ := i.uploads.Progress(); running {
			return ErrUploadsRunning
		}
	}
	if err := configx.Update(func(c *configx.Config) { c.Photos.Selected = email }); err != nil {
		return err
	}
	return i.ApplyAccount()
}

func (i *Instance) ApplyAccount() error {
	i.switchMu.Lock()
	defer i.switchMu.Unlock()

	selected := configx.Get().Photos.Selected
	key := library.AccountKey(selected)
	if key == i.active {
		return nil
	}

	i.syncMgr.Block()
	if !i.syncMgr.StopAndWait(10 * time.Second) {
		i.syncMgr.Unblock()
		return errors.New("the running library sync did not stop in time")
	}
	i.uploads.Cancel()
	for n := 0; n < 120; n++ {
		if running, _, _, _, _ := i.uploads.Progress(); !running {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}

	dir := library.AccountDir(i.dataDir, selected)
	if err := i.index.SwitchTo(filepath.Join(dir, "library.json")); err != nil {
		i.syncMgr.Unblock()
		return fmt.Errorf("open library for %s: %w", selected, err)
	}
	if err := i.thumbs.SetDir(filepath.Join(dir, "thumbcache")); err != nil {
		i.log.Warn("thumbnail cache folder", "err", err)
	}
	i.uploads.SetDataDir(dir)
	i.syncMgr.ClearStatus()
	i.active = key
	epoch := i.epoch.Add(1)
	i.syncMgr.Unblock()

	i.log.Info("library switched account", "account", selected, "epoch", epoch)
	i.hub.Publish("account", server.AccountInfo{Account: selected, Epoch: epoch})
	if selected != "" {
		i.syncMgr.Trigger(false)
	}
	if i.onApplied != nil {
		go i.onApplied(selected)
	}
	return nil
}

func migrateLegacyLibrary(log *slog.Logger, dataDir, selected string) {
	if selected == "" {
		return
	}
	legacy := filepath.Join(dataDir, "library.json")
	if _, err := os.Stat(legacy); err != nil {
		return
	}
	dir := library.AccountDir(dataDir, selected)
	target := filepath.Join(dir, "library.json")
	if _, err := os.Stat(target); err == nil {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	if err := os.Rename(legacy, target); err != nil {
		log.Warn("could not move existing library", "err", err)
		return
	}
	for _, name := range []string{"thumbcache", "resumable.json"} {
		from, to := filepath.Join(dataDir, name), filepath.Join(dir, name)
		if _, err := os.Stat(from); err != nil {
			continue
		}
		if _, err := os.Stat(to); err == nil {
			continue
		}
		_ = os.Rename(from, to)
	}
	log.Info("moved existing library into per-account folder", "account", selected)
}

func URLs(port int, lan bool) []string {
	p := strconv.Itoa(port)
	out := []string{"http://localhost:" + p}
	if !lan {
		return out
	}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return out
	}
	var lanIPs []string
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok {
			continue
		}
		ip := ipnet.IP.To4()
		if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			continue
		}
		lanIPs = append(lanIPs, ip.String())
	}
	sort.Strings(lanIPs)
	for _, ip := range lanIPs {
		out = append(out, "http://"+net.JoinHostPort(ip, p))
	}
	return out
}
