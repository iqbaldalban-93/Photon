<script setup lang="ts">
import { computed, nextTick, ref, onMounted, watch } from 'vue'
import { ConfigManager, LibraryWebStatus } from '../../bindings/app/backend'
import { Browser, Clipboard } from '@wailsio/runtime'
import { toast } from 'vue-sonner'
import Button from '../components/ui/button/Button.vue'
import { Input } from '../components/ui/input'
import { Label } from '@/components/ui/label'
import SectionCard from '../components/app/SectionCard.vue'
import HintTip from '../components/app/HintTip.vue'
import ToggleRow from '../components/app/ToggleRow.vue'
import { AppWindow, Copy, Database, ExternalLink, Globe, ImageUp, ListChecks, Zap } from '@lucide/vue'

import {
    NumberField,
    NumberFieldContent,
    NumberFieldDecrement,
    NumberFieldIncrement,
    NumberFieldInput,
} from '@/components/ui/number-field'

interface Settings {
    proxy: string
    useQuota: boolean
    saver: boolean
    recursive: boolean
    forceUpload: boolean
    deleteFromHost: boolean
    disableUnsupportedFilesFilter: boolean
    setDateFromFilename: boolean
    closeToTray: boolean
    notifyOnComplete: boolean
    autoRetryFailed: boolean
    useExifDate: boolean
    uploadThreads: number
}

const settings = ref<Settings>({
    proxy: '',
    useQuota: false,
    saver: false,
    recursive: false,
    forceUpload: false,
    deleteFromHost: false,
    disableUnsupportedFilesFilter: false,
    setDateFromFilename: false,
    closeToTray: false,
    notifyOnComplete: true,
    autoRetryFailed: true,
    useExifDate: true,
    uploadThreads: 0
})
const isHydrating = ref(true)

const historyCount = ref<number | null>(null)

async function refreshHistoryCount() {
    try {
        historyCount.value = await ConfigManager.GetUploadHistoryCount()
    } catch (error) {
        console.error('Failed to read upload history count:', error)
    }
}

async function clearUploadHistory() {
    try {
        await ConfigManager.ClearUploadHistory()
        historyCount.value = 0
        toast.success('Upload history cleared.')
    } catch (error) {
        console.error('Failed to clear upload history:', error)
        toast.error('Failed to clear upload history.')
    }
}

const libraryStatus = ref<LibraryWebStatus | null>(null)
const libraryEnabled = ref(false)
const libraryLan = ref(true)
const libraryPort = ref('8787')
const libraryBusy = ref(false)

function applyLibraryStatus(status: LibraryWebStatus) {
    libraryStatus.value = status
    if (status.error) toast.error(status.error)
}

const portChanged = computed(() => Number(libraryPort.value) !== libraryStatus.value?.port)

async function runLibrary(action: () => Promise<LibraryWebStatus>) {
    libraryBusy.value = true
    try {
        applyLibraryStatus(await action())
    } catch (error) {
        console.error('Failed to update the browser library:', error)
        toast.error(error instanceof Error ? error.message : 'Could not update the browser library.')
    } finally {
        libraryBusy.value = false
    }
}

async function applyLibraryPort() {
    const port = Number(libraryPort.value)
    if (!Number.isInteger(port) || port < 1024 || port > 65535) {
        toast.error('Enter a port between 1024 and 65535.')
        return
    }
    await runLibrary(() => ConfigManager.SetLibraryWebPort(port))
}

async function copyUrl(url: string) {
    try {
        await Clipboard.SetText(url)
        toast.success('Address copied.')
    } catch {
        toast.error('Could not copy the address.')
    }
}

onMounted(async () => {
    try {
        const config = await ConfigManager.GetConfig()
        libraryEnabled.value = config.libraryWebEnabled || false
        libraryLan.value = config.libraryWebLan ?? true
        libraryPort.value = String(config.libraryWebPort || 8787)
        libraryStatus.value = await ConfigManager.GetLibraryWebStatus()
        settings.value = {
            proxy: config.proxy || '',
            useQuota: config.useQuota || false,
            saver: config.saver || false,
            recursive: config.recursive || false,
            forceUpload: config.forceUpload || false,
            deleteFromHost: config.deleteFromHost || false,
            disableUnsupportedFilesFilter: config.disableUnsupportedFilesFilter || false,
            setDateFromFilename: config.setDateFromFilename || false,
            closeToTray: config.closeToTray || false,
            notifyOnComplete: config.notifyOnComplete ?? true,
            autoRetryFailed: config.autoRetryFailed ?? true,
            useExifDate: config.useExifDate ?? true,
            uploadThreads: config.uploadThreads || 1
        }
        await refreshHistoryCount()
    } catch (error) {
        console.error('Failed to load settings:', error)
    } finally {
        await nextTick()
        isHydrating.value = false
    }
})

function persist(mutate: () => Promise<unknown>) {
    return async (_newValue: unknown) => {
        if (isHydrating.value) return
        try {
            await mutate()
        } catch (error) {
            console.error('Failed to persist setting:', error)
        }
    }
}

