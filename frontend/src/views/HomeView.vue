<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { UploadService } from '../../bindings/app/backend'
import { toast } from 'vue-sonner'
import Button from '../components/ui/button/Button.vue'
import { Input } from '../components/ui/input'
import { Switch } from '../components/ui/switch'
import { Select, SelectContent, SelectTrigger, SelectValue } from '../components/ui/select'
import SelectItem from '../components/ui/CustomSelectItem.vue'
import EditableSelect from '../components/ui/EditableSelect.vue'
import SectionCard from '../components/app/SectionCard.vue'
import AppLogo from '../components/app/AppLogo.vue'
import HintTip from '../components/app/HintTip.vue'
import GoogleSignInSheet from '../components/app/GoogleSignInSheet.vue'
import { useCredentials } from '../stores/credentials'
import { useAlbums } from '../stores/albums'
import { useWatchers } from '../stores/watchers'
import { useResumeBatch } from '../stores/resumeBatch'
import { isAlbumKey, uploadManager } from '../upload/manager'
import { baseName } from '../utils/format'
import { reportError } from '../utils/errors'
import {
  Check, FileUp, FolderOpen, FolderPlus, FolderSync,
  History, Images, KeyRound, Pencil, RotateCw, X,
} from '@lucide/vue'

const credentials = useCredentials()
const albums = useAlbums()
const watchers = useWatchers()
const resume = useResumeBatch()

const { state: uploadState } = uploadManager

const isSelectingFiles = ref(false)

const signInOpen = ref(false)
const hasAccount = computed(() => credentials.accountOptions.value.length > 0)

const pickedAlbumKey = computed({
  get: () =>
    albums.remoteAlbums.value.some(a => a.key === albums.existingAlbumKeyValue.value)
      ? albums.existingAlbumKeyValue.value
      : '',
  set: (key: string) => {
    if (key) albums.existingAlbumKey.value = key
  },
})

const pickedAlbum = computed(() =>
  albums.remoteAlbums.value.find(a => a.key === pickedAlbumKey.value),
)

function albumLabel(album: { key: string; name: string; itemCount: number }): string {
  return `${album.name || album.key.slice(0, 12) + '…'} (${album.itemCount})`
}

const albumPickerPlaceholder = computed(() => {
  if (albums.remoteAlbumsLoading.value) return 'Loading albums…'
  if (albums.remoteAlbums.value.length) return 'Pick an album'
  return 'No albums fetched'
})

watch([albums.albumMode, credentials.selectedAccount], ([mode, account]) => {
  if (mode === 'existing' && account) {
    albums.fetchRemoteAlbums()
  }
})

watch(credentials.selectedAccount, async (newValue, oldValue) => {
  if (!newValue) return
  try {
    await credentials.selectAccount(newValue)
    if (oldValue && oldValue !== newValue) {
      albums.resetTarget()
    }
    albums.restoreForAccount(newValue)
  } catch (error) {
    console.error('Failed to update selected value:', error)
    toast.error('Failed to update selected account.')
  }
})

onMounted(async () => {
  try {
    await Promise.all([
      credentials.refreshCredentials(),
      watchers.refreshWatchers(),
    ])
  } catch (error) {
    console.error('Failed to load configuration:', error)
  }
})

async function pickAndUpload(kind: 'files' | 'folder') {
  if (!hasAccount.value) return
  isSelectingFiles.value = true
  try {
    if (kind === 'files') {
      const files = await UploadService.SelectFiles()
      if (files.length > 0) startUpload(files)
    } else {
      const folder = await UploadService.SelectFolder()
      if (folder) startUpload([folder])
    }
  } catch (error) {
    reportError(error, kind === 'files' ? 'Failed to open file picker' : 'Failed to open folder picker')
  } finally {
    isSelectingFiles.value = false
  }
}

function selectFiles() {
  return pickAndUpload('files')
}

function selectFolder() {
  return pickAndUpload('folder')
}

function startUpload(files: string[]) {
  const wasAlreadyUploading = uploadState.isUploading
  uploadManager.startBatch(files, albums.albumValue.value, albums.albumMode.value)
  if (wasAlreadyUploading) {
    toast.info('Added to upload queue', {
      description: 'These files will start uploading once the current batch finishes.',
    })
  }
}

const albumModeOptions: Array<{ value: 'none' | 'new' | 'existing'; label: string }> = [
  { value: 'none', label: 'No album' },
  { value: 'new', label: 'New album' },
  { value: 'existing', label: 'Existing' },
]

function watchedAlbumLabel(album: string): string {
  if (!album) return 'No album'
  if (isAlbumKey(album)) return `Existing album · ${album.slice(0, 10)}…`
  return `New album "${album}"`
}

async function pickWatchedFolder() {
  try {
    const folder = await UploadService.SelectFolder()
    if (folder) {
      await watchers.addWatcher(folder, albums.albumValue.value)
    }
  } catch (error) {
    reportError(error, 'Failed to open folder picker')
  }
}
</script>

