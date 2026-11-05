<script setup lang="ts">
import { Info } from '@lucide/vue'
import {
  TooltipContent,
  TooltipPortal,
  TooltipProvider,
  TooltipRoot,
  TooltipTrigger,
} from 'reka-ui'

defineProps<{ text: string }>()
</script>

<template>
  <TooltipProvider :delay-duration="150">
    <TooltipRoot>
      <TooltipTrigger as-child>
        <button
          type="button"
          class="inline-flex size-5 shrink-0 cursor-help items-center justify-center rounded-full text-muted-foreground/70 transition-colors hover:text-primary focus-visible:text-primary"
          aria-label="More info"
        >
          <Info :size="14" />
        </button>
      </TooltipTrigger>
      <TooltipPortal>
        <TooltipContent
          side="top"
          :side-offset="6"
          :collision-padding="12"
          class="z-50 max-w-[240px] rounded-xl border border-border bg-popover px-3 py-2 text-xs leading-snug text-popover-foreground shadow-xl shadow-black/40 data-[state=delayed-open]:animate-in data-[state=delayed-open]:fade-in-0"
        >
          {{ text }}
        </TooltipContent>
      </TooltipPortal>
    </TooltipRoot>
  </TooltipProvider>
</template>
