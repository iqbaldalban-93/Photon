package backend

import "time"

// Payload types for the events emitted by the upload pipeline. The JSON
// shapes are part of the frontend contract and must not change.

type StartUploadEvent struct {
	Files []string `json:"files"`
	Album string   `json:"album"`
}

type SkipFileEvent struct {
	Path string `json:"path"`
}

type CancelQueuedFileEvent struct {
	Path string `json:"path"`
}

type QueuedFileCancelledEvent struct {
	Path            string `json:"Path"`
	RemainingQueued int    `json:"RemainingQueued"`
}

type UploadConcurrencyStatus struct {
	Limit int `json:"Limit"`
	Max   int `json:"Max"`
}

type UploadPreflightProgress struct {
	Scanned int `json:"Scanned"`
}

type UploadQueuedEvent struct {
	Position int      `json:"Position"`
	Total    int      `json:"Total"`
	Paths    []string `json:"Paths"`
}

type UploadBatchStart struct {
	Total      int      `json:"Total"`
	TotalBytes int64    `json:"TotalBytes"`
	Files      []string `json:"Files"`
	Phase      string   `json:"Phase"`
}

type FileUploadResult struct {
	MediaKey     string   `json:"MediaKey"`
	IsError      bool     `json:"IsError"`
	Skipped      bool     `json:"Skipped"`
	SkipCode     string   `json:"SkipCode"`
	SkipReason   string   `json:"SkipReason"`
	Error        error    `json:"-"`
	ErrorMessage string   `json:"ErrorMessage"`
	Path         string   `json:"Path"`
	Paths        []string `json:"Paths"`
}

type ThreadStatus struct {
	WorkerID      int    `json:"WorkerID"`
	Status        string `json:"Status"`
	FilePath      string `json:"FilePath"`
	FileName      string `json:"FileName"`
	Message       string `json:"Message"`
	BytesUploaded int64  `json:"BytesUploaded"`
	BytesTotal    int64  `json:"BytesTotal"`
	Attempt       int    `json:"Attempt"`
}

type PreflightWarning struct {
	Paths   []string `json:"Paths"`
	Code    string   `json:"Code"`
	Message string   `json:"Message"`
}

type fileMeta struct {
	Size    int64
	ModTime time.Time
}

type UploadWorkKind string

const UploadWorkSingle UploadWorkKind = "single"

type SingleMedia struct {
	Path string
}

type UploadWorkItem struct {
	Kind   UploadWorkKind
	Single *SingleMedia
}

func uploadWorkPaths(item UploadWorkItem) []string {
	if item.Single != nil {
		return []string{item.Single.Path}
	}
	return nil
}

func uploadWorkPrimaryPath(item UploadWorkItem) string {
	paths := uploadWorkPaths(item)
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}
