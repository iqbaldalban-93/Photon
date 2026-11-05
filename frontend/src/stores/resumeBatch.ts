import { ref } from 'vue'
import { Events } from '@wailsio/runtime'
import { uploadManager, type ResumableBatchInfo } from '../upload/manager'

export const resumableBatch = ref<ResumableBatchInfo | null>(null)

Events.On('resumableBatchFound', (event: { data: ResumableBatchInfo }) => {
  resumableBatch.value = event.data
})
Events.Emit('checkResumableBatch')

export function resumeInterruptedBatch() {
  const batch = resumableBatch.value
  if (!batch) return
  resumableBatch.value = null
  uploadManager.startBatch(batch.Files, batch.Album)
}

export function dismissResumableBatch() {
  resumableBatch.value = null
}

export function useResumeBatch() {
  return {
    resumableBatch,
    resumeInterruptedBatch,
    dismissResumableBatch,
  }
}
