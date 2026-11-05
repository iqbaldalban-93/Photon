<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import Button from '../components/ui/button/Button.vue'
import { Progress } from '../components/ui/progress'
import { ScrollArea } from '../components/ui/scroll-area'
import EmptyState from '../components/app/EmptyState.vue'
import ProgressRing from '../components/app/ProgressRing.vue'
import { Clipboard } from '@wailsio/runtime'
import { uploadManager, type UploadFileEntry, type ThreadStatus } from '../upload/manager'
import { formatBytes, formatSeconds } from '../utils/format'
import {
  X, Clock, AlertTriangle, Pause, Play, Images, CircleCheck, CircleX,
  Loader2, CloudUpload, Trash2, Copy, RotateCcw, Gauge,
} from '@lucide/vue'

const { state } = uploadManager
const copyButtonText = ref('Copy as JSON')

function handleCopyClick() {
  uploadManager.copyResultsAsJson()
  copyButtonText.value = 'Copied!'
  setTimeout(() => copyButtonText.value = 'Copy as JSON', 1000)
}

const elapsedSeconds = ref(0)
let elapsedInterval: ReturnType<typeof setInterval> | null = null

onMounted(() => {
  const tick = () => { elapsedSeconds.value = uploadManager.elapsedSeconds() }
  tick()
  elapsedInterval = setInterval(tick, 1000)
})

onUnmounted(() => {
  if (elapsedInterval) {
    clearInterval(elapsedInterval)
  }
})

const threadsList = computed(() => {
  return Array.from(state.threads.values())
    .filter(thread => thread.Status !== 'idle')
    .sort((a, b) => a.WorkerID - b.WorkerID)
})

const progressPercent = computed(() => {
  if (state.totalBytes > 0) {
    return Math.min(100, Math.round((state.uploadedBytes / state.totalBytes) * 100))
  }
  if (state.totalFiles === 0) return 0
  return Math.round((state.uploadedFiles / state.totalFiles) * 100)
})

const speedDisplay = computed(() => {
  if (state.uploadSpeed <= 0) return '--'
  return formatBytes(state.uploadSpeed) + '/s'
})

const elapsedDisplay = computed(() => formatSeconds(elapsedSeconds.value))

const etaDisplay = computed(() => {
  if (state.uploadSpeed <= 0) return '--'
  const remaining = state.totalBytes - state.uploadedBytes
  if (remaining <= 0) return '--'
  return formatSeconds(Math.round(remaining / state.uploadSpeed))
})

const bytesDisplay = computed(() => {
  if (state.totalBytes === 0) return ''
  return `${formatBytes(state.uploadedBytes)} / ${formatBytes(state.totalBytes)}`
})

function warningFiles(paths: string[]): string {
  return paths.map(path => path.split(/[\\/]/).pop()).filter(Boolean).join(', ')
}

const MAX_VISIBLE_FILES = 2000
const displayFiles = computed<UploadFileEntry[]>(() => {
  const active: UploadFileEntry[] = []
  const attention: UploadFileEntry[] = []
  const pending: UploadFileEntry[] = []
  const queued: UploadFileEntry[] = []
  const done: UploadFileEntry[] = []
  for (const file of state.fileList) {
    switch (file.Status) {
      case 'uploading': active.push(file); break
      case 'failed':
      case 'skipped': attention.push(file); break
      case 'pending': pending.push(file); break
      case 'queued': queued.push(file); break
      default: done.push(file)
    }
  }
  done.reverse()
  const ordered = [...active, ...attention, ...pending, ...queued, ...done]
  return ordered.length > MAX_VISIBLE_FILES ? ordered.slice(0, MAX_VISIBLE_FILES) : ordered
})

const displayTotal = computed(() => {
  let queued = 0
  for (const file of state.fileList) {
    if (file.Status === 'queued') queued++
  }
  return state.totalFiles + queued
})

const filesTruncated = computed(() => state.fileList.length > MAX_VISIBLE_FILES)

function filePercent(file: UploadFileEntry): number {
  if (!file.BytesTotal) return 0
  return Math.min(100, Math.round((file.BytesUploaded || 0) / file.BytesTotal * 100))
}