watch(() => settings.value.proxy, persist(() => ConfigManager.SetProxy(settings.value.proxy)))

watch(() => settings.value.useQuota, persist(() => ConfigManager.SetUseQuota(settings.value.useQuota)))
watch(() => settings.value.saver, persist(() => ConfigManager.SetSaver(settings.value.saver)))
watch(() => settings.value.recursive, persist(() => ConfigManager.SetRecursive(settings.value.recursive)))
watch(() => settings.value.forceUpload, persist(() => ConfigManager.SetForceUpload(settings.value.forceUpload)))
watch(() => settings.value.deleteFromHost, persist(() => ConfigManager.SetDeleteFromHost(settings.value.deleteFromHost)))
watch(() => settings.value.disableUnsupportedFilesFilter, persist(() => ConfigManager.SetDisableUnsupportedFilesFilter(settings.value.disableUnsupportedFilesFilter)))
watch(() => settings.value.setDateFromFilename, persist(() => ConfigManager.SetSetDateFromFilename(settings.value.setDateFromFilename)))
watch(() => settings.value.closeToTray, persist(() => ConfigManager.SetCloseToTray(settings.value.closeToTray)))
watch(() => settings.value.notifyOnComplete, persist(() => ConfigManager.SetNotifyOnComplete(settings.value.notifyOnComplete)))
watch(() => settings.value.autoRetryFailed, persist(() => ConfigManager.SetAutoRetryFailed(settings.value.autoRetryFailed)))
watch(() => settings.value.useExifDate, persist(() => ConfigManager.SetUseExifDate(settings.value.useExifDate)))

watch(libraryEnabled, async (enabled) => {
    if (isHydrating.value) return
    await runLibrary(() => ConfigManager.SetLibraryWebEnabled(enabled))
})
watch(libraryLan, async (lan) => {
    if (isHydrating.value) return
    await runLibrary(() => ConfigManager.SetLibraryWebLAN(lan))
})

watch(() => settings.value.uploadThreads, async (_newValue) => {
    if (isHydrating.value) return
    if (settings.value.uploadThreads < 1) {
        settings.value.uploadThreads = 1
    } else {
        await ConfigManager.SetUploadThreads(settings.value.uploadThreads)
    }
})
</script>

