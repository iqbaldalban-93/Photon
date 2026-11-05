<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  value: number
  size?: number
  stroke?: number
  live?: boolean
  tone?: 'primary' | 'success' | 'muted'
}>(), { size: 132, stroke: 10, live: false, tone: 'primary' })

const radius = computed(() => (props.size - props.stroke) / 2)
const circumference = computed(() => 2 * Math.PI * radius.value)
const offset = computed(() => circumference.value * (1 - Math.min(100, Math.max(0, props.value)) / 100))
const toneVar = computed(() => ({
  primary: 'var(--primary)',
  success: 'var(--success)',
  muted: 'var(--muted-foreground)',
}[props.tone]))
</script>

<template>
  <div
    class="relative shrink-0"
    :style="{ width: `${size}px`, height: `${size}px` }"
    role="progressbar"
    :aria-valuenow="Math.round(value)"
    aria-valuemin="0"
    aria-valuemax="100"
  >
    <svg
      :width="size"
      :height="size"
      :viewBox="`0 0 ${size} ${size}`"
      class="-rotate-90"
    >
      <circle
        :cx="size / 2"
        :cy="size / 2"
        :r="radius"
        fill="none"
        stroke="var(--accent)"
        :stroke-width="stroke"
        opacity="0.7"
      />
      <circle
        :cx="size / 2"
        :cy="size / 2"
        :r="radius"
        fill="none"
        :stroke="toneVar"
        :stroke-width="stroke"
        stroke-linecap="round"
        :stroke-dasharray="circumference"
        :stroke-dashoffset="offset"
        style="transition: stroke-dashoffset 0.4s ease-out, stroke 0.3s"
      />
      <circle
        v-if="live"
        :cx="size / 2"
        :cy="size / 2"
        :r="radius"
        fill="none"
        stroke="var(--primary-foreground)"
        :stroke-width="stroke - 6"
        stroke-linecap="round"
        pathLength="60"
        class="ring-live"
        opacity="0.9"
      />
    </svg>
    <div class="absolute inset-0 flex flex-col items-center justify-center">
      <slot />
    </div>
  </div>
</template>