function workerPercent(thread: ThreadStatus): number | null {
  if (thread.Status !== 'uploading' || !thread.BytesTotal) return null
  return Math.min(100, Math.round((thread.BytesUploaded / thread.BytesTotal) * 100))
}

function fileMessage(file: UploadFileEntry): string {
  const msg = file.Message || ''
  return msg.replace(/^Error:\s*/, '')
}

function fileStatusIcon(file: UploadFileEntry) {
  if (file.Status === 'success') return CircleCheck
  if (file.Status === 'failed') return CircleX
  if (file.Status === 'skipped') return Clock
  if (file.Status === 'pending') return Clock
  if (file.Status === 'queued') return Clock
  return Loader2
}

function fileStatusClass(file: UploadFileEntry): string {
  if (file.Status === 'success') return 'text-success'
  if (file.Status === 'failed') return 'text-destructive'
  if (file.Status === 'skipped') return 'text-warning'
  if (file.Status === 'pending') return 'text-muted-foreground'
  if (file.Status === 'queued') return 'text-muted-foreground'
  if (state.isPaused) return 'text-muted-foreground'
  return 'text-primary spin-slow'
}

function fileHasAlbum(file: UploadFileEntry): boolean {
  return file.AlbumMode === 'new' || file.AlbumMode === 'existing'
}

function fileInAlbum(file: UploadFileEntry): boolean {
  return state.albumAddedPaths.has(file.Path)
}

function fileAlbumClass(file: UploadFileEntry): string {
  return fileInAlbum(file) ? 'text-spark' : 'text-muted-foreground/40'
}

function fileAlbumTitle(file: UploadFileEntry): string {
  if (fileInAlbum(file)) return 'Added to album'
  if (file.Status === 'success') return 'Not added to album yet'
  return 'Will be added to album once uploaded'
}

const anyFileHasAlbum = computed(() => state.fileList.some(fileHasAlbum))

const albumKeysDisplay = computed(() => {
  const status = state.albumStatus
  if (!status?.IsComplete || !status.AlbumKeys?.length) return ''
  return status.AlbumKeys.join(', ')
})

function cancelFile(file: UploadFileEntry) {
  if (file.Status === 'pending') {
    uploadManager.skipFile(file.Path)
  } else if (file.Status === 'queued') {
    uploadManager.cancelQueuedFile(file.Path)
  }
}

function cancelFileTitle(file: UploadFileEntry): string {
  return file.Status === 'queued' ? 'Remove from queue' : 'Skip this file'
}

async function copyAlbumKeys() {
  if (!albumKeysDisplay.value) return
  try {
    await Clipboard.SetText(albumKeysDisplay.value)
  } catch (error) {
    console.error('Failed to copy album keys:', error)
  }
}

const headerTitle = computed(() => {
  if (state.isUploading) return state.isPaused ? 'Paused' : 'Uploading'
  return state.results.fail.length ? 'Done, with a few misses' : 'All done'
})

const ringTone = computed<'primary' | 'success' | 'muted'>(() => {
  if (!state.isUploading) return 'success'
  return state.isPaused ? 'muted' : 'primary'
})

const isScanning = computed(() => state.isUploading && state.totalFiles === 0)
</script>

