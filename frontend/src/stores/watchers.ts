import { ref } from 'vue'
import { ConfigManager, WatchService, type WatchedFolder } from '../../bindings/app/backend'
import { toast } from 'vue-sonner'
import { reportError } from '../utils/errors'

export const watchedFolders = ref<WatchedFolder[]>([])
export const isAddingWatcher = ref(false)

export const editingWatcherPath = ref('')
export const editWatcherAlbum = ref('')

export async function refreshWatchers() {
  const config = await ConfigManager.GetConfig()
  watchedFolders.value = config.watchedFolders || []
}

export async function addWatcher(folder: string, album: string) {
  isAddingWatcher.value = true
  try {
    await WatchService.AddWatchedFolder(folder, album)
    await refreshWatchers()
    toast.success('Folder is now being watched.')
  } catch (error) {
    reportError(error, 'Failed to add watched folder')
  } finally {
    isAddingWatcher.value = false
  }
}

export async function removeWatcher(path: string) {
  try {
    await WatchService.RemoveWatchedFolder(path)
    if (editingWatcherPath.value === path) {
      cancelWatcherEdit()
    }
    await refreshWatchers()
  } catch (error) {
    reportError(error, 'Failed to remove watched folder')
  }
}

export async function toggleWatcher(path: string, enabled: boolean) {
  try {
    await WatchService.SetWatchedFolderEnabled(path, enabled)
    await refreshWatchers()
  } catch (error) {
    reportError(error, 'Failed to update watched folder')
  }
}

export function startWatcherEdit(path: string, album: string) {
  editingWatcherPath.value = path
  editWatcherAlbum.value = album
}

export function cancelWatcherEdit() {
  editingWatcherPath.value = ''
  editWatcherAlbum.value = ''
}

export async function saveWatcherAlbum(path: string) {
  try {
    await WatchService.SetWatchedFolderAlbum(path, editWatcherAlbum.value.trim())
    await refreshWatchers()
    toast.success('Watcher album updated.')
  } catch (error) {
    reportError(error, 'Failed to update watcher album')
  } finally {
    cancelWatcherEdit()
  }
}

export function useWatchers() {
  return {
    watchedFolders,
    isAddingWatcher,
    editingWatcherPath,
    editWatcherAlbum,
    refreshWatchers,
    addWatcher,
    removeWatcher,
    toggleWatcher,
    startWatcherEdit,
    cancelWatcherEdit,
    saveWatcherAlbum,
  }
}
