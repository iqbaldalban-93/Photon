import type { UploadResultEvent, UploadResults } from "../utils/uploadResults";

// Event payload shapes crossing the Wails boundary. The JSON field names are
// the contract with the Go backend (PascalCase) and must not change.

export interface ThreadStatus {
  WorkerID: number;
  Status: string;
  FilePath: string;
  FileName: string;
  Message: string;
  BytesUploaded: number;
  BytesTotal: number;
  Attempt: number;
}

export type FileUploadResult = UploadResultEvent;

export interface PreflightWarning {
  Paths: string[];
  Code: string;
  Message: string;
}

export interface UploadBatchStart {
  Total: number;
  TotalBytes: number;
  Files?: string[];
  Phase?: string;
}

export interface UploadPreflightProgress {
  Scanned: number;
}

export interface UploadConcurrencyStatus {
  Limit: number;
  Max: number;
}

export interface ResumableBatchInfo {
  Files: string[];
  Album: string;
  Total: number;
}

export interface WatcherBatchStartedEvent {
  Files?: string[];
  Album: string;
}

export interface UploadQueuedEvent {
  Position: number;
  Total: number;
  Paths?: string[];
}

export interface QueuedFileCancelledEvent {
  Path: string;
  RemainingQueued: number;
}

export interface AlbumStatus {
  AlbumName: string;
  ItemsAdded: number;
  TotalItems: number;
  AlbumKeys: string[];
  IsComplete: boolean;
  Paths?: string[];
}

export interface AlbumErrorEvent {
  AlbumName: string;
  Error: string;
  IsError: boolean;
}

export type AlbumMode = 'none' | 'new' | 'existing';

export interface UploadFileEntry {
  Path: string;
  Name: string;
  Status: string;
  Message?: string;
  BytesUploaded?: number;
  BytesTotal?: number;
  AlbumMode?: AlbumMode;
}

export interface UploadState {
  isUploading: boolean;
  isPaused: boolean;
  totalFiles: number;
  uploadedFiles: number;
  threads: Map<number, ThreadStatus>;
  results: UploadResults;
  warnings: PreflightWarning[];
  totalBytes: number;
  uploadedBytes: number;
  startTime: number;
  endTime: number;
  uploadSpeed: number;
  chainId: number;
  startedByUser: boolean;
  albumTarget: string;
  albumMode: AlbumMode;
  albumStatus: AlbumStatus | null;
  albumError: string;
  albumAddedPaths: Set<string>;
  fileList: UploadFileEntry[];
  queuedBatches: number;
  preflightScanned: number;
  concurrencyLimit: number;
  concurrencyMax: number;
}

export function isSkippedUploadWarning(code: string): boolean {
  return code === "ambiguous-filename-stem";
}

export function isAlbumKey(target: string): boolean {
  return target.length > 6 && target.startsWith("AF1Qip");
}
