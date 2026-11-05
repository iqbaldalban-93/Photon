package main

import (
	"embed"
	"fmt"
	"log"
	"os"
	"strings"

	"app/backend"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/trayicon.png
var trayIcon []byte

//go:embed build/notification.png
var notificationIcon []byte

var title = "Photon"

func main() {
	runGUI()
}

func runGUI() {
	normalizeFrontendDevServerURL()

	watchService := &backend.WatchService{}

	wailsApp := application.New(application.Options{
		Name:        "com.DalBan.Photon",
		Description: "Google Photos unofficial client",
		Services: []application.Service{
			application.NewService(&backend.ConfigManager{}),
			application.NewService(&backend.UploadService{}),
			application.NewService(watchService),
		},
		Assets: application.AssetOptions{
			Handler: application.BundledAssetFileServer(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	window := wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:               title,
		Frameless:           false,
		Width:               440,
		Height:              700,
		MinWidth:            400,
		MinHeight:           600,
		DisableResize:       false,
		MaximiseButtonState: application.ButtonEnabled,
		BackgroundType:      application.BackgroundTypeTranslucent,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 0,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		URL: "/",
	})

	window.Show()

	_ = backend.LoadConfig()

	backend.StartLibraryWebIfEnabled()

	window.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		if !backend.CurrentConfig().CloseToTray {
			return
		}
		event.Cancel()
		window.Hide()
	})

	backend.OnCloseToTrayChanged = func(enabled bool) {
		if enabled {
			return
		}
		if !window.IsVisible() {
			window.Show()
		}
	}

	app := backend.NewWailsApp(wailsApp)
	trayTracker := &backend.TrayTracker{}
	app.OnEvent = trayTracker.Handle
	uploadManager := backend.NewUploadManager(app)

	tray := newTrayController(wailsApp, window, trayTracker, watchService, uploadManager)
	tray.Start()
	defer tray.Stop()

	backend.OnAccountChanged = func(email string) {
		wailsApp.Event.Emit(backend.EventAccountChanged, email)
		tray.Refresh()
	}

	if err := backend.AttachWatchService(watchService, app, uploadManager); err != nil {
		wailsApp.Logger.Warn("watched folders disabled", "error", err)
	}

	backend.SetNotificationIcon(backend.PrepareNotificationIcon(os.TempDir(), notificationIcon))

	wailsApp.Event.On(backend.EventUploadCancel, func(e *application.CustomEvent) {
		uploadManager.Cancel()
	})
	wailsApp.Event.On(backend.EventUploadPause, func(e *application.CustomEvent) {
		uploadManager.Pause()
	})
	wailsApp.Event.On(backend.EventUploadResume, func(e *application.CustomEvent) {
		uploadManager.Resume()
	})

	wailsApp.Event.On(backend.EventStartUpload, func(e *application.CustomEvent) {
		if data, ok := e.Data.(backend.StartUploadEvent); ok {
			wailsApp.Logger.Info("Starting upload", "fileCount", len(data.Files), "album", data.Album)
			uploadManager.Upload(app, data.Files, data.Album)
		} else {
			wailsApp.Logger.Error("startUpload: unexpected data type", "type", fmt.Sprintf("%T", e.Data))
		}
	})

	wailsApp.Event.On(backend.EventSkipFile, func(e *application.CustomEvent) {
		if data, ok := e.Data.(backend.SkipFileEvent); ok {
			uploadManager.SkipFile(data.Path)
		}
	})

	wailsApp.Event.On(backend.EventCancelQueuedFile, func(e *application.CustomEvent) {
		if data, ok := e.Data.(backend.CancelQueuedFileEvent); ok {
			uploadManager.CancelQueuedFile(app, data.Path)
		}
	})

	wailsApp.Event.On(backend.EventCheckResumableBatch, func(e *application.CustomEvent) {
		if batch := backend.LoadAndClearResumableBatch(); batch != nil {
			wailsApp.Event.Emit(backend.EventResumableBatchFound, backend.ResumableBatchInfo{
				Files: batch.Paths,
				Album: batch.Album,
				Total: batch.Total,
			})
		}
	})

	err := wailsApp.Run()
	watchService.Shutdown()
	backend.StopLibraryWeb()
	if err != nil {
		log.Fatal(err)
	}
}

func normalizeFrontendDevServerURL() {
	const envName = "FRONTEND_DEVSERVER_URL"

	value := os.Getenv(envName)
	value = strings.Replace(value, "http://localhost:", "http://127.0.0.1:", 1)
	value = strings.Replace(value, "https://localhost:", "https://127.0.0.1:", 1)
	if value != os.Getenv(envName) {
		_ = os.Setenv(envName, value)
	}
}