<template>
  <div
    class="h-full overflow-y-auto"
    style="--wails-draggable: none"
  >
    <div class="mx-auto grid w-full max-w-xl sm:max-w-2xl grid-cols-1 gap-3 px-4 pt-4 pb-3 md:grid-cols-2 md:items-start">
      <h1 class="font-display text-2xl font-bold md:col-span-2">
        Settings
      </h1>

      <SectionCard
        title="Speed"
        :icon="Zap"
      >
        <div class="flex items-center justify-between gap-4">
          <div class="flex min-w-0 items-center gap-1.5">
            <Label class="text-[13.5px] leading-tight">Upload threads</Label>
            <HintTip text="How many files are sent at the same time." />
          </div>
          <NumberField
            v-model="settings.uploadThreads"
            class="flex shrink-0 items-center gap-1"
          >
            <NumberFieldContent>
              <NumberFieldDecrement
                class="cursor-pointer"
                :disabled="settings.uploadThreads <= 1"
              />
              <NumberFieldInput class="h-9 w-[84px]" />
              <NumberFieldIncrement class="cursor-pointer" />
            </NumberFieldContent>
          </NumberField>
        </div>
      </SectionCard>

      <SectionCard
        title="Quality and dates"
        :icon="ImageUp"
      >
        <div class="divide-y divide-border/60 [&>*:first-child]:pt-0 [&>*:last-child]:pb-0">
          <ToggleRow
            v-model="settings.useQuota"
            title="Original quality"
            description="No compression. Counts against your storage quota."
          />
          <ToggleRow
            v-model="settings.saver"
            title="Storage saver"
            description="Compress photos and videos to use less space."
          />
          <ToggleRow
            v-model="settings.useExifDate"
            title="Use the date the photo was taken"
            description="Reads the EXIF capture date instead of the file date."
          />
          <ToggleRow
            v-model="settings.setDateFromFilename"
            title="Read the date from the file name"
            description="Used when a file has no capture date."
          />
        </div>
      </SectionCard>

      <SectionCard
        title="While uploading"
        :icon="ListChecks"
      >
        <div class="divide-y divide-border/60 [&>*:first-child]:pt-0 [&>*:last-child]:pb-0">
          <ToggleRow
            v-model="settings.recursive"
            title="Include subfolders"
            description="Scan inside folders you choose."
          />
          <ToggleRow
            v-model="settings.autoRetryFailed"
            title="Retry failed files"
            description="Tries each failed file once more, 10 seconds after the batch ends."
          />
          <ToggleRow
            v-model="settings.forceUpload"
            title="Upload duplicates too"
            description="Skips the duplicate check and sends everything."
          />
          <ToggleRow
            v-model="settings.disableUnsupportedFilesFilter"
            title="Try unsupported file types"
            description="Sends files Google Photos may reject."
          />
          <ToggleRow
            v-model="settings.deleteFromHost"
            title="Move originals to the Recycle Bin"
            description="Once a file is uploaded, the copy on this computer is removed."
            destructive
          />
        </div>
      </SectionCard>

      <SectionCard
        title="Browser library"
        :icon="Globe"
        hint="Opens your Google Photos library and albums in any web browser, using the account you picked on Home."
      >
        <div class="divide-y divide-border/60 [&>*:first-child]:pt-0 [&>*:last-child]:pb-0">
          <ToggleRow
            v-model="libraryEnabled"
            title="Turn on the browser library"
            description="Browse photos, videos and albums, and upload, from a web browser."
            :disabled="libraryBusy"
          />
          <ToggleRow
            v-model="libraryLan"
            title="Allow other devices on my network"
            description="Phones and computers on the same Wi-Fi or LAN can open it. Off means this computer only."
            :disabled="libraryBusy"
          />
        </div>

        <div class="flex items-center justify-between gap-3">
          <div class="flex items-center gap-1.5">
            <Label
              for="library-port"
              class="text-[13.5px]"
            >Port</Label>
            <HintTip text="The number at the end of the address. Change it if another program already uses this port." />
          </div>
          <div class="flex items-center gap-2">
            <Input
              id="library-port"
              v-model="libraryPort"
              type="number"
              inputmode="numeric"
              min="1024"
              max="65535"
              class="font-display h-9 w-24 rounded-xl text-center font-semibold tabular-nums"
              @keydown.enter="applyLibraryPort"
            />
            <Button
              size="sm"
              variant="secondary"
              :disabled="libraryBusy || !portChanged"
              @click="applyLibraryPort"
            >
              Apply
            </Button>
          </div>
        </div>

        <div
          v-if="libraryEnabled"
          class="flex flex-col gap-2 rounded-2xl bg-muted p-3"
        >
          <div class="flex items-center gap-2">
            <span
              class="size-2 rounded-full"
              :class="libraryStatus?.running ? 'bg-success' : 'ring-1 ring-inset ring-muted-foreground'"
            />
            <span class="text-[13px] font-semibold">
              {{ libraryStatus?.running ? 'Running' : 'Not running' }}
            </span>
            <HintTip text="The first time, open the localhost address in a browser on this computer to create your login. After that you can sign in from any device." />
          </div>
          <p
            v-if="libraryStatus?.error"
            class="text-xs leading-snug text-destructive"
          >
            {{ libraryStatus.error }}
          </p>
          <div
            v-for="url in libraryStatus?.running ? libraryStatus.urls : []"
            :key="url"
            class="flex items-center gap-1.5"
          >
            <span class="min-w-0 flex-1 truncate font-mono text-xs text-primary">{{ url }}</span>
            <Button
              variant="ghost"
              size="icon"
              class="size-8 shrink-0"
              title="Copy address"
              @click="copyUrl(url)"
            >
              <Copy :size="14" />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              class="size-8 shrink-0"
              title="Open in browser"
              @click="Browser.OpenURL(url)"
            >
              <ExternalLink :size="14" />
            </Button>
          </div>
        </div>
      </SectionCard>

      <SectionCard
        title="App"
        :icon="AppWindow"
      >
        <div class="divide-y divide-border/60 [&>*:first-child]:pt-0 [&>*:last-child]:pb-0">
          <ToggleRow
            v-model="settings.notifyOnComplete"
            title="Notify me when uploads finish"
            description="Shows a system notification once every batch is done."
          />
          <ToggleRow
            v-model="settings.closeToTray"
            title="Keep running in the tray"
            description="Closing the window leaves uploads going in the background."
          />
        </div>
        <div class="flex flex-col gap-1.5 pt-1">
          <div class="flex items-center gap-1.5">
            <Label
              for="proxy"
              class="text-[13.5px]"
            >Proxy</Label>
            <HintTip text="Traffic through the proxy still verifies TLS certificates." />
          </div>
          <Input
            id="proxy"
            v-model="settings.proxy"
            type="text"
            placeholder="http://127.0.0.1:8080 (optional)"
          />
        </div>
      </SectionCard>

      <SectionCard
        title="Data"
        :icon="Database"
      >
        <div class="flex items-center justify-between gap-4">
          <div class="min-w-0">
            <div class="flex items-center gap-1.5">
              <Label class="text-[13.5px] leading-tight">Upload history</Label>
              <HintTip text="Remembered uploads let Photon skip repeats without going online." />
            </div>
            <span class="mt-1 block text-xs text-muted-foreground tabular-nums">
              {{ historyCount === null ? 'Loading…' : `${historyCount} upload${historyCount === 1 ? '' : 's'}` }}
            </span>
          </div>
          <Button
            variant="secondary"
            size="sm"
            class="shrink-0"
            title="Forget all remembered uploads"
            @click="clearUploadHistory"
          >
            Clear
          </Button>
        </div>
      </SectionCard>
    </div>
  </div>
</template>
