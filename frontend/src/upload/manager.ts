import { Clipboard, Events } from "@wailsio/runtime";
import { reactive } from "vue";
import { baseName } from "../utils/format";
import { recordUploadResult } from "../utils/uploadResults";
import {
  isAlbumKey,
  isSkippedUploadWarning,
  type AlbumErrorEvent,
  type AlbumMode,
  type AlbumStatus,
  type FileUploadResult,
  type PreflightWarning,
  type QueuedFileCancelledEvent,
  type ThreadStatus,
  type UploadBatchStart,
  type UploadConcurrencyStatus,
  type UploadFileEntry,
  type UploadPreflightProgress,
  type UploadQueuedEvent,
  type UploadState,
  type WatcherBatchStartedEvent,
} from "./types";

export type {
  AlbumErrorEvent,
  AlbumMode,
  AlbumStatus,
  FileUploadResult,
  PreflightWarning,
  ResumableBatchInfo,
  ThreadStatus,
  UploadBatchStart,
  UploadConcurrencyStatus,
  UploadFileEntry,
  UploadPreflightProgress,
  UploadQueuedEvent,
  UploadState,
  WatcherBatchStartedEvent,
} from "./types";

export { isAlbumKey };

class UploadManager {
  private static instance: UploadManager;

  public state = reactive<UploadState>({
    isUploading: false,
    isPaused: false,
    totalFiles: 0,
    uploadedFiles: 0,
    threads: new Map<number, ThreadStatus>(),
    results: {
      success: [],
      fail: [],
      skipped: [],
      warnings: [],
    },
    warnings: [],
    totalBytes: 0,
    uploadedBytes: 0,
    startTime: 0,
    endTime: 0,
    uploadSpeed: 0,
    chainId: 0,
    startedByUser: false,
    albumTarget: '',
    albumMode: 'none',
    albumStatus: null,
    albumError: '',
    albumAddedPaths: new Set<string>(),
    fileList: [],
    queuedBatches: 0,
    preflightScanned: 0,
    concurrencyLimit: 0,
    concurrencyMax: 0,
  });

  private static readonly MAX_TRACKED_FILES = 5000;

  private static readonly SPEED_WINDOW_MS = 8000;
  private static readonly SPEED_MIN_SPAN_MS = 2000;

  private static readonly THREAD_FLUSH_MS = 200;
  private speedWindow: Array<{ t: number; b: number }> = [];
  private speedTimer: ReturnType<typeof setInterval> | null = null;
  private pausedAt: number = 0;
  private pausedTotalMs: number = 0;
  private manualStartPending: boolean = false;
  private completedBytes: number = 0;
  private fileBytes: Map<string, number> = new Map();
  private totalBytesAdjustment: number = 0;
  private currentBatchReportedBytes: number = 0;
  private fileIndex: Map<string, UploadFileEntry> = new Map();

  private cancelledPaths: Set<string> = new Set();

  private pendingThreads: Map<number, ThreadStatus> = new Map();
  private threadFlushTimer: ReturnType<typeof setTimeout> | null = null;

  private constructor() {
    this.resetUploadResults = this.resetUploadResults.bind(this);
    this.clearActivity = this.clearActivity.bind(this);
    this.cancelUpload = this.cancelUpload.bind(this);
    this.copyResultsAsJson = this.copyResultsAsJson.bind(this);

    this.setupEventListeners();
  }

  public static getInstance(): UploadManager {
    if (!UploadManager.instance) {
      UploadManager.instance = new UploadManager();
    }
    return UploadManager.instance;
  }

  private chainActive: boolean = false;

  private awaitingBatchPreflightCompletion: boolean = false;

  private albumQueue: Array<{ target: string; mode: AlbumMode }> = [];
  private albumQueuePeekIndex: number = 0;
  private albumQueueConsumeIndex: number = 0;

  private currentBatchAlbum: { target: string; mode: AlbumMode } | undefined;