<template>
  <div
    class="h-full overflow-y-auto"
    style="--wails-draggable: none"
  >
    <div class="mx-auto flex w-full max-w-xl sm:max-w-2xl flex-col gap-3 px-4 pt-4 pb-3">
      <header class="flex items-center justify-between gap-3">
        <div class="flex items-center gap-2.5">
          <AppLogo :size="30" />
          <span class="font-display text-2xl font-bold tracking-tight">Photon</span>
        </div>
        <EditableSelect
          v-if="hasAccount"
          v-model="credentials.selectedAccount.value"
          :options="credentials.accountOptions.value"
          avatar
          show-sign-in
          trigger-class="h-9 w-[220px] rounded-full bg-surface py-1 pr-3 pl-1 hover:bg-raised"
          @update:options="(newOptions: string[]) => credentials.accountOptions.value = newOptions"
          @item-added="credentials.addCredentials"
          @item-removed="credentials.removeCredentials"
          @sign-in="signInOpen = true"
        />
      </header>

      <div
        v-if="resume.resumableBatch.value"
        class="flex items-start gap-3 rounded-2xl bg-primary/12 px-4 py-3 ring-1 ring-primary/40"
      >
        <History
          :size="18"
          class="mt-0.5 shrink-0 text-primary"
        />
        <div class="min-w-0 flex-1">
          <p class="text-[13px] leading-snug">
            Last time stopped partway through
            <strong class="font-semibold">{{ resume.resumableBatch.value.Total }}
              file{{ resume.resumableBatch.value.Total === 1 ? '' : 's' }}</strong>.
            Pick up where it left off?
          </p>
          <div class="mt-2.5 flex gap-2">
            <Button
              size="sm"
              @click="resume.resumeInterruptedBatch"
            >
              Resume upload
            </Button>
            <Button
              size="sm"
              variant="ghost"
              @click="resume.dismissResumableBatch"
            >
              Dismiss
            </Button>
          </div>
        </div>
      </div>

      <section
        v-if="!hasAccount"
        class="flex flex-col gap-3 rounded-3xl bg-surface p-4"
      >
        <h2 class="font-display flex items-center gap-1.5 text-[15px] font-semibold">
          Connect your account
          <HintTip text="Sign in with Google, or paste credentials you captured from the mobile app. You only need to do this once." />
        </h2>
        <Button
          class="cursor-pointer select-none"
          @click="signInOpen = true"
        >
          Sign in with Google
        </Button>
        <EditableSelect
          v-model="credentials.selectedAccount.value"
          :options="credentials.accountOptions.value"
          @update:options="(newOptions: string[]) => credentials.accountOptions.value = newOptions"
          @item-added="credentials.addCredentials"
          @item-removed="credentials.removeCredentials"
        />
      </section>

      <div
        v-if="credentials.tokenBindingEmail.value"
        class="flex items-start gap-2.5 rounded-2xl bg-warning/12 px-3 py-2.5 ring-1 ring-warning/40"
      >
        <KeyRound
          :size="16"
          class="mt-0.5 shrink-0 text-warning"
        />
        <div class="min-w-0 flex-1">
          <p class="text-xs leading-snug">
            This credential needs a token binding key from the rooted Android device it was captured from.
          </p>
          <Button
            size="sm"
            class="mt-2 bg-warning text-background hover:brightness-110"
            :disabled="credentials.isExtractingTokenBinding.value"
            @click="credentials.addTokenBindingAliasFromADB"
          >
            {{ credentials.isExtractingTokenBinding.value ? 'Reading ADB…' : 'Read from ADB' }}
          </Button>
        </div>
      </div>

      <template v-if="hasAccount">
        <section class="viewfinder rounded-3xl bg-surface px-5 py-8">
          <i /><i /><i /><i />
          <div class="grid grid-cols-2 gap-2.5">
            <Button
              size="lg"
              :disabled="isSelectingFiles"
              @click="selectFiles"
            >
              <FileUp :size="18" />
              Files
            </Button>
            <Button
              size="lg"
              variant="secondary"
              :disabled="isSelectingFiles"
              @click="selectFolder"
            >
              <FolderOpen :size="18" />
              Folder
            </Button>
          </div>
          <p
            v-if="uploadState.isUploading"
            class="mt-3 flex items-center justify-center gap-1 text-xs text-muted-foreground"
          >
            Adding to the queue
            <HintTip text="An upload is running. Anything you add now waits in line and starts by itself." />
          </p>
        </section>

        <div class="grid grid-cols-1 gap-3 md:grid-cols-2 md:items-start">
          <SectionCard
            title="Destination"
            :icon="Images"
            hint="No album: uploads go to your library only. New album: creates one with the name you type. Existing: adds to an album you pick. If none are listed, choose New album."
          >
          <div
            class="grid grid-cols-3 gap-1 rounded-full bg-muted p-1"
            role="radiogroup"
            aria-label="Album destination"
          >
            <button
              v-for="option in albumModeOptions"
              :key="option.value"
              type="button"
              role="radio"
              :aria-checked="albums.albumMode.value === option.value"
              class="h-8 cursor-pointer rounded-full text-[13px] font-semibold transition-colors"
              :class="albums.albumMode.value === option.value
                ? 'bg-foreground text-background'
                : 'text-muted-foreground hover:text-foreground'"
              @click="albums.albumMode.value = option.value"
            >
              {{ option.label }}
            </button>
          </div>
          <Input
            v-if="albums.albumMode.value === 'new'"
            v-model="albums.newAlbumName.value"
            type="text"
            placeholder="Name the new album"
          />
          <template v-else-if="albums.albumMode.value === 'existing'">
            <div class="flex gap-2">
              <Select v-model="pickedAlbumKey">
                <SelectTrigger class="min-w-0 flex-1">
                  <SelectValue :placeholder="albumPickerPlaceholder">
                    <template v-if="pickedAlbum">
                      {{ albumLabel(pickedAlbum) }}
                    </template>
                  </SelectValue>
                </SelectTrigger>
                <SelectContent class="max-h-64">
                  <SelectItem
                    v-for="album in albums.remoteAlbums.value"
                    :key="album.key"
                    :value="album.key"
                    class="text-[13px]"
                  >
                    <span class="truncate">{{ albumLabel(album) }}</span>
                  </SelectItem>
                </SelectContent>
              </Select>
              <Button
                variant="secondary"
                size="icon"
                class="size-10"
                title="Refresh album list"
                :disabled="albums.remoteAlbumsLoading.value"
                @click="albums.fetchRemoteAlbums(true)"
              >
                <RotateCw
                  :size="15"
                  :class="albums.remoteAlbumsLoading.value ? 'animate-spin' : ''"
                />
              </Button>
            </div>
            <p
              v-if="albums.remoteAlbumsError.value"
              class="text-xs leading-snug text-destructive"
            >
              Couldn't load your albums. Press refresh to try again.
            </p>
          </template>
        </SectionCard>

        <SectionCard
          title="Watched folders"
          :icon="FolderSync"
          hint="Media in a watched folder uploads on its own, into the destination chosen when you added it. Files already there are checked when the app starts or the folder is added; ones already in your library are skipped."
        >
          <template #action>
            <Button
              variant="secondary"
              size="sm"
              :disabled="watchers.isAddingWatcher.value"
              @click="pickWatchedFolder"
            >
              <FolderPlus :size="15" />
              Watch a folder
            </Button>
          </template>

          <div
            v-if="watchers.watchedFolders.value.length"
            class="flex flex-col gap-1.5"
          >
            <div
              v-for="w in watchers.watchedFolders.value"
              :key="w.path"
              class="flex items-center gap-3 rounded-2xl bg-muted px-3 py-2.5"
              :class="w.enabled ? '' : 'opacity-60'"
            >
              <span class="flex size-8 shrink-0 items-center justify-center rounded-xl bg-raised">
                <FolderOpen
                  :size="15"
                  class="text-primary"
                />
              </span>
              <div class="min-w-0 flex-1">
                <p
                  class="truncate text-[13px] font-semibold"
                  :title="w.path"
                >
                  {{ baseName(w.path) }}
                </p>
                <Input
                  v-if="watchers.editingWatcherPath.value === w.path"
                  v-model="watchers.editWatcherAlbum.value"
                  class="mt-1 h-7 rounded-lg px-2 text-xs"
                  placeholder="Album name, AF1Qip key, or empty"
                  @keydown.enter="watchers.saveWatcherAlbum(w.path)"
                />
                <p
                  v-else
                  class="truncate text-xs text-muted-foreground"
                >
                  {{ w.enabled ? watchedAlbumLabel(w.album) : 'Paused' }}
                </p>
              </div>
              <template v-if="watchers.editingWatcherPath.value === w.path">
                <Button
                  variant="ghost"
                  size="icon"
                  class="size-8 shrink-0 text-success"
                  title="Save album"
                  @click="watchers.saveWatcherAlbum(w.path)"
                >
                  <Check :size="16" />
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  class="size-8 shrink-0"
                  title="Cancel"
                  @click="watchers.cancelWatcherEdit"
                >
                  <X :size="15" />
                </Button>
              </template>
              <template v-else>
                <Switch
                  class="shrink-0"
                  :model-value="w.enabled"
                  title="Pause or resume this watcher"
                  @update:model-value="(v: boolean) => watchers.toggleWatcher(w.path, v)"
                />
                <Button
                  variant="ghost"
                  size="icon"
                  class="size-8 shrink-0"
                  title="Change album"
                  @click="watchers.startWatcherEdit(w.path, w.album)"
                >
                  <Pencil :size="14" />
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  class="size-8 shrink-0"
                  title="Stop watching"
                  @click="watchers.removeWatcher(w.path)"
                >
                  <X :size="15" />
                </Button>
              </template>
            </div>
          </div>
          <p
            v-else
            class="text-[13px] text-muted-foreground"
          >
            No folders watched yet.
          </p>
          </SectionCard>
        </div>
      </template>
    </div>
    <GoogleSignInSheet v-model:open="signInOpen" />
  </div>
</template>
