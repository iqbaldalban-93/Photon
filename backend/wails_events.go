package backend

import "github.com/wailsapp/wails/v3/pkg/application"

// Canonical names of every Wails event crossing between the Go backend and
// the frontend. Registration happens in init below; emit and listen sites
// must use these constants so a rename cannot desynchronize the two sides.
const (
	EventUploadStart             = "uploadStart"
	EventUploadStop              = "uploadStop"
	EventFileStatus              = "FileStatus"
	EventThreadStatus            = "ThreadStatus"
	EventUploadWarning           = "uploadWarning"
	EventUploadCancel            = "uploadCancel"
	EventUploadPause             = "uploadPause"
	EventUploadResume            = "uploadResume"
	EventUploadPauseState        = "uploadPauseState"
	EventUploadTotalBytes        = "uploadTotalBytes"
	EventUploadTotalBytesDelta   = "uploadTotalBytesDelta"
	EventStartUpload             = "startUpload"
	EventUploadQueued            = "uploadQueued"
	EventAlbumProgress           = "albumProgress"
	EventAlbumComplete           = "albumComplete"
	EventAlbumError              = "albumError"
	EventSkipFile                = "skipFile"
	EventCancelQueuedFile        = "cancelQueuedFile"
	EventQueuedFileCancelled     = "queuedFileCancelled"
	EventUploadConcurrency       = "uploadConcurrency"
	EventUploadPreflightProgress = "uploadPreflightProgress"
	EventCheckResumableBatch     = "checkResumableBatch"
	EventResumableBatchFound     = "resumableBatchFound"
	EventWatcherBatchStarted     = "watcherBatchStarted"
	EventRetryBatchStarted       = "retryBatchStarted"
	EventUploadAutoRetry         = "uploadAutoRetry"
	EventAccountChanged          = "accountChanged"
)

func init() {
	application.RegisterEvent[UploadBatchStart](EventUploadStart)
	application.RegisterEvent[application.Void](EventUploadStop)
	application.RegisterEvent[FileUploadResult](EventFileStatus)
	application.RegisterEvent[ThreadStatus](EventThreadStatus)
	application.RegisterEvent[PreflightWarning](EventUploadWarning)
	application.RegisterEvent[application.Void](EventUploadCancel)
	application.RegisterEvent[application.Void](EventUploadPause)
	application.RegisterEvent[application.Void](EventUploadResume)
	application.RegisterEvent[bool](EventUploadPauseState)
	application.RegisterEvent[int64](EventUploadTotalBytes)
	application.RegisterEvent[int64](EventUploadTotalBytesDelta)
	application.RegisterEvent[StartUploadEvent](EventStartUpload)
	application.RegisterEvent[UploadQueuedEvent](EventUploadQueued)
	application.RegisterEvent[AlbumStatus](EventAlbumProgress)
	application.RegisterEvent[AlbumStatus](EventAlbumComplete)
	application.RegisterEvent[AlbumError](EventAlbumError)
	application.RegisterEvent[SkipFileEvent](EventSkipFile)
	application.RegisterEvent[CancelQueuedFileEvent](EventCancelQueuedFile)
	application.RegisterEvent[QueuedFileCancelledEvent](EventQueuedFileCancelled)
	application.RegisterEvent[UploadConcurrencyStatus](EventUploadConcurrency)
	application.RegisterEvent[UploadPreflightProgress](EventUploadPreflightProgress)
	application.RegisterEvent[application.Void](EventCheckResumableBatch)
	application.RegisterEvent[ResumableBatchInfo](EventResumableBatchFound)
	application.RegisterEvent[WatcherBatchEvent](EventWatcherBatchStarted)
	application.RegisterEvent[WatcherBatchEvent](EventRetryBatchStarted)
	application.RegisterEvent[int](EventUploadAutoRetry)
}