  private setupEventListeners() {
    Events.On("uploadStart", (event: { data: UploadBatchStart }) => {
      const isNewChain = !this.chainActive;
      this.chainActive = true;

      if (isNewChain) {
        this.resetBatchState();
        this.completedBytes = 0;
        this.fileBytes.clear();
        this.totalBytesAdjustment = 0;
        this.currentBatchReportedBytes = 0;
        this.state.startTime = Date.now();
        this.pausedAt = 0;
        this.pausedTotalMs = 0;
        this.speedWindow = [];
        this.state.startedByUser = this.manualStartPending;
        this.manualStartPending = false;
        this.state.chainId += 1;
      }

      this.state.totalFiles += event.data.Total;
      this.state.totalBytes += event.data.TotalBytes;
      this.state.isUploading = true;
      this.state.isPaused = false;
      this.setPausedClock(false);
      this.state.threads.clear();
      this.clearThreadQueue();
      this.state.endTime = 0;
      this.startSpeedSampler();
      this.state.preflightScanned = 0;
      this.state.concurrencyLimit = 0;
      this.state.concurrencyMax = 0;
      if (event.data.Phase === 'preflight') {
        this.awaitingBatchPreflightCompletion = false;
      }
      const startingBatchAlbum = this.awaitingBatchPreflightCompletion
        ? this.currentBatchAlbum
        : this.albumQueue[this.albumQueueConsumeIndex];
      if (!this.awaitingBatchPreflightCompletion) {
        this.currentBatchAlbum = startingBatchAlbum;
      }
      if (event.data.Files?.length) {
        for (const path of event.data.Files) {
          this.cancelledPaths.delete(path);
          this.upsertFileEntry({
            Path: path,
            Name: baseName(path),
            Status: 'pending',
            AlbumMode: startingBatchAlbum?.mode ?? 'none',
          });
        }
      }
      if (!this.awaitingBatchPreflightCompletion) {
        this.currentBatchReportedBytes = 0;
        this.state.queuedBatches = Math.max(0, this.state.queuedBatches - 1);
        this.state.albumTarget = startingBatchAlbum?.target ?? '';
        this.state.albumMode = startingBatchAlbum?.mode ?? 'none';
        this.albumQueueConsumeIndex += 1;
      }
      this.awaitingBatchPreflightCompletion = !this.awaitingBatchPreflightCompletion;
    });

    Events.On("uploadQueued", (event: { data: UploadQueuedEvent }) => {
      this.state.queuedBatches = event.data.Position;
      if (this.albumQueuePeekIndex < this.albumQueueConsumeIndex) {
        this.albumQueuePeekIndex = this.albumQueueConsumeIndex;
      }
      const queuedBatchAlbum = this.albumQueue[this.albumQueuePeekIndex];
      this.albumQueuePeekIndex += 1;
      if (event.data.Paths?.length) {
        for (const path of event.data.Paths) {
          if (this.fileIndex.has(path)) continue;
          this.cancelledPaths.delete(path);
          this.upsertFileEntry({
            Path: path,
            Name: baseName(path),
            Status: 'queued',
            AlbumMode: queuedBatchAlbum?.mode ?? 'none',
          });
        }
      }
    });

    Events.On("queuedFileCancelled", (event: { data: QueuedFileCancelledEvent }) => {
      const { Path, RemainingQueued } = event.data;
      this.removeFileRow(Path);
      this.state.queuedBatches = RemainingQueued;
    });

    Events.On("uploadPreflightProgress", (event: { data: UploadPreflightProgress }) => {
      this.state.preflightScanned = event.data.Scanned;
    });

    Events.On("uploadConcurrency", (event: { data: UploadConcurrencyStatus }) => {
      this.state.concurrencyLimit = event.data.Limit;
      this.state.concurrencyMax = event.data.Max;
    });

    Events.On("uploadTotalBytes", (event: { data: number }) => {
      const delta = event.data - this.currentBatchReportedBytes;
      this.currentBatchReportedBytes = event.data;
      this.state.totalBytes += delta;
    });

    Events.On("uploadTotalBytesDelta", (event: { data: number }) => {
      this.state.totalBytes += event.data;
    });

    Events.On("uploadWarning", (event: { data: PreflightWarning }) => {
      this.state.warnings.push(event.data);
      if (!isSkippedUploadWarning(event.data.Code)) {
        this.state.results.warnings.push({
          paths: event.data.Paths,
          code: event.data.Code,
          reason: event.data.Message,
        });
      }
    });

    Events.On("ThreadStatus", (event: { data: ThreadStatus }) => {
      const thread = event.data;

      if (thread.Status === 'uploading' && thread.FilePath && thread.BytesTotal > 0) {
        this.fileBytes.set(thread.FilePath, thread.BytesTotal);
      }

      if (thread.Status === 'error') {
        const prevThread = this.state.threads.get(thread.WorkerID);
        if (prevThread?.Message !== thread.Message) {
          window.dispatchEvent(new CustomEvent('uploadError', { detail: thread }));
        }
        this.state.threads.set(thread.WorkerID, thread);
        this.pendingThreads.delete(thread.WorkerID);
        return;
      }

      this.pendingThreads.set(thread.WorkerID, thread);
      this.scheduleThreadFlush();
    });

    Events.On("FileStatus", (event: { data: FileUploadResult }) => {
      const { Path, Paths } = event.data;

      this.recordFileResult(event.data);

      this.state.uploadedFiles += recordUploadResult(this.state.results, event.data);
      if (!event.data.IsError) {
        const completedFileBytes = this.fileBytes.get(Path);
        if (completedFileBytes && completedFileBytes > 0) {
          this.completedBytes += completedFileBytes;
        }
      }
      for (const completedPath of Paths?.length ? Paths : [Path]) {
        this.fileBytes.delete(completedPath);
      }
      this.updateBytesAndSpeed();

      if (
        this.state.totalFiles > 0
        && this.state.queuedBatches === 0
        && this.state.uploadedFiles >= this.state.totalFiles
        && this.state.results.success.length === 0
        && this.state.results.fail.length === 0
      ) {
        this.state.isUploading = false;
        this.markFinished();
      }
    });

    Events.On("uploadStop", () => {
      this.state.isUploading = false;
      this.state.isPaused = false;
      this.manualStartPending = false;
      this.flushPendingThreads();
      this.markFinished();
      this.state.queuedBatches = 0;
      this.chainActive = false;
      this.awaitingBatchPreflightCompletion = false;
      this.resetAlbumQueue();
    });

    const handleAlbumStatus = (name: string) => (event: { data: AlbumStatus }) => {
      this.state.albumStatus = event.data;
      this.state.albumError = '';
      this.markPathsAddedToAlbum(event.data.Paths);
      window.dispatchEvent(new CustomEvent(name, { detail: event.data }));
    };
    Events.On("albumProgress", handleAlbumStatus("albumProgress"));
    Events.On("albumComplete", handleAlbumStatus("albumComplete"));

    Events.On("albumError", (event: { data: AlbumErrorEvent }) => {
      this.state.albumStatus = null;
      this.state.albumError = event.data.Error;
      window.dispatchEvent(new CustomEvent('albumError', { detail: event.data }));
    });

    Events.On("uploadPauseState", (event: { data: boolean }) => {
      this.state.isPaused = event.data;
      this.setPausedClock(event.data);
      if (event.data) {
        this.speedWindow = [];
        this.state.uploadSpeed = 0;
      }
    });

    const enqueueAlbum = (event: { data: WatcherBatchStartedEvent }) => {
      this.enqueueAlbumForTarget(event.data.Album ?? '');
    };
    Events.On("watcherBatchStarted", enqueueAlbum);
    Events.On("retryBatchStarted", enqueueAlbum);

    Events.On("uploadAutoRetry", (event: { data: number }) => {
      window.dispatchEvent(new CustomEvent('uploadAutoRetry', { detail: { count: event.data } }));
    });
  }

