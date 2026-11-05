package backend

import (
	"path/filepath"

	"app/internal/resumebatch"
)

type ResumableBatch = resumebatch.Batch

type ResumableBatchInfo struct {
	Files []string `json:"Files"`
	Album string   `json:"Album"`
	Total int      `json:"Total"`
}

func resumeFilePath() string {
	dir := filepath.Dir(ConfigPath)
	if dir == "" || dir == "." {
		dir = filepath.Join(getUserConfigDir(), "Photon")
	}
	return filepath.Join(dir, "resume_batch.json")
}

func clearResumableBatch() {
	_ = resumebatch.Clear(resumeFilePath())
}

func LoadAndClearResumableBatch() *ResumableBatch {
	return resumebatch.LoadAndClear(resumeFilePath())
}
