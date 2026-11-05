package library

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"app/libweb/gphotos"
)

type SyncStatus struct {
	Running      bool      `json:"running"`
	Phase        string    `json:"phase"`
	Pages        int       `json:"pages"`
	ItemsUpdated int       `json:"itemsUpdated"`
	ItemsDeleted int       `json:"itemsDeleted"`
	StartedAt    time.Time `json:"startedAt,omitempty"`
	FinishedAt   time.Time `json:"finishedAt,omitempty"`
	LastError    string    `json:"lastError,omitempty"`
}

type SyncManager struct {
	index *Index
	log   *slog.Logger

	mu      sync.Mutex
	status  SyncStatus
	cancel  context.CancelFunc
	blocked bool
}

func NewSyncManager(index *Index, log *slog.Logger) *SyncManager {
	return &SyncManager{index: index, log: log}
}

func (sm *SyncManager) Status() SyncStatus {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.status
}

func (sm *SyncManager) Cancel() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if sm.cancel != nil {
		sm.cancel()
	}
}

func (sm *SyncManager) Block() {
	sm.mu.Lock()
	sm.blocked = true
	sm.mu.Unlock()
}

func (sm *SyncManager) Unblock() {
	sm.mu.Lock()
	sm.blocked = false
	sm.mu.Unlock()
}

func (sm *SyncManager) StopAndWait(timeout time.Duration) bool {
	sm.Cancel()
	deadline := time.Now().Add(timeout)
	for {
		sm.mu.Lock()
		running := sm.status.Running
		sm.mu.Unlock()
		if !running {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func (sm *SyncManager) ClearStatus() {
	sm.mu.Lock()
	if !sm.status.Running {
		sm.status = SyncStatus{}
	}
	sm.mu.Unlock()
}

func (sm *SyncManager) Trigger(full bool) bool {
	sm.mu.Lock()
	if sm.status.Running || sm.blocked {
		sm.mu.Unlock()
		return false
	}
	ctx, cancel := context.WithCancel(context.Background())
	sm.cancel = cancel
	sm.status = SyncStatus{Running: true, Phase: "init", StartedAt: time.Now()}
	sm.mu.Unlock()

	go func() {
		defer func() {
			sm.mu.Lock()
			sm.status.Running = false
			sm.status.Phase = "idle"
			sm.status.FinishedAt = time.Now()
			sm.cancel = nil
			sm.mu.Unlock()
			_ = sm.index.Save()
		}()
		err := sm.run(ctx, full)
		if err != nil {
			sm.mu.Lock()
			sm.status.LastError = err.Error()
			sm.mu.Unlock()
			sm.log.Error("library sync failed", "err", err)
		} else {
			sm.mu.Lock()
			sm.status.LastError = ""
			sm.mu.Unlock()
			sm.log.Info("library sync complete",
				"pages", sm.status.Pages, "updated", sm.status.ItemsUpdated, "deleted", sm.status.ItemsDeleted)
		}
	}()
	return true
}

func (sm *SyncManager) setPhase(p string) {
	sm.mu.Lock()
	sm.status.Phase = p
	sm.mu.Unlock()
}

func (sm *SyncManager) bumpPage(items, deleted int) {
	sm.mu.Lock()
	sm.status.Pages++
	sm.status.ItemsUpdated += items
	sm.status.ItemsDeleted += deleted
	sm.mu.Unlock()
}

func (sm *SyncManager) run(ctx context.Context, full bool) error {
	api, err := gphotos.NewApi()
	if err != nil {
		return err
	}

	_, _, initDone := sm.index.Tokens()
	if full {
		sm.index.ResetForFullSync()
		initDone = false
	}

	if !initDone {
		return sm.initialSync(ctx, api)
	}
	return sm.deltaSync(ctx, api)
}

func (sm *SyncManager) initialSync(ctx context.Context, api *gphotos.Api) error {
	syncToken, resumeToken, _ := sm.index.Tokens()

	if resumeToken != "" {
		if err := sm.processPagesInit(ctx, api, resumeToken); err != nil {
			return err
		}
	}

	sm.setPhase("init")
	body, err := api.GetLibraryState(ctx, syncToken)
	if err != nil {
		return err
	}
	page, err := gphotos.ParseLibraryResponse(body, sm.log)
	if err != nil {
		return err
	}
	sm.index.ApplyPage(page)
	sm.index.SetTokens(page.SyncToken, page.ResumeToken)
	sm.bumpPage(len(page.Items), len(page.Deletions))

	if page.ResumeToken != "" {
		if err := sm.processPagesInit(ctx, api, page.ResumeToken); err != nil {
			return err
		}
	}
	sm.index.SetInitDone()
	return sm.index.Save()
}

func (sm *SyncManager) processPagesInit(ctx context.Context, api *gphotos.Api, resumeToken string) error {
	sm.setPhase("pages")
	next := resumeToken
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		body, err := api.GetLibraryPageInit(ctx, next)
		if err != nil {
			return err
		}
		page, err := gphotos.ParseLibraryResponse(body, sm.log)
		if err != nil {
			return err
		}
		sm.index.ApplyPage(page)
		sm.index.SetTokens("", page.ResumeToken)
		sm.bumpPage(len(page.Items), len(page.Deletions))
		sm.index.SaveIfStale(20 * time.Second)
		if page.ResumeToken == "" {
			return nil
		}
		next = page.ResumeToken
	}
}

func (sm *SyncManager) deltaSync(ctx context.Context, api *gphotos.Api) error {
	const maxCycles = 10
	previous := ""
	for cycle := 0; cycle < maxCycles; cycle++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		syncToken, _, _ := sm.index.Tokens()
		if previous != "" && syncToken == previous {
			sm.log.Warn("sync token unchanged between cycles; stopping")
			return nil
		}

		sm.setPhase("delta")
		body, err := api.GetLibraryState(ctx, syncToken)
		if err != nil {
			return err
		}
		page, err := gphotos.ParseLibraryResponse(body, sm.log)
		if err != nil {
			return err
		}
		sm.index.ApplyPage(page)
		sm.index.SetTokens(page.SyncToken, page.ResumeToken)
		sm.bumpPage(len(page.Items), len(page.Deletions))

		if page.ResumeToken != "" {
			sm.setPhase("delta-pages")
			next := page.ResumeToken
			for {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				pbody, err := api.GetLibraryPage(ctx, next, syncToken)
				if err != nil {
					return err
				}
				ppage, err := gphotos.ParseLibraryResponse(pbody, sm.log)
				if err != nil {
					return err
				}
				sm.index.ApplyPage(ppage)
				sm.index.SetTokens("", ppage.ResumeToken)
				sm.bumpPage(len(ppage.Items), len(ppage.Deletions))
				sm.index.SaveIfStale(20 * time.Second)
				if ppage.ResumeToken == "" {
					break
				}
				next = ppage.ResumeToken
			}
		}

		if !page.Continue {
			break
		}
		previous = syncToken
	}
	return sm.index.Save()
}
