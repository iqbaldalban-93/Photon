package backend

import (
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type uploadRecord struct {
	Sha1     []byte
	MediaKey string
}

var (
	historyOnce   sync.Once
	historyDBH    *sql.DB
	historyInitEr error
)

func historyDBPath() string {
	dir := filepath.Dir(ConfigPath)
	if dir == "" || dir == "." {
		dir = filepath.Join(getUserConfigDir(), "Photon")
	}
	return filepath.Join(dir, "history.db")
}

func historyDB() (*sql.DB, error) {
	historyOnce.Do(func() {
		path := historyDBPath()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			historyInitEr = err
			return
		}
		db, err := sql.Open("sqlite", path)
		if err != nil {
			historyInitEr = err
			return
		}
		db.SetMaxOpenConns(1)
		for _, pragma := range []string{
			`PRAGMA journal_mode=WAL`,
			`PRAGMA synchronous=NORMAL`,
			`PRAGMA busy_timeout=5000`,
		} {
			if _, err := db.Exec(pragma); err != nil {
				_ = db.Close()
				historyInitEr = err
				return
			}
		}
		if err := migrateHistorySchema(db); err != nil {
			_ = db.Close()
			historyInitEr = err
			return
		}
		historyDBH = db
	})
	if historyInitEr != nil {
		slog.Debug("upload history unavailable", "error", historyInitEr)
		return nil, historyInitEr
	}
	return historyDBH, nil
}

func migrateHistorySchema(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS uploads_v2 (
		account     TEXT NOT NULL,
		path        TEXT NOT NULL,
		size        INTEGER NOT NULL,
		mtime       INTEGER NOT NULL,
		sha1        BLOB NOT NULL,
		media_key   TEXT NOT NULL,
		uploaded_at INTEGER NOT NULL,
		PRIMARY KEY (account, path, size, mtime)
	)`); err != nil {
		return err
	}
	var legacy int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='uploads'`).Scan(&legacy); err != nil || legacy == 0 {
		return err
	}
	cfg := CurrentConfig()
	if len(cfg.Credentials) == 1 && cfg.Selected != "" {
		if _, err := db.Exec(`INSERT OR IGNORE INTO uploads_v2 (account, path, size, mtime, sha1, media_key, uploaded_at)
			SELECT ?, path, size, mtime, sha1, media_key, uploaded_at FROM uploads`, cfg.Selected); err != nil {
			return err
		}
	}
	_, err := db.Exec(`DROP TABLE uploads`)
	return err
}

func lookupUploadHistory(account, path string, size, mtime int64) (uploadRecord, bool) {
	db, err := historyDB()
	if err != nil {
		return uploadRecord{}, false
	}
	var rec uploadRecord
	err = db.QueryRow(`SELECT sha1, media_key FROM uploads_v2 WHERE account = ? AND path = ? AND size = ? AND mtime = ?`,
		account, path, size, mtime).Scan(&rec.Sha1, &rec.MediaKey)
	if err != nil {
		return uploadRecord{}, false
	}
	return rec, true
}

func recordUpload(account, path string, size, mtime int64, sha1 []byte, mediaKey string) {
	db, err := historyDB()
	if err != nil {
		return
	}
	_, err = db.Exec(`INSERT OR REPLACE INTO uploads_v2 (account, path, size, mtime, sha1, media_key, uploaded_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		account, path, size, mtime, sha1, mediaKey, time.Now().Unix())
	if err != nil {
		slog.Debug("could not record upload history", "error", err)
	}
}

func countUploadHistory() int64 {
	db, err := historyDB()
	if err != nil {
		return 0
	}
	var n int64
	if err := db.QueryRow(`SELECT COUNT(*) FROM uploads_v2`).Scan(&n); err != nil {
		return 0
	}
	return n
}

func clearUploadHistory() error {
	db, err := historyDB()
	if err != nil {
		return err
	}
	_, err = db.Exec(`DELETE FROM uploads_v2`)
	return err
}
