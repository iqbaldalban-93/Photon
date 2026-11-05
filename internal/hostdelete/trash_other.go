//go:build !windows

package hostdelete

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

func Trash(path string) error {
	switch runtime.GOOS {
	case "linux":
		gio, err := exec.LookPath("gio")
		if err != nil {
			return fmt.Errorf("no trash available (install gio/glib to enable moving files to the trash): %w", err)
		}
		if out, err := exec.Command(gio, "trash", "--", path).CombinedOutput(); err != nil {
			slog.Debug("gio trash failed", "error", err, "output", string(out))
			return fmt.Errorf("could not move file to the trash: %w", err)
		}
		return nil
	case "darwin":
		abs, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		if out, ferr := exec.Command("osascript",
			"-e", "on run argv",
			"-e", `tell application "Finder" to delete (POSIX file (item 1 of argv) as alias)`,
			"-e", "end run",
			abs).CombinedOutput(); ferr == nil {
			return nil
		} else {
			slog.Debug("finder trash failed, falling back to rename", "error", ferr, "output", string(out))
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("could not locate the trash: %w", err)
		}
		name := filepath.Base(abs)
		trashPath := filepath.Join(home, ".Trash", name)
		if _, statErr := os.Stat(trashPath); statErr == nil {
			trashPath = filepath.Join(home, ".Trash", fmt.Sprintf("%d-%s", time.Now().UnixNano(), name))
		}
		if err := os.Rename(abs, trashPath); err != nil {
			return fmt.Errorf("could not move file to the trash (is it on another volume, or does Photon lack access to the Trash?): %w", err)
		}
		return nil
	}
	return fmt.Errorf("moving files to the trash is not supported on %s", runtime.GOOS)
}
