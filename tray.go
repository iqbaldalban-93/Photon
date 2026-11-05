package main

import (
	_ "embed"
	"errors"
	"log/slog"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"app/backend"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed build/trayicon-active.png
var trayIconActive []byte

//go:embed build/trayicon-paused.png
var trayIconPaused []byte

//go:embed build/trayicon-error.png
var trayIconError []byte

type trayController struct {
	window  *application.WebviewWindow
	flyout  *application.WebviewWindow
	tray    *application.SystemTray
	app     *application.App
	tracker *backend.TrayTracker
	watch   *backend.WatchService
	uploads *backend.UploadManager

	mu       sync.Mutex
	lastIcon string
	lastText string
	stop     chan struct{}

	flyoutVisible atomic.Bool
	lastHideNano  atomic.Int64
}

func newTrayController(app *application.App, window *application.WebviewWindow,
	tracker *backend.TrayTracker, watch *backend.WatchService, uploads *backend.UploadManager) *trayController {
	return &trayController{app: app, window: window, tracker: tracker, watch: watch, uploads: uploads, stop: make(chan struct{})}
}

func (c *trayController) iconFor(name string) []byte {
	switch name {
	case "active":
		return trayIconActive
	case "paused":
		return trayIconPaused
	case "error":
		return trayIconError
	}
	return trayIcon
}

func (c *trayController) showWindow() { c.window.Show().Focus() }

func (c *trayController) toggleFlyout() {
	if c.flyout.IsVisible() {
		c.flyout.Hide()
		return
	}

	if time.Since(time.Unix(0, c.lastHideNano.Load())) < 400*time.Millisecond {
		return
	}
	_ = c.tray.PositionWindow(c.flyout, 8)
	c.flyout.Show().Focus()
}

func (c *trayController) Start() {
	t := c.app.SystemTray.New()
	t.SetIcon(c.iconFor("idle"))
	t.SetTooltip(title)
	c.tray = t

	c.createFlyout()
	t.OnClick(func() {})
	t.OnRightClick(c.toggleFlyout)
	t.OnDoubleClick(func() {
		c.flyout.Hide()
		c.showWindow()
	})

	c.app.Event.On("trayAction", func(e *application.CustomEvent) {
		if act, ok := e.Data.(string); ok {
			c.handleAction(act)
		}
	})
	c.app.Event.On("trayStateRequest", func(*application.CustomEvent) { go c.refresh() })

	c.app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		go func() {
			tick := time.NewTicker(time.Second)
			defer tick.Stop()
			for {
				select {
				case <-tick.C:
					c.refresh()
				case <-c.stop:
					return
				}
			}
		}()
	})
}

func (c *trayController) createFlyout() {
	w := c.app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             "tray-flyout",
		Title:            title,
		URL:              "/tray.html#tray",
		Width:            340,
		Height:           556,
		Frameless:        true,
		AlwaysOnTop:      true,
		Hidden:           true,
		DisableResize:    true,
		BackgroundColour: application.NewRGB(10, 10, 10),
		Windows:          application.WindowsWindow{HiddenOnTaskbar: true},
	})
	w.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		e.Cancel()
		w.Hide()
	})
	w.OnWindowEvent(events.Common.WindowShow, func(*application.WindowEvent) {
		c.flyoutVisible.Store(true)
		go c.refresh()
	})
	w.OnWindowEvent(events.Common.WindowHide, func(*application.WindowEvent) {
		c.flyoutVisible.Store(false)
		c.lastHideNano.Store(time.Now().UnixNano())
	})

	w.OnWindowEvent(events.Common.WindowLostFocus, func(*application.WindowEvent) {
		w.Hide()
	})
	c.flyout = w
}

func (c *trayController) handleAction(act string) {

	for prefix, force := range map[string]bool{"switchAccount:": false, "switchAccountForce:": true} {
		if email, ok := strings.CutPrefix(act, prefix); ok {
			go func() {
				if err := backend.SwitchAccount(email, force); err != nil && !errors.Is(err, backend.ErrWebUploadsRunning) {
					slog.Warn("tray: could not switch account", "error", err)
				}
				c.refresh()
			}()
			return
		}
	}
	switch act {
	case "openLibrary":
		if url := backend.LibraryWebURL(); url != "" {
			openBrowser(url)
		}
	case "sync":
		backend.TriggerLibrarySync()
	case "pauseUploads":
		backend.TrayToggleUploadPause(c.tracker, c.uploads)
	case "cancelUploads":
		backend.TrayCancelUploads(c.tracker, c.uploads)
	case "toggleWatch":
		c.watch.SetPaused(!c.watch.IsPaused())
	case "showApp":
		c.flyout.Hide()
		c.showWindow()
		return
	case "quit":
		c.app.Quit()
		return
	default:
		return
	}
	go c.refresh()
}

func (c *trayController) Refresh() { go c.refresh() }

func (c *trayController) Stop() {
	select {
	case <-c.stop:
	default:
		close(c.stop)
	}
}

func (c *trayController) refresh() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.tray == nil {
		return
	}
	s := backend.CollectTrayState(c.tracker, c.watch)

	application.InvokeSync(func() {
		if s.Icon != c.lastIcon {
			c.tray.SetIcon(c.iconFor(s.Icon))
			c.lastIcon = s.Icon
		}
		text := s.StatusLine + "\n" + s.LibraryLine
		if s.ActiveAccount != "" {
			text = s.ActiveAccount + "\n" + text
		}
		if s.VideoLine != "" {
			text += "\n" + s.VideoLine
		}
		if text != c.lastText {
			c.tray.SetTooltip(title + "\n" + text)
			c.lastText = text
		}
	})
	if c.flyoutVisible.Load() {
		c.app.Event.Emit("trayState", s)
	}
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
