
import { onUnmounted } from 'vue'
import { toast } from 'vue-sonner'

export function setupUploadToasts() {
  const uploadErrorHandler = (e: Event) => {
    const event = e as CustomEvent<{ FileName: string; Message: string }>
    const { FileName, Message } = event.detail
    const errorMessage = Message.replace(/^Error:\s*/, '')
    toast.error(FileName ? `Upload failed: ${FileName}` : 'Upload failed', {
      description: errorMessage,
      duration: 10000,
      important: true,
    })
  }

  const albumCompleteHandler = (e: Event) => {
    const event = e as CustomEvent<{ AlbumName: string; ItemsAdded: number; TotalItems: number; AlbumKeys: string[] }>
    const { AlbumName, ItemsAdded, TotalItems, AlbumKeys } = event.detail
    toast.success(`Album updated: ${AlbumName}`, {
      description: AlbumKeys?.length
        ? `${ItemsAdded}/${TotalItems} items added · ${AlbumKeys.join(', ')}`
        : `${ItemsAdded}/${TotalItems} items added`,
      duration: 6000,
    })
  }

  const albumErrorHandler = (e: Event) => {
    const event = e as CustomEvent<{ AlbumName: string; Error: string }>
    const { AlbumName, Error } = event.detail
    const errorMessage = Error.replace(/^Error:\s*/, '')
    toast.error(`Album failed: ${AlbumName || 'Unknown album'}`, {
      description: errorMessage,
      duration: 10000,
      important: true,
    })
  }

  const uploadAutoRetryHandler = (e: Event) => {
    const event = e as CustomEvent<{ count: number }>
    const count = event.detail?.count ?? 0
    toast.info(`Retrying ${count} failed file${count === 1 ? '' : 's'} in 10 seconds…`)
  }

  window.addEventListener('uploadError', uploadErrorHandler)
  window.addEventListener('albumComplete', albumCompleteHandler)
  window.addEventListener('albumError', albumErrorHandler)
  window.addEventListener('uploadAutoRetry', uploadAutoRetryHandler)

  onUnmounted(() => {
    window.removeEventListener('uploadError', uploadErrorHandler)
    window.removeEventListener('albumComplete', albumCompleteHandler)
    window.removeEventListener('albumError', albumErrorHandler)
    window.removeEventListener('uploadAutoRetry', uploadAutoRetryHandler)
  })
}
