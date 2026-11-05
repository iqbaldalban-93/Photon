<script setup lang="ts">
import type { Component } from 'vue'
import HintTip from './HintTip.vue'

defineProps<{
  title?: string
  icon?: Component
  hint?: string
}>()
</script>

<template>
  <section class="flex flex-col gap-3 rounded-3xl bg-surface p-4">
    <header
      v-if="title || $slots.action"
      class="flex min-h-6 items-center justify-between gap-2"
    >
      <h2 class="font-display flex items-center gap-2 text-[15px] font-semibold">
        <component
          :is="icon"
          v-if="icon"
          :size="16"
          class="text-primary"
        />
        <span>{{ title }}</span>
        <HintTip
          v-if="hint"
          :text="hint"
        />
      </h2>
      <slot name="action" />
    </header>
    <slot />
  </section>
</template>
