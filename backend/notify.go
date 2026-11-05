package backend

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/gen2brain/beeep"
)

const notificationAppName = "Photon"

const notificationIconFilename = notificationAppName + "-notification.png"

func init() {
	beeep.AppName = notificationAppName
}

func PrepareNotificationIcon(dir string, png []byte) string {
	path := filepath.Join(dir, notificationIconFilename)
	if err := os.WriteFile(path, png, 0o600); err != nil {
		slog.Warn("could not write notification icon", "error", err)
		return ""
	}
	return path
}

var notificationIconPath atomic.Pointer[string]

func SetNotificationIcon(path string) {
	notificationIconPath.Store(&path)
}

func notifyBatchChainComplete(summary batchSummary, cancelled bool) {
	if !CurrentConfig().NotifyOnComplete {
		return
	}

	var message string
	if cancelled {
		message = fmt.Sprintf("Upload cancelled - %d uploaded, %d failed, %d skipped", summary.uploaded, summary.failed, summary.skipped)
	} else {
		message = fmt.Sprintf("Upload complete - %d uploaded, %d failed, %d skipped", summary.uploaded, summary.failed, summary.skipped)
	}

	go func() {
		var icon string
		if p := notificationIconPath.Load(); p != nil {
			icon = *p
		}
		if err := beeep.Notify("Photon", message, icon); err != nil {
			slog.Warn("could not show completion notification", "error", err)
		}
	}()
}