  private scheduleThreadFlush() {
    if (this.threadFlushTimer !== null) return;
    this.threadFlushTimer = setTimeout(() => {
      this.threadFlushTimer = null;
      this.flushPendingThreads();
    }, UploadManager.THREAD_FLUSH_MS);
  }

  private clearThreadQueue() {
    if (this.threadFlushTimer !== null) {
      clearTimeout(this.threadFlushTimer);
      this.threadFlushTimer = null;
    }
    this.pendingThreads.clear();
  }

  private flushPendingThreads() {
    if (this.pendingThreads.size === 0) return;
    const batch = Array.from(this.pendingThreads.values());
    this.pendingThreads.clear();

    for (const thread of batch) {
      if (
        thread.FilePath
        && thread.FileName
        && ['hashing', 'checking', 'uploading', 'finalizing'].includes(thread.Status)
      ) {
        const settled = this.fileIndex.get(thread.FilePath);
        const isSettled = settled
          && (settled.Status === 'success' || settled.Status === 'failed' || settled.Status === 'skipped');
        if (!isSettled) {
          this.upsertFileEntry({
            Path: thread.FilePath,
            Name: thread.FileName,
            Status: 'uploading',
            Message: thread.Status === 'uploading' ? undefined : thread.Message,
            BytesUploaded: thread.Status === 'uploading' ? thread.BytesUploaded : undefined,
            BytesTotal: thread.Status === 'uploading' ? thread.BytesTotal : undefined,
          });
        }
      }

      this.state.threads.set(thread.WorkerID, thread);
    }

    this.updateBytesAndSpeed();
  }

