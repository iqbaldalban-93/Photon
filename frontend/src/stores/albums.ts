import { computed, ref, watch } from 'vue'
import { UploadService, type AlbumInfo } from '../../bindings/app/backend'
import { selectedAccount } from './credentials'
import { describe } from '../utils/errors'

export type AlbumMode = 'none' | 'new' | 'existing'

export const albumMode = ref<AlbumMode>('none')
export const newAlbumName = ref('')
export const existingAlbumKey = ref('')

export const existingAlbumKeyValue = computed(() => {
  const raw = existingAlbumKey.value.trim()
  const match = raw.match(/AF1Qip\S*/)
  return match ? match[0] : raw
})

export const albumValue = computed(() => {
  if (albumMode.value === 'new') return newAlbumName.value.trim()
  if (albumMode.value === 'existing') return existingAlbumKeyValue.value
  return ''
})

export const remoteAlbums = ref<AlbumInfo[]>([])
export const remoteAlbumsLoading = ref(false)
export const remoteAlbumsError = ref('')

const CACHE_PREFIX = 'photon.albums.'

interface AlbumCache {
  key: string
  albums: AlbumInfo[]
}

function readCache(account: string): AlbumCache | null {
  try {
    const raw = localStorage.getItem(CACHE_PREFIX + account)
    return raw ? (JSON.parse(raw) as AlbumCache) : null
  } catch {
    return null
  }
}

function writeCache(account: string) {
  try {
    const data: AlbumCache = { key: existingAlbumKeyValue.value, albums: remoteAlbums.value }
    localStorage.setItem(CACHE_PREFIX + account, JSON.stringify(data))
  } catch {
  }
}

let fetchedThisSession = false

export function restoreForAccount(account: string) {
  if (!account) return
  const cached = readCache(account)
  if (!cached) return
  if (!remoteAlbums.value.length) remoteAlbums.value = cached.albums || []
  if (!existingAlbumKey.value) existingAlbumKey.value = cached.key || ''
}

export async function fetchRemoteAlbums(force = false) {
  if (remoteAlbumsLoading.value) return
  if (!force && fetchedThisSession) return
  const account = selectedAccount.value
  if (!account) return
  remoteAlbumsLoading.value = true
  remoteAlbumsError.value = ''
  try {
    const albums = await UploadService.ListAlbums()
    if (account !== selectedAccount.value) return
    remoteAlbums.value = (albums || []).slice().sort((a, b) => a.name.localeCompare(b.name))
    fetchedThisSession = true
    writeCache(account)
  } catch (error) {
    console.error('Failed to load albums:', error)
    remoteAlbumsError.value = describe(error)
  } finally {
    remoteAlbumsLoading.value = false
  }
}

watch(existingAlbumKey, () => {
  if (selectedAccount.value && existingAlbumKey.value) writeCache(selectedAccount.value)
})

export function resetTarget() {
  existingAlbumKey.value = ''
  albumMode.value = 'none'
  remoteAlbums.value = []
  remoteAlbumsError.value = ''
  fetchedThisSession = false
}

export function useAlbums() {
  return {
    albumMode,
    newAlbumName,
    existingAlbumKey,
    existingAlbumKeyValue,
    albumValue,
    remoteAlbums,
    remoteAlbumsLoading,
    remoteAlbumsError,
    fetchRemoteAlbums,
    resetTarget,
    restoreForAccount,
  }
}
