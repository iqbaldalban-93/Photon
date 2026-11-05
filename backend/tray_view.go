package backend

import (
	"fmt"
)

// TrayState view model: composes the tracker snapshot with the web-library
// and account state, plus the tray action entry points and format helpers.

type TrayState struct {
	StatusLine  string `json:"statusLine"`
	LibraryLine string `json:"libraryLine"`
	VideoLine   string `json:"videoLine"`

	Icon           string `json:"icon"`
	UploadActive   bool   `json:"uploadActive"`
	UploadPaused   bool   `json:"uploadPaused"`
	WatchAvailable bool   `json:"watchAvailable"`
	WatchPaused    bool   `json:"watchPaused"`
	LibraryRunning bool   `json:"libraryRunning"`
	LibrarySyncing bool   `json:"librarySyncing"`
	LibraryURL     string `json:"libraryUrl"`

	Accounts      []TrayAccount `json:"accounts"`
	ActiveAccount string        `json:"activeAccount"`
	WebUploading  bool          `json:"webUploading"`
	SpeedBps      int64         `json:"speedBps"`

	Phase     string `json:"phase"`
	Done      int    `json:"done"`
	Total     int    `json:"total"`
	Percent   int    `json:"percent"`
	Failed    int    `json:"failed"`
	Items     int    `json:"items"`
	Videos    int    `json:"videos"`
	Photos    int    `json:"photos"`
	AllVideos bool   `json:"allVideos"`
	TotalSize string `json:"totalSize"`
	VideoSize string `json:"videoSize"`
	PhotoSize string `json:"photoSize"`
}

type TrayAccount struct {
	Email  string `json:"email"`
	Active bool   `json:"active"`
}

func (s TrayState) StructureKey() string {
	accts := ""
	for _, a := range s.Accounts {
		accts += a.Email + ","
	}
	return fmt.Sprintf("%s|%t|%t|%t|%t|%t|%t|%s|%t|%s|%s", s.Icon, s.UploadActive, s.UploadPaused,
		s.WatchAvailable, s.WatchPaused, s.LibraryRunning, s.LibrarySyncing, s.LibraryURL, s.VideoLine != "",
		accts, s.ActiveAccount)
}