  private updateBytesAndSpeed() {
    let activeUploadedBytes = 0;

    this.state.threads.forEach((thread) => {
      if (thread.Status === 'uploading' && thread.BytesTotal > 0) {
        activeUploadedBytes += thread.BytesUploaded;
      } else if (thread.Status === 'finalizing' && thread.FilePath) {
        activeUploadedBytes += this.fileBytes.get(thread.FilePath) ?? 0;
      }
    });

    const totalUploaded = this.completedBytes + activeUploadedBytes;
    this.state.uploadedBytes = totalUploaded;
  }

  private setPausedClock(paused: boolean) {
    const now = Date.now();
    if (paused && !this.pausedAt) {
      this.pausedAt = now;
    } else if (!paused && this.pausedAt) {
      this.pausedTotalMs += now - this.pausedAt;
      this.pausedAt = 0;
    }
  }

  private markFinished() {
    this.setPausedClock(false);
    this.state.endTime = Date.now();
    this.stopSpeedSampler();
    this.state.uploadSpeed = 0;
  }

  private startSpeedSampler() {
    if (this.speedTimer) return;
    this.speedTimer = setInterval(() => this.sampleSpeed(), 1000);
  }

  private stopSpeedSampler() {
    if (this.speedTimer) {
      clearInterval(this.speedTimer);
      this.speedTimer = null;
    }
    this.speedWindow = [];
  }

  private sampleSpeed() {
    if (this.state.isPaused) {
      this.state.uploadSpeed = 0;
      return;
    }
    const now = Date.now();
    this.speedWindow.push({ t: now, b: this.state.uploadedBytes });
    while (
      this.speedWindow.length > 2
      && now - this.speedWindow[0].t > UploadManager.SPEED_WINDOW_MS
    ) {
      this.speedWindow.shift();
    }
    const first = this.speedWindow[0];
    const span = now - first.t;
    if (span < UploadManager.SPEED_MIN_SPAN_MS) {
      this.state.uploadSpeed = 0;
      return;
    }
    this.state.uploadSpeed = Math.max(0, (this.state.uploadedBytes - first.b) / (span / 1000));
  }

  public elapsedSeconds(): number {
    if (!this.state.startTime) return 0;
    const now = this.state.endTime || Date.now();
    const pausedNow = this.pausedAt ? now - this.pausedAt : 0;
    return Math.max(0, Math.floor((now - this.state.startTime - this.pausedTotalMs - pausedNow) / 1000));
  }

  public enqueueAlbumTarget(target: string, mode: AlbumMode = 'none') {
    this.manualStartPending = true;
    this.albumQueue.push({ target, mode });
  }

  private resetAlbumQueue() {
    this.albumQueue = [];
    this.albumQueuePeekIndex = 0;
    this.albumQueueConsumeIndex = 0;
  }

  private enqueueAlbumForTarget(album: string) {
    const mode: AlbumMode = !album
      ? 'none'
      : isAlbumKey(album)
        ? 'existing'
        : 'new';
    this.albumQueue.push({ target: album, mode });
  }

  public startBatch(files: string[], album: string, mode?: AlbumMode) {
    if (mode !== undefined) {
      this.enqueueAlbumTarget(album, mode);
    } else {
      this.enqueueAlbumForTarget(album);
    }
    Events.Emit('startUpload', { files, album });
  }

  private markPathsAddedToAlbum(paths?: string[]) {
    if (!paths?.length) return;
    for (const path of paths) {
      this.state.albumAddedPaths.add(path);
    }
  }

