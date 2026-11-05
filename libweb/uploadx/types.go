package uploadx

import (
	"time"

	"app/internal/resumebatch"
)

// Status model and event payload types published over SSE. The JSON shapes
// are the browser-library contract and must not change.

type fileMeta struct {
	Size    int64
	ModTime time.Time
}

type ThreadStatus struct {
	WorkerID      int    `json:"workerId"`
	Status        string `json:"status"`
	FilePath      string `json:"filePath"`
	FileName      string `json:"fileName"`
	Message       string `json:"message"`
	BytesUploaded int64  `json:"bytesUploaded"`
	BytesTotal    int64  `json:"bytesTotal"`
	Attempt       int    `json:"attempt"`
}

type FileUploadResult struct {
	Path         string `json:"path"`
	FileName     string `json:"fileName"`
	MediaKey     string `json:"mediaKey,omitempty"`
	IsError      bool   `json:"isError"`
	Skipped      bool   `json:"skipped"`
	SkipCode     string `json:"skipCode,omitempty"`
	SkipReason   string `json:"skipReason,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

type Warning struct {
	Paths   []string `json:"paths"`
	Code    string   `json:"code"`
	Message string   `json:"message"`
}

type AlbumProgress struct {
	AlbumName  string `json:"albumName"`
	ItemsAdded int    `json:"itemsAdded"`
	TotalItems int    `json:"totalItems"`
	IsComplete bool   `json:"isComplete"`
	Error      string `json:"error,omitempty"`
}

type QueuedBatch struct {
	Position int      `json:"position"`
	Total    int      `json:"total"`
	Paths    []string `json:"paths"`
	Album    string   `json:"album"`
}

type Status struct {
	Running      bool               `json:"running"`
	Paused       bool               `json:"paused"`
	Total        int                `json:"total"`
	TotalBytes   int64              `json:"totalBytes"`
	Done         int                `json:"done"`
	Failed       int                `json:"failed"`
	Skipped      int                `json:"skipped"`
	Scanning     int                `json:"scanning"`
	Concurrency  [2]int             `json:"concurrency"`
	Workers      []ThreadStatus     `json:"workers"`
	Results      []FileUploadResult `json:"results"`
	Warnings     []Warning          `json:"warnings"`
	Album        *AlbumProgress     `json:"album,omitempty"`
	Queued       []QueuedBatch      `json:"queued"`
	Resumable    *ResumableBatch    `json:"resumable,omitempty"`
	StartedAt    time.Time          `json:"startedAt,omitempty"`
	FinishedAt   time.Time          `json:"finishedAt,omitempty"`
	LastErrorMsg string             `json:"lastError,omitempty"`
}

type ResumableBatch = resumebatch.Batch

type sentMark struct {
	path  string
	bytes int64
}

type queuedUpload struct {
	Paths []string
	Album string
}

type logger struct{}

func (logger) Info(msg string, kv ...any)  {}
func (logger) Error(msg string, kv ...any) {}
func (logger) Warn(msg string, kv ...any)  {}
