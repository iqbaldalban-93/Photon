<script setup lang="ts">
import { useColorMode } from '@vueuse/core'
import { computed, ref, watch } from 'vue'
import { Activity, House, Settings } from '@lucide/vue'
import './index.css'
import HomeView from './views/HomeView.vue'
import ActivityView from './views/ActivityView.vue'
import SettingsView from './views/SettingsView.vue'
import Toaster from './components/ui/sonner/Sonner.vue'
import { setupUploadToasts } from './utils/uploadToasts'
import { uploadManager } from './upload/manager'

useColorMode().value = 'dark'

setupUploadToasts()

const { state: uploadState } = uploadManager

const activeTab = ref<'home' | 'activity' | 'settings'>('home')

watch(() => uploadState.chainId, () => {
  if (uploadState.startedByUser && activeTab.value === 'home') {
    activeTab.value = 'activity'
  }
})

const tabs = [
  { id: 'home', label: 'Home', icon: House },
  { id: 'activity', label: 'Activity', icon: Activity },
  { id: 'settings', label: 'Settings', icon: Settings },
] as const

const activeIndex = computed(() => tabs.findIndex(t => t.id === activeTab.value))
</script>

<template>
  <main
    class="flex h-screen w-screen flex-col bg-background"
    style="--wails-draggable: none"
  >
    <div class="min-h-0 flex-1">
      <HomeView
        v-if="activeTab === 'home'"
        class="h-full"
      />
      <ActivityView
        v-else-if="activeTab === 'activity'"
        class="h-full"
      />
      <SettingsView
        v-else
        class="h-full"
      />
    </div>

    <nav
      class="shrink-0 px-4 pt-1 pb-3"
      style="--wails-draggable: none"
      aria-label="Main"
    >
      <div class="relative mx-auto grid max-w-xl sm:max-w-2xl grid-cols-3 rounded-full bg-surface p-1.5">
        <span
          class="absolute inset-y-1.5 left-1.5 w-[calc((100%-0.75rem)/3)] rounded-full bg-primary transition-transform duration-300 ease-[cubic-bezier(.4,1.3,.5,1)]"
          :style="{ transform: `translateX(${activeIndex * 100}%)` }"
        />
        <button
          v-for="tab in tabs"
          :key="tab.id"
          type="button"
          class="relative z-10 flex h-10 cursor-pointer select-none items-center justify-center gap-2 rounded-full text-sm font-semibold transition-colors"
          :class="activeTab === tab.id ? 'text-primary-foreground' : 'text-muted-foreground hover:text-foreground'"
          :aria-current="activeTab === tab.id ? 'page' : undefined"
          @click="activeTab = tab.id"
        >
          <component
            :is="tab.icon"
            :size="17"
            :stroke-width="activeTab === tab.id ? 2.5 : 2"
          />
          {{ tab.label }}
          <span
            v-if="tab.id === 'activity' && uploadState.isUploading"
            class="absolute right-3 top-2.5 size-2 rounded-full pulse-slow"
            :class="activeTab === 'activity' ? 'bg-primary-foreground' : 'bg-primary'"
          />
        </button>
      </div>
    </nav>

    <Toaster
      theme="dark"
      position="top-center"
      expand
      :visible-toasts="4"
    />
  </main>
</template>