func CollectTrayState(t *TrayTracker, w *WatchService) TrayState {
	var s TrayState
	u := t.snapshot()

	var webRunning, webPaused bool
	var webTotal, webDone, webFailed int
	var webSpeed float64
	var syncRunning bool
	var syncErr string
	var stats map[string]any
	if inst := libraryInstance(); inst != nil {
		s.LibraryRunning = true
		s.LibraryURL = LibraryWebURL()
		webRunning, webPaused, webTotal, webDone, webFailed = inst.WebUploadProgress()
		webSpeed = inst.WebUploadSpeed()
		st := inst.SyncStatus()
		syncRunning, syncErr = st.Running, st.LastError
		stats = inst.LibraryStats()
	}
	s.LibrarySyncing = syncRunning
	s.WebUploading = webRunning

	cfg := CurrentConfig()
	s.ActiveAccount = cfg.Selected
	s.Accounts = []TrayAccount{}
	for _, e := range AccountEmails(cfg) {
		s.Accounts = append(s.Accounts, TrayAccount{Email: e, Active: sameAccount(e, cfg.Selected)})
		if sameAccount(e, cfg.Selected) {
			s.ActiveAccount = e
		}
	}

	if w != nil {
		s.WatchAvailable = true
		s.WatchPaused = w.IsPaused()
	}

	s.UploadActive = u.Active || webRunning
	s.UploadPaused = (u.Active && u.Paused) || (webRunning && webPaused)

	var speed float64
	if u.Active && !u.Paused && u.Phase != "scanning" {
		speed += u.SpeedBps
	}
	if webRunning && !webPaused {
		speed += webSpeed
	}
	s.SpeedBps = int64(speed)
	speedSuffix := ""
	if s.SpeedBps > 0 {
		speedSuffix = " · " + formatBytes(s.SpeedBps) + "/s"
	}

	switch {
	case u.Active && u.Phase == "scanning":
		s.StatusLine = fmt.Sprintf("Scanning files… (%d)", u.Scanned)
	case u.Active:
		s.StatusLine = progressLine("Uploading", u.Done, u.Total, desktopPercent(u), u.Paused) + speedSuffix
	case webRunning:
		s.StatusLine = progressLine("Uploading (web)", webDone, webTotal, filePercent(webDone, webTotal), webPaused) + speedSuffix
	case syncRunning:
		s.StatusLine = "Syncing library…"
	case s.WatchPaused:
		s.StatusLine = "Idle · watched folders paused"
	default:
		s.StatusLine = "Idle"
	}
	if u.Active && syncRunning {
		s.StatusLine += " · syncing library"
	}

	if s.LibraryRunning {
		items, _ := stats["totalItems"].(int)
		videos, _ := stats["videos"].(int)
		photos, _ := stats["photos"].(int)
		totalSize, _ := stats["totalSize"].(int64)
		videoSize, _ := stats["videoSize"].(int64)
		photoSize := max(totalSize-videoSize, 0)
		s.Items, s.Videos, s.Photos = items, videos, photos
		s.TotalSize, s.VideoSize, s.PhotoSize = formatBytes(totalSize), formatBytes(videoSize), formatBytes(photoSize)
		s.AllVideos = items > 0 && videos == items
		s.LibraryLine = fmt.Sprintf("Videos: %s · %s", formatCount(videos), s.VideoSize)
		s.VideoLine = fmt.Sprintf("Photos: %s · %s", formatCount(photos), s.PhotoSize)
	} else {
		s.LibraryLine = "Library: web library is off"
	}

	switch {
	case u.Active:
		s.Phase, s.Done, s.Total, s.Failed = u.Phase, u.Done, u.Total, u.Failed
	case webRunning:
		s.Phase, s.Done, s.Total, s.Failed = "uploading", webDone, webTotal, webFailed
	}
	switch {
	case u.Active:
		s.Percent = desktopPercent(u)
	default:
		s.Percent = filePercent(s.Done, s.Total)
	}

	switch {
	case s.UploadPaused:
		s.Icon = "paused"
	case u.HadError || (!u.Active && u.Failed > 0) || webFailed > 0 || syncErr != "":
		s.Icon = "error"
	case s.UploadActive || syncRunning:
		s.Icon = "active"
	case s.WatchPaused:
		s.Icon = "paused"
	default:
		s.Icon = "idle"
	}
	return s
}

func desktopPercent(u trayUploadSnapshot) int {
	if u.TotalBytes > 0 {

		return int(min(100, max(0, (u.DoneBytes*100+u.TotalBytes/2)/u.TotalBytes)))
	}
	return filePercent(u.Done, u.Total)
}

func filePercent(done, total int) int {
	if total <= 0 {
		return 0
	}
	return min(100, done*100/total)
}

func progressLine(verb string, done, total, percent int, paused bool) string {
	line := verb
	if paused {
		line = "Paused"
	}
	if total > 0 {
		return fmt.Sprintf("%s %d/%d (%d%%)", line, done, total, percent)
	}
	return line + "…"
}

func formatCount(n int) string {
	s := fmt.Sprintf("%d", n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit && exp < 4; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTP"[exp])
}

func TrayToggleUploadPause(t *TrayTracker, um *UploadManager) {
	u := t.snapshot()
	webRunning, webPaused := false, false
	if inst := libraryInstance(); inst != nil {
		webRunning, webPaused, _, _, _ = inst.WebUploadProgress()
	}
	pausedNow := (u.Active && u.Paused) || (webRunning && webPaused)
	if u.Active && um != nil {
		if pausedNow {
			um.Resume()
		} else {
			um.Pause()
		}
	}
	if webRunning {
		SetWebUploadsPaused(!pausedNow)
	}
}

func TrayCancelUploads(t *TrayTracker, um *UploadManager) {
	if t.snapshot().Active && um != nil {
		um.Cancel()
	}
	if inst := libraryInstance(); inst != nil {
		if running, _, _, _, _ := inst.WebUploadProgress(); running {
			inst.CancelWebUploads()
		}
	}
}