  private upsertFileEntry(entry: UploadFileEntry) {
    const existing = this.fileIndex.get(entry.Path);
    if (existing) {
      Object.assign(existing, entry);
    } else {
      this.state.fileList.push(entry);
      this.fileIndex.set(entry.Path, this.state.fileList[this.state.fileList.length - 1]);
    }
    this.pruneFileListIfNeeded();
  }

  private removeFileRow(path: string) {
    this.fileIndex.delete(path);
    const idx = this.state.fileList.findIndex((f) => f.Path === path);
    if (idx !== -1) {
      this.state.fileList.splice(idx, 1);
    }
  }

  private pruneFileListIfNeeded() {
    const overflow = this.state.fileList.length - UploadManager.MAX_TRACKED_FILES;
    if (overflow <= 0) return;

    let removed = 0;
    let i = 0;
    while (i < this.state.fileList.length && removed < overflow) {
      const status = this.state.fileList[i].Status;
      if (status === 'success' || status === 'failed' || status === 'skipped') {
        this.fileIndex.delete(this.state.fileList[i].Path);
        this.state.fileList.splice(i, 1);
        removed++;
      } else {
        i++;
      }
    }
  }

  private recordFileResult(event: FileUploadResult) {
    const path = event.Path;
    if (!path) return;

    if (this.cancelledPaths.has(path)) return;

    const existing = this.fileIndex.get(path);
    const name = existing?.Name || baseName(path);
    let status = 'success';
    let message: string | undefined;
    if (event.IsError) {
      status = 'failed';
      message = event.ErrorMessage;
    } else if (event.Skipped) {
      status = 'skipped';
      message = event.SkipReason;
    }

    this.upsertFileEntry({
      Path: path,
      Name: name,
      Status: status,
      Message: message,
      BytesUploaded: undefined,
      BytesTotal: undefined,
    });
  }

  public resetUploadResults(): void {
    this.state.results.success = [];
    this.state.results.fail = [];
    this.state.results.skipped = [];
    this.state.results.warnings = [];
  }

  public clearActivity() {
    if (this.state.isUploading) return;
    this.chainActive = false;
    this.resetAlbumQueue();
    this.resetBatchState();
    this.clearThreadQueue();
    this.state.threads.clear();
    this.state.startTime = 0;
    this.state.endTime = 0;
    this.stopSpeedSampler();
    this.state.queuedBatches = 0;
    this.state.preflightScanned = 0;
    this.state.concurrencyLimit = 0;
    this.state.concurrencyMax = 0;
  }

  private resetBatchState() {
    this.resetUploadResults();
    this.state.warnings = [];
    this.state.albumStatus = null;
    this.state.albumError = '';
    this.state.albumAddedPaths = new Set<string>();
    this.state.fileList = [];
    this.fileIndex.clear();
    this.cancelledPaths.clear();
    this.state.totalFiles = 0;
    this.state.uploadedFiles = 0;
    this.state.totalBytes = 0;
    this.state.uploadedBytes = 0;
    this.state.uploadSpeed = 0;
  }

  public cancelUpload() {
    Events.Emit("uploadCancel");
  }

  private dropFile(path: string, eventName: 'skipFile' | 'cancelQueuedFile') {
    this.cancelledPaths.add(path);
    this.removeFileRow(path);
    Events.Emit(eventName, { path });
  }

  public skipFile(path: string) {
    this.dropFile(path, 'skipFile');
  }

  public cancelQueuedFile(path: string) {
    this.dropFile(path, 'cancelQueuedFile');
  }

  public retryFailed() {
    if (this.state.isUploading) return;
    const failedPaths = this.state.fileList
      .filter((f) => f.Status === 'failed')
      .map((f) => f.Path);
    if (!failedPaths.length) return;
    this.startBatch(failedPaths, this.state.albumTarget, this.state.albumMode);
  }

  public pauseUpload() {
    Events.Emit("uploadPause");
  }

  public resumeUpload() {
    Events.Emit("uploadResume");
  }

  public async copyResultsAsJson() {
    const resultsJson = JSON.stringify(this.state.results, null, 2);
    try {
      await Clipboard.SetText(resultsJson);
      console.log("Upload results copied to clipboard");
      return true;
    } catch (error) {
      console.error("Failed to copy results:", error);
      return false;
    }
  }
}

export const uploadManager = UploadManager.getInstance();