<template>
  <div
    class="mx-auto flex h-full w-full max-w-xl sm:max-w-2xl flex-col px-4 pt-4 pb-2"
    style="--wails-draggable: none"
  >
    <EmptyState
      v-if="!state.isUploading && state.fileList.length === 0"
      :icon="CloudUpload"
      title="Nothing uploading"
      hint="Choose files or a folder on Home and the progress shows up here."
    />

    <template v-else>
      <div class="mb-2.5 flex items-center justify-between gap-2">
        <h1 class="font-display truncate text-2xl font-bold">
          {{ headerTitle }}
        </h1>
        <span class="shrink-0 rounded-full bg-surface px-3 py-1 text-[13px] font-semibold tabular-nums">
          {{ state.uploadedFiles }}<span class="text-muted-foreground"> of {{ displayTotal }}</span>
        </span>
      </div>

      <div class="mb-2.5 flex items-center gap-4 rounded-3xl bg-surface p-3.5">
        <ProgressRing
          :value="isScanning ? 0 : progressPercent"
          :size="104"
          :stroke="10"
          :live="state.isUploading && !state.isPaused && !isScanning"
          :tone="ringTone"
        >
          <CircleCheck
            v-if="!state.isUploading"
            :size="40"
            class="text-success"
            :stroke-width="1.8"
          />
          <template v-else-if="isScanning">
            <Loader2
              :size="28"
              class="spin-slow text-primary"
            />
          </template>
          <template v-else>
            <span class="font-display text-[30px] leading-none font-bold tabular-nums">{{ progressPercent }}<span class="text-base text-muted-foreground">%</span></span>
          </template>
        </ProgressRing>

        <div class="min-w-0 flex-1">
          <p
            v-if="isScanning"
            class="text-sm leading-snug"
          >
            {{ state.preflightScanned > 0 ? `Scanning… ${state.preflightScanned} found so far` : 'Scanning your selection…' }}
          </p>
          <dl
            v-else-if="state.isUploading"
            class="grid grid-cols-2 gap-x-3 gap-y-1.5"
          >
            <div class="col-span-2">
              <dt class="text-xs text-muted-foreground">
                Speed
              </dt>
              <dd class="font-display text-2xl leading-tight font-bold tabular-nums">
                {{ speedDisplay }}
              </dd>
            </div>
            <div>
              <dt class="text-xs text-muted-foreground">
                Time left
              </dt>
              <dd class="text-[15px] font-semibold tabular-nums">
                {{ etaDisplay }}
              </dd>
            </div>
            <div>
              <dt class="text-xs text-muted-foreground">
                Elapsed
              </dt>
              <dd class="text-[15px] font-semibold tabular-nums">
                {{ elapsedDisplay }}
              </dd>
            </div>
            <div
              v-if="bytesDisplay"
              class="col-span-2 truncate text-xs text-muted-foreground tabular-nums"
            >
              {{ bytesDisplay }}
            </div>
          </dl>

          <div
            v-else
            class="grid grid-cols-2 gap-x-3 gap-y-2"
          >
            <div>
              <p class="font-display text-2xl leading-none font-bold tabular-nums text-success">
                {{ state.results.success.length }}
              </p>
              <p class="mt-0.5 text-xs text-muted-foreground">
                Uploaded
              </p>
            </div>
            <div>
              <p
                class="font-display text-2xl leading-none font-bold tabular-nums"
                :class="state.results.fail.length ? 'text-destructive' : 'text-muted-foreground'"
              >
                {{ state.results.fail.length }}
              </p>
              <p class="mt-0.5 text-xs text-muted-foreground">
                Failed
              </p>
            </div>
            <div>
              <p
                class="font-display text-2xl leading-none font-bold tabular-nums"
                :class="state.results.skipped.length ? 'text-warning' : 'text-muted-foreground'"
              >
                {{ state.results.skipped.length }}
              </p>
              <p class="mt-0.5 text-xs text-muted-foreground">
                Skipped
              </p>
            </div>
            <div>
              <p
                class="font-display text-2xl leading-none font-bold tabular-nums"
                :class="state.results.warnings.length ? 'text-warning' : 'text-muted-foreground'"
              >
                {{ state.results.warnings.length }}
              </p>
              <p class="mt-0.5 text-xs text-muted-foreground">
                Warnings
              </p>
            </div>
          </div>

          <p
            v-if="state.isUploading && state.concurrencyMax > 0 && state.concurrencyLimit < state.concurrencyMax"
            class="mt-2 inline-flex items-center gap-1.5 rounded-full bg-warning/15 px-2.5 py-0.5 text-xs text-warning"
            title="Automatically slowed down due to repeated retries"
          >
            <Gauge :size="13" />
            Slowed to {{ state.concurrencyLimit }} of {{ state.concurrencyMax }} at once
          </p>
        </div>
      </div>

      <div
        v-if="!state.isUploading"
        class="mb-3 flex flex-wrap items-center justify-end gap-2"
      >
        <Button
          v-if="state.results.fail.length"
          size="sm"
          title="Retry failed files"
          @click="() => uploadManager.retryFailed()"
        >
          <RotateCcw :size="14" />
          Retry {{ state.results.fail.length }} failed
        </Button>
        <Button
          variant="secondary"
          size="sm"
          class="min-w-[104px]"
          @click="handleCopyClick"
        >
          {{ copyButtonText }}
        </Button>
        <Button
          variant="ghost"
          size="sm"
          title="Clear activity"
          @click="uploadManager.clearActivity()"
        >
          <Trash2 :size="14" />
          Clear
        </Button>
      </div>

      <div
        v-if="state.warnings.length"
        class="mb-3 space-y-1.5"
      >
        <div
          v-for="(warning, index) in state.warnings"
          :key="`${warning.Code}-${index}`"
          class="flex gap-2.5 rounded-2xl bg-warning/12 px-3 py-2.5 text-warning ring-1 ring-warning/30"
        >
          <AlertTriangle
            :size="16"
            class="mt-0.5 shrink-0"
          />
          <div class="min-w-0">
            <p class="text-[13px] leading-snug text-foreground">
              {{ warning.Message }}
            </p>
            <p class="truncate text-xs opacity-80">
              {{ warningFiles(warning.Paths) }}
            </p>
          </div>
        </div>
      </div>

      <div
        v-if="state.albumTarget"
        class="mb-2.5 flex gap-3 rounded-2xl bg-surface px-3.5 py-2.5"
      >
        <Images
          :size="16"
          class="mt-0.5 shrink-0 text-spark"
        />
        <div class="min-w-0 flex-1">
          <p class="text-[13px] leading-snug">
            <template v-if="state.albumStatus && !state.albumError">
              {{ state.albumStatus.IsComplete ? 'Album updated:' : 'Adding to album:' }}
              <span class="font-semibold">{{ state.albumStatus.AlbumName }}</span>
            </template>
            <template v-else-if="state.albumError">
              Album failed
            </template>
            <template v-else>
              Waiting for uploads to finish…
            </template>
          </p>
          <p
            v-if="state.albumStatus"
            class="truncate text-xs text-muted-foreground"
          >
            {{ state.albumStatus.ItemsAdded }} of {{ state.albumStatus.TotalItems }} items added<template v-if="!state.albumStatus.IsComplete">
              …
            </template>
          </p>
          <div
            v-if="albumKeysDisplay"
            class="mt-1 flex items-center gap-1.5"
          >
            <span
              class="min-w-0 truncate font-mono text-xs text-primary"
              :title="albumKeysDisplay"
            >{{ albumKeysDisplay }}</span>
            <button
              type="button"
              class="shrink-0 cursor-pointer select-none text-muted-foreground transition-colors hover:text-foreground"
              title="Copy album key"
              @click="copyAlbumKeys"
            >
              <Copy :size="13" />
            </button>
          </div>
          <p
            v-else-if="state.albumError"
            class="truncate text-xs text-destructive"
          >
            {{ state.albumError }}
          </p>
        </div>
      </div>

      <div
        v-if="threadsList.length"
        class="-mx-4 mb-2.5 flex gap-2 overflow-x-auto px-4"
      >
        <div
          v-for="thread in threadsList"
          :key="thread.WorkerID"
          class="relative w-[148px] shrink-0 overflow-hidden rounded-2xl bg-surface px-3 pt-2 pb-2.5"
        >
          <p class="flex items-baseline justify-between gap-2 text-xs text-muted-foreground">
            <span>Lane {{ thread.WorkerID + 1 }}</span>
            <span
              v-if="workerPercent(thread) !== null"
              class="font-semibold tabular-nums text-foreground"
            >{{ workerPercent(thread) }}%</span>
          </p>
          <p class="mt-0.5 truncate text-[13px]">
            {{ thread.FileName || thread.Message }}
          </p>
          <span
            class="absolute inset-x-0 bottom-0 h-1 bg-accent/60"
          >
            <span
              class="block h-full bg-primary transition-[width] duration-300"
              :style="{ width: `${workerPercent(thread) ?? 0}%` }"
            />
          </span>
        </div>
      </div>

      <div class="mb-2 flex min-h-0 flex-1 flex-col">
        <p class="mb-2 flex items-center gap-3 text-xs text-muted-foreground">
          <span class="font-display text-sm font-semibold text-foreground">Files</span>
          <span class="flex items-center gap-1 tabular-nums text-success">
            <CircleCheck :size="13" />
            {{ state.uploadedFiles }}
          </span>
          <span class="flex items-center gap-1 tabular-nums text-destructive">
            <CircleX :size="13" />
            {{ state.results.fail.length }}
          </span>
          <span class="flex items-center gap-1 tabular-nums text-warning">
            <Clock :size="13" />
            {{ state.results.skipped.length }}
          </span>
          <span
            v-if="anyFileHasAlbum"
            class="ml-auto flex items-center gap-1"
            title="The picture icon on each row shows whether that file has been added to the album yet"
          >
            <Images
              :size="13"
              class="text-spark"
            />
            in album
            <Images
              :size="13"
              class="text-muted-foreground/40"
            />
            not yet
          </span>
        </p>
        <ScrollArea class="min-h-0 flex-1">
          <div class="grid grid-cols-1 items-start gap-1 pr-3 md:grid-cols-2">
            <div
              v-for="file in displayFiles"
              :key="file.Path"
              class="flex items-center gap-3 rounded-2xl bg-surface px-3 py-2"
            >
              <component
                :is="fileStatusIcon(file)"
                :size="16"
                class="shrink-0"
                :class="fileStatusClass(file)"
              />
              <div class="min-w-0 flex-1">
                <div class="flex items-center justify-between gap-2">
                  <span
                    class="truncate text-[13px]"
                    :title="file.Path"
                  >{{ file.Name }}</span>
                  <span
                    v-if="file.Status === 'uploading' && file.BytesTotal"
                    class="shrink-0 text-xs font-semibold tabular-nums"
                  >
                    {{ filePercent(file) }}%
                  </span>
                </div>
                <template v-if="file.Status === 'uploading' && file.BytesTotal">
                  <Progress
                    :model-value="filePercent(file)"
                    class="mt-1.5 h-1"
                  />
                </template>
                <p
                  v-else-if="file.Status === 'uploading' && file.Message"
                  class="truncate text-xs text-muted-foreground"
                >
                  {{ file.Message }}
                </p>
                <p
                  v-else-if="file.Status === 'failed' || file.Status === 'skipped'"
                  class="truncate text-xs text-muted-foreground"
                >
                  {{ fileMessage(file) }}
                </p>
                <p
                  v-else-if="file.Status === 'pending'"
                  class="truncate text-xs text-muted-foreground"
                >
                  Waiting
                </p>
                <p
                  v-else-if="file.Status === 'queued'"
                  class="truncate text-xs text-muted-foreground"
                >
                  In line for the next batch
                </p>
              </div>
              <Images
                v-if="fileHasAlbum(file) && file.Status !== 'failed' && file.Status !== 'skipped'"
                :size="14"
                class="shrink-0"
                :class="fileAlbumClass(file)"
                :title="fileAlbumTitle(file)"
              />
              <button
                v-if="file.Status === 'pending' || file.Status === 'queued'"
                type="button"
                class="shrink-0 cursor-pointer select-none rounded-full p-1 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                :title="cancelFileTitle(file)"
                @click="() => cancelFile(file)"
              >
                <X :size="14" />
              </button>
            </div>
          </div>
        </ScrollArea>
        <p
          v-if="filesTruncated"
          class="mt-1 text-xs text-muted-foreground"
        >
          Showing the first {{ MAX_VISIBLE_FILES }} of {{ state.fileList.length }} files
        </p>
      </div>

      <div
        v-if="state.isUploading"
        class="flex gap-2 pt-1"
      >
        <Button
          class="flex-1"
          :variant="state.isPaused ? 'default' : 'secondary'"
          @click="() => state.isPaused ? uploadManager.resumeUpload() : uploadManager.pauseUpload()"
        >
          <Play
            v-if="state.isPaused"
            :size="15"
          />
          <Pause
            v-else
            :size="15"
          />
          {{ state.isPaused ? 'Resume' : 'Pause' }}
        </Button>
        <Button
          variant="destructive"
          class="flex-1"
          @click="() => uploadManager.cancelUpload()"
        >
          <X :size="15" />
          Cancel
        </Button>
      </div>
    </template>
  </div>
</template>
