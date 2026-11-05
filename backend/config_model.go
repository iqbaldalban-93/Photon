package backend

// The persisted app configuration model. JSON tags shape the frontend
// contract; koanf tags map onto the YAML config file keys.

type Config struct {
	Credentials                   []string        `json:"credentials" koanf:"credentials"`
	Selected                      string          `json:"selected" koanf:"selected"`
	Proxy                         string          `json:"proxy" koanf:"proxy"`
	UseQuota                      bool            `json:"useQuota" koanf:"use_quota"`
	Saver                         bool            `json:"saver" koanf:"saver"`
	Recursive                     bool            `json:"recursive" koanf:"recursive"`
	ForceUpload                   bool            `json:"forceUpload" koanf:"force_upload"`
	UploadThreads                 int             `json:"uploadThreads" koanf:"upload_threads"`
	DeleteFromHost                bool            `json:"deleteFromHost" koanf:"delete_from_host"`
	DisableUnsupportedFilesFilter bool            `json:"disableUnsupportedFilesFilter" koanf:"disable_unsupported_files_filter"`
	SetDateFromFilename           bool            `json:"setDateFromFilename" koanf:"set_date_from_filename"`
	ExcludePattern                string          `json:"excludePattern" koanf:"exclude_pattern"`
	CloseToTray                   bool            `json:"closeToTray" koanf:"close_to_tray"`
	NotifyOnComplete              bool            `json:"notifyOnComplete" koanf:"notify_on_complete"`
	AutoRetryFailed               bool            `json:"autoRetryFailed" koanf:"auto_retry_failed"`
	UseExifDate                   bool            `json:"useExifDate" koanf:"use_exif_date"`
	WatchedFolders                []WatchedFolder `json:"watchedFolders" koanf:"watched_folders"`
	SavedAlbums                   []SavedAlbum    `json:"savedAlbums" koanf:"saved_albums"`

	LibraryWebEnabled bool `json:"libraryWebEnabled" koanf:"library_web_enabled"`
	LibraryWebPort    int  `json:"libraryWebPort" koanf:"library_web_port"`
	LibraryWebLAN     bool `json:"libraryWebLan" koanf:"library_web_lan"`
}

type WatchedFolder struct {
	Path    string `json:"path" koanf:"path"`
	Album   string `json:"album" koanf:"album"`
	Enabled bool   `json:"enabled" koanf:"enabled"`
}

type SavedAlbum struct {
	Account string `json:"Account" koanf:"account"`
	Name    string `json:"Name" koanf:"name"`
	Key     string `json:"Key" koanf:"key"`
}

type ConfigManager struct{}
