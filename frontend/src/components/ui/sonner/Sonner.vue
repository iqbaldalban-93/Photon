<script lang="ts" setup>
import type { ToasterProps, ToastT } from "vue-sonner"
import { Toaster as Sonner, toast } from "vue-sonner"

const props = defineProps<ToasterProps>()

function dismissOnContextMenu(event: MouseEvent) {
  const target = event.target as HTMLElement | null
  if (!target) return
  const toastEl = target.closest<HTMLElement>('[data-sonner-toast]')
  if (!toastEl) return
  event.preventDefault()

  const container = toastEl.parentElement
  if (!container) return
  const siblings = Array.from(container.querySelectorAll<HTMLElement>(':scope > [data-sonner-toast]'))
  const index = siblings.indexOf(toastEl)
  if (index < 0) return

  const active = toast.getToasts().filter((t): t is ToastT => !('dismiss' in t && t.dismiss))
  const item = active[index]
  if (item) {
    toast.dismiss(item.id as string | number)
  }
}
</script>

<template>
  <Sonner
    class="toaster group"
    v-bind="props"
    :style="{
      '--normal-bg': 'var(--popover)',
      '--normal-text': 'var(--popover-foreground)',
      '--normal-border': 'var(--border)',
      '--border-radius': '1rem',
    }"
    @contextmenu="dismissOnContextMenu"
  />
</template>