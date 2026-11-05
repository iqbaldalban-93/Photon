package backend

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	gphotoclient "app/internal/gphotos"
	"app/internal/hashx"
	"app/internal/uploadcore"
)

// uploadFileWithCallback is the per-file desktop upload pipeline: hash,
// history/remote dedupe check, upload token, resumable upload, commit, and
// optional delete-from-host.

func uploadFileWithCallback(ctx context.Context, deps workerDeps, filePath string, workerID int, callback ProgressCallback) (string, bool, error) {
	api := deps.api
	paused := deps.paused
	fileName := filepath.Base(filePath)
	mediakey := ""

	var statSize int64
	var statMtime int64
	var haveStat bool
	if statInfo, statErr := os.Stat(filePath); statErr == nil {
		statMtime = statInfo.ModTime().Unix()
		statSize = statInfo.Size()
		haveStat = true
	} else if deps.fileMetaFor != nil {
		if meta, ok := deps.fileMetaFor(filePath); ok {
			statMtime = meta.ModTime.Unix()
			statSize = meta.Size
			haveStat = true
		}
	}

	var sha1_hash_bytes []byte
	var historyKey string
	historyHit := false
	if haveStat {
		if rec, ok := lookupUploadHistory(api.account, filePath, statSize, statMtime); ok && len(rec.Sha1) == sha1.Size && rec.MediaKey != "" {
			historyHit = true
			historyKey = rec.MediaKey
			sha1_hash_bytes = rec.Sha1
			callback(EventThreadStatus, ThreadStatus{
				WorkerID: workerID,
				Status:   "hashing",
				FilePath: filePath,
				FileName: fileName,
				Message:  "Hash reused from upload history",
			})
		}
	}
	if !historyHit {
		callback(EventThreadStatus, ThreadStatus{
			WorkerID: workerID,
			Status:   "hashing",
			FilePath: filePath,
			FileName: fileName,
			Message:  "Hashing...",
		})
		var err error
		sha1_hash_bytes, err = hashx.CalculateSHA1(ctx, filePath)
		if err != nil {
			return "", false, fmt.Errorf("error calculating hash file: %w", err)
		}
	}

	sha1_hash_b64 := base64.StdEncoding.EncodeToString(sha1_hash_bytes)

	if deps.AlwaysDeduplicate || !deps.cfg.ForceUpload {
		if historyHit && !deps.cfg.DeleteFromHost {
			callback(EventThreadStatus, ThreadStatus{
				WorkerID: workerID,
				Status:   "checking",
				FilePath: filePath,
				FileName: fileName,
				Message:  "Already in library (upload history)",
			})
			if haveStat {
				callback(EventUploadTotalBytesDelta, -statSize)
			}
			callback(EventThreadStatus, ThreadStatus{
				WorkerID: workerID,
				Status:   "completed",
				FilePath: filePath,
				FileName: fileName,
				Message:  "Already in library",
			})
			if deps.cfg.DeleteFromHost {
				if err := deleteFileFromHost(filePath); err != nil {
					return historyKey, true, fmt.Errorf("file exists in library but failed to delete local copy: %w", err)
				}
			}
			return historyKey, true, nil
		}

		callback(EventThreadStatus, ThreadStatus{
			WorkerID: workerID,
			Status:   "checking",
			FilePath: filePath,
			FileName: fileName,
			Message:  "Checking if file exists in library...",
		})

		var checkErr error
		mediakey, checkErr = api.FindRemoteMediaByHash(ctx, sha1_hash_bytes)
		if checkErr != nil {
			callback(EventThreadStatus, ThreadStatus{
				WorkerID: workerID,
				Status:   "checking",
				FilePath: filePath,
				FileName: fileName,
				Message:  fmt.Sprintf("Hash check warning: %v, proceeding with upload", checkErr),
			})
		}
		if len(mediakey) > 0 {
			if haveStat {
				recordUpload(api.account, filePath, statSize, statMtime, sha1_hash_bytes, mediakey)
			}
			if haveStat {
				callback(EventUploadTotalBytesDelta, -statSize)
			}
			callback(EventThreadStatus, ThreadStatus{
				WorkerID: workerID,
				Status:   "completed",
				FilePath: filePath,
				FileName: fileName,
				Message:  "Already in library",
			})
			if deps.cfg.DeleteFromHost {
				if err := deleteFileFromHost(filePath); err != nil {
					return mediakey, true, fmt.Errorf("file exists in library but failed to delete local copy: %w", err)
				}
			}
			return mediakey, true, nil
		}
	}

	fileSize := statSize
	fileBaseName := fileName
	if !haveStat {
		fileInfo, statErr := os.Stat(filePath)
		if statErr != nil {
			return "", false, fmt.Errorf("error getting file info: %w", statErr)
		}
		fileSize = fileInfo.Size()
		fileBaseName = fileInfo.Name()
	}

	if haveStat {
		if cur, curErr := os.Stat(filePath); curErr == nil && (cur.Size() != statSize || cur.ModTime().Unix() != statMtime) {
			return "", false, fmt.Errorf("file changed while it was being processed; it will be retried")
		}
	}

	uploadTimestamp := resolveUploadTimestamp(filePath, statMtime, deps.cfg)
	live := CurrentConfig() // device flags are read at commit time, as before

	// Shared per-file pipeline: token → resumable upload → retried commit.
	mediaKey, err := uploadcore.UploadFile(ctx, filePath, uploadcore.Options{
		API:         api,
		Concurrency: deps.concurrency,
		Sink: func(status, message string, uploaded, total int64, attempt int) {
			callback(EventThreadStatus, ThreadStatus{
				WorkerID:      workerID,
				Status:        status,
				FilePath:      filePath,
				FileName:      fileName,
				Message:       message,
				BytesUploaded: uploaded,
				BytesTotal:    total,
				Attempt:       attempt,
			})
		},
		Device:          gphotoclient.DeviceFromFlags(live.Saver, live.UseQuota),
		FileName:        fileBaseName,
		FileSize:        fileSize,
		SHA1Hash:        sha1_hash_bytes,
		SHA1B64:         sha1_hash_b64,
		UploadTimestamp: uploadTimestamp,
		OnPause: func() bool {
			return paused != nil && paused()
		},
	})
	if err != nil {
		return "", false, err
	}

	if haveStat {
		recordUpload(api.account, filePath, statSize, statMtime, sha1_hash_bytes, mediaKey)
	}

	if deps.cfg.DeleteFromHost {
		if err := deleteFileFromHost(filePath); err != nil {
			return mediaKey, false, fmt.Errorf("uploaded successfully but failed to delete file: %w", err)
		}
	}

	return mediaKey, false, nil
}
