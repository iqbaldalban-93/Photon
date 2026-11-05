<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { Events } from '@wailsio/runtime'
import {
  ArrowUp,
  CircleCheck,
  CirclePause,
  CirclePlay,
  Check,
  ChevronDown,
  Film,
  FolderOpen,
  FolderX,
  Globe,
  HardDrive,
  Image as ImageIcon,
  Power,
  RefreshCw,
  TriangleAlert,
  X,
} from '@lucide/vue'
import AppLogo from '@/components/app/AppLogo.vue'
import ProgressRing from '@/components/app/ProgressRing.vue'
import { formatBytes } from '@/utils/format'

interface TrayState {
  statusLine: string
  libraryLine: string
  videoLine: string
  icon: 'idle' | 'active' | 'paused' | 'error'
  uploadActive: boolean
  uploadPaused: boolean
  watchAvailable: boolean
  watchPaused: boolean
  libraryRunning: boolean
  librarySyncing: boolean
  libraryUrl: string
  accounts: { email: string; active: boolean }[]
  activeAccount: string
  webUploading: boolean
  speedBps: number
  phase: '' | 'scanning' | 'uploading'
  done: number
  total: number
  percent: number
  failed: number
  items: number
  videos: number
  photos: number
  allVideos: boolean
  totalSize: string
  videoSize: string
  photoSize: string
}

const state = ref<TrayState | null>(null)
let off: (() => void) | undefined

const fmt = (n: number) => n.toLocaleString('en-US')
const act = (name: string) => Events.Emit('trayAction', name)

onMounted(() => {
  off = Events.On('trayState', (e: { data: TrayState }) => {
    state.value = e.data
  })
  Events.Emit('trayStateRequest')
  window.addEventListener('focus', requestState)
})
onBeforeUnmount(() => {
  off?.()
  window.removeEventListener('focus', requestState)
})
function requestState() {
  Events.Emit('trayStateRequest')
  pickerOpen.value = false
  pendingEmail.value = null
}

const s = computed(() => state.value)

const pickerOpen = ref(false)
const pendingEmail = ref<string | null>(null)
const hasChoice = computed(() => (s.value?.accounts.length ?? 0) > 1)

function togglePicker() {
  if (!hasChoice.value) return
  pickerOpen.value = !pickerOpen.value
  pendingEmail.value = null
}

function pickAccount(email: string, active: boolean) {
  if (active) {
    pickerOpen.value = false
    return
  }
  if (s.value?.webUploading) {
    pendingEmail.value = email
    return
  }
  switchTo(email, false)
}

function switchTo(email: string, force: boolean) {
  act((force ? 'switchAccountForce:' : 'switchAccount:') + email)
  pickerOpen.value = false
  pendingEmail.value = null
}

const heading = computed(() => {
  const v = s.value
  if (!v) return 'Loading…'
  if (v.uploadPaused) return 'Uploads paused'
  if (v.phase === 'scanning') return 'Scanning files'
  if (v.uploadActive) return 'Uploading'
  if (v.librarySyncing) return 'Syncing library'
  if (v.icon === 'error') return 'Needs attention'
  if (v.watchPaused) return 'Watching paused'
  return 'All caught up'
})

const subline = computed(() => {
  const v = s.value
  if (!v) return ''
  if (v.uploadActive && v.phase !== 'scanning' && v.total > 0) {
    const failed = v.failed ? ` · ${v.failed} failed` : ''
    return `${fmt(v.done)} of ${fmt(v.total)} files${failed}`
  }
  return v.statusLine
})

const speedText = computed(() => {
  const v = s.value
  if (!v || !v.uploadActive || v.uploadPaused || v.phase === 'scanning') return ''
  return v.speedBps > 0 ? `${formatBytes(v.speedBps)}/s` : '--'
})

const tone = computed<'primary' | 'success' | 'muted'>(() => {
  const v = s.value
  if (!v) return 'muted'
  if (v.uploadPaused || v.icon === 'error') return 'muted'
  if (v.uploadActive || v.librarySyncing) return 'primary'
  return 'success'
})

const showProgress = computed(() => !!s.value && s.value.uploadActive && s.value.phase !== 'scanning' && s.value.total > 0)
const ringValue = computed(() => (showProgress.value ? s.value!.percent : s.value?.uploadActive || s.value?.librarySyncing ? 30 : 100))
const ringLive = computed(() => !!s.value && (s.value.uploadActive || s.value.librarySyncing) && !s.value.uploadPaused)

