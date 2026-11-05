<script setup lang="ts">
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import HintTip from './HintTip.vue'

const model = defineModel<boolean>({ default: false })

defineProps<{
  title: string
  description?: string
  htmlFor?: string
  destructive?: boolean
  disabled?: boolean
}>()
</script>

<template>
  <div class="flex items-center justify-between gap-4 py-2.5">
    <div class="flex min-w-0 items-center gap-1.5">
      <Label
        :for="htmlFor"
        class="cursor-pointer text-[13.5px] leading-tight"
        :class="destructive ? 'text-destructive' : ''"
      >{{ title }}</Label>
      <HintTip
        v-if="description"
        :text="description"
      />
    </div>
    <Switch
      :id="htmlFor"
      v-model="model"
      class="shrink-0"
      :variant="destructive ? 'destructive' : 'default'"
      :disabled="disabled"
    />
  </div>
</template>
