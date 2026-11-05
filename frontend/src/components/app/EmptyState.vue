<script setup lang="ts">
import type { Component } from 'vue'
import HintTip from './HintTip.vue'

defineProps<{
  icon: Component
  title: string
  hint?: string
}>()
</script>

<template>
  <div class="flex flex-1 flex-col items-center justify-center gap-3 px-8 text-center">
    <div class="viewfinder flex size-28 items-center justify-center rounded-3xl bg-surface">
      <i /><i /><i /><i />
      <component
        :is="icon"
        :size="34"
        class="text-primary"
        :stroke-width="1.6"
      />
    </div>
    <p class="font-display mt-2 flex items-center gap-1.5 text-xl font-semibold">
      {{ title }}
      <HintTip
        v-if="hint"
        :text="hint"
      />
    </p>
    <slot />
  </div>
</template>