const pill = computed(() => {
  const icon = s.value?.icon ?? 'idle'
  return {
    idle: { label: 'Idle', dot: 'bg-muted-foreground' },
    active: { label: 'Working', dot: 'bg-foreground animate-pulse' },
    paused: { label: 'Paused', dot: 'bg-warning' },
    error: { label: 'Attention', dot: 'bg-destructive' },
  }[icon]
})
</script>

<template>
  <main
    class="flex h-screen w-screen flex-col gap-3 overflow-hidden border border-white/10 bg-background p-4 select-none"
    style="--wails-draggable: none"
  >
    <header class="flex items-center gap-2.5">
      <AppLogo :size="26" />
      <span class="font-display text-lg font-semibold">Photon</span>
      <span class="ml-auto inline-flex items-center gap-1.5 rounded-full border border-border bg-surface px-2.5 py-1 text-xs text-muted-foreground">
        <span
          class="size-1.5 rounded-full"
          :class="pill.dot"
        />
        {{ pill.label }}
      </span>
    </header>

    <section
      v-if="s && s.accounts.length"
      class="relative z-10"
    >
      <button
        class="flex w-full items-center gap-3 rounded-2xl border border-border bg-surface px-3 py-2 text-left transition-colors"
        :class="hasChoice ? 'cursor-pointer hover:bg-accent' : 'cursor-default'"
        :aria-expanded="pickerOpen"
        aria-haspopup="listbox"
        @click="togglePicker"
      >
        <span class="font-display grid size-8 shrink-0 place-items-center rounded-full bg-accent text-sm font-semibold uppercase">
          {{ (s.activeAccount || '?').charAt(0) }}
        </span>
        <span class="min-w-0 flex-1">
          <span class="block text-[0.6875rem] leading-tight text-muted-foreground">Library account</span>
          <span class="block truncate text-sm leading-tight font-medium">{{ s.activeAccount || 'No account selected' }}</span>
        </span>
        <ChevronDown
          v-if="hasChoice"
          class="size-4 shrink-0 text-muted-foreground transition-transform"
          :class="{ 'rotate-180': pickerOpen }"
        />
      </button>

      <div
        v-if="pickerOpen"
        class="absolute inset-x-0 top-full z-20 mt-1.5 max-h-52 overflow-y-auto rounded-2xl border border-border p-1.5 shadow-lg"
        style="background: var(--raised)"
        role="listbox"
      >
        <template v-if="pendingEmail">
          <p class="px-2.5 pt-1.5 pb-2 text-xs text-muted-foreground">
            Web uploads are running. Switching to
            <span class="font-medium text-foreground">{{ pendingEmail }}</span>
            will cancel them.
          </p>
          <div class="grid grid-cols-2 gap-1.5 p-1">
            <button
              class="tray-btn"
              @click="pendingEmail = null"
            >
              Keep current
            </button>
            <button
              class="tray-btn !bg-primary !text-primary-foreground"
              @click="switchTo(pendingEmail!, true)"
            >
              Switch anyway
            </button>
          </div>
        </template>
        <template v-else>
          <button
            v-for="a in s.accounts"
            :key="a.email"
            role="option"
            :aria-selected="a.active"
            class="flex w-full items-center gap-2.5 rounded-xl px-2.5 py-2 text-left text-sm transition-colors hover:bg-accent"
            @click="pickAccount(a.email, a.active)"
          >
            <span class="font-display grid size-6 shrink-0 place-items-center rounded-full bg-accent text-xs font-semibold uppercase">
              {{ a.email.charAt(0) }}
            </span>
            <span class="min-w-0 flex-1 truncate">{{ a.email }}</span>
            <Check
              v-if="a.active"
              class="size-4 shrink-0"
            />
          </button>
        </template>
      </div>
    </section>

    <section class="flex items-center gap-4 rounded-2xl border border-border bg-surface p-4">
      <ProgressRing
        :value="ringValue"
        :size="76"
        :stroke="7"
        :live="ringLive"
        :tone="tone"
      >
        <span
          v-if="showProgress"
          class="font-display text-lg leading-none font-semibold tabular-nums"
        >{{ s!.percent }}<span class="text-xs text-muted-foreground">%</span></span>
        <TriangleAlert
          v-else-if="s?.icon === 'error'"
          class="size-6 text-muted-foreground"
        />
        <CirclePause
          v-else-if="s?.uploadPaused"
          class="size-6 text-muted-foreground"
        />
        <RefreshCw
          v-else-if="s?.librarySyncing || s?.uploadActive"
          class="size-5 animate-spin text-foreground"
        />
        <CircleCheck
          v-else
          class="size-6 text-foreground"
        />
      </ProgressRing>
      <div class="min-w-0">
        <p class="font-display truncate text-lg leading-tight font-semibold">
          {{ heading }}
        </p>
        <p class="mt-0.5 truncate text-sm text-muted-foreground">
          {{ subline }}
        </p>
        <p
          v-if="speedText"
          class="mt-1 flex items-center gap-1 text-xs text-muted-foreground tabular-nums"
        >
          <ArrowUp class="size-3" /> {{ speedText }}
        </p>
        <p
          v-if="s?.librarySyncing && s?.uploadActive"
          class="mt-0.5 text-xs text-muted-foreground"
        >
          Library sync in progress
        </p>
      </div>
    </section>

    <section
      v-if="s?.libraryRunning"
      class="grid grid-cols-2 gap-2"
    >
      <div class="rounded-2xl border border-border bg-surface px-4 py-3">
        <p class="flex items-center gap-1.5 text-xs text-muted-foreground">
          <Film class="size-3.5" /> Videos
        </p>
        <p class="font-display mt-1 text-lg leading-tight font-semibold tabular-nums">
          {{ fmt(s.videos) }}
        </p>
        <p class="text-xs text-muted-foreground tabular-nums">
          {{ s.videoSize }}
        </p>
      </div>
      <div class="rounded-2xl border border-border bg-surface px-4 py-3">
        <p class="flex items-center gap-1.5 text-xs text-muted-foreground">
          <ImageIcon class="size-3.5" /> Photos
        </p>
        <p class="font-display mt-1 text-lg leading-tight font-semibold tabular-nums">
          {{ fmt(s.photos) }}
        </p>
        <p class="text-xs text-muted-foreground tabular-nums">
          {{ s.photoSize }}
        </p>
      </div>
    </section>
    <section
      v-else-if="s"
      class="flex items-center gap-3 rounded-2xl border border-dashed border-border px-4 py-3 text-sm text-muted-foreground"
    >
      <HardDrive class="size-4" /> Web library is off
    </section>

    <section class="grid grid-cols-2 gap-2">
      <button
        class="tray-btn"
        :disabled="!s?.libraryRunning"
        @click="act('openLibrary')"
      >
        <Globe class="size-4" /> Open library
      </button>
      <button
        class="tray-btn"
        :disabled="!s?.libraryRunning || s?.librarySyncing"
        @click="act('sync')"
      >
        <RefreshCw
          class="size-4"
          :class="{ 'animate-spin': s?.librarySyncing }"
        />
        {{ s?.librarySyncing ? 'Syncing…' : 'Sync now' }}
      </button>
      <button
        class="tray-btn"
        :disabled="!s?.uploadActive"
        @click="act('pauseUploads')"
      >
        <component
          :is="s?.uploadPaused ? CirclePlay : CirclePause"
          class="size-4"
        />
        {{ s?.uploadPaused ? 'Resume uploads' : 'Pause uploads' }}
      </button>
      <button
        class="tray-btn"
        :disabled="!s?.watchAvailable"
        @click="act('toggleWatch')"
      >
        <component
          :is="s?.watchPaused ? FolderX : FolderOpen"
          class="size-4"
        />
        {{ s?.watchPaused ? 'Resume watching' : 'Pause watching' }}
      </button>
    </section>

    <button
      v-if="s?.uploadActive"
      class="tray-btn tray-btn-danger"
      @click="act('cancelUploads')"
    >
      <X class="size-4" /> Cancel uploads
    </button>

    <footer class="mt-auto flex items-center gap-2">
      <button
        class="tray-btn flex-1 !bg-primary !text-primary-foreground hover:!opacity-90"
        @click="act('showApp')"
      >
        Open Photon
      </button>
      <button
        class="tray-btn !w-11 shrink-0 !px-0"
        title="Quit Photon"
        aria-label="Quit Photon"
        @click="act('quit')"
      >
        <Power class="size-4" />
      </button>
    </footer>
  </main>
</template>

<style scoped>
.tray-btn {
  display: inline-flex;
  height: 2.5rem;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  border-radius: 0.75rem;
  border: 1px solid var(--border);
  background: var(--raised);
  padding: 0 0.75rem;
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--foreground);
  white-space: nowrap;
  cursor: pointer;
  transition: background 0.15s, transform 0.1s, opacity 0.15s;
}
.tray-btn:hover:not(:disabled) {
  background: var(--accent);
}
.tray-btn:active:not(:disabled) {
  transform: scale(0.97);
}
.tray-btn:disabled {
  opacity: 0.4;
  cursor: default;
}
.tray-btn-danger {
  color: var(--muted-foreground);
}
</style>
