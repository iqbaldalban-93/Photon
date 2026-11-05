<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { cn } from '@/lib/utils'
import { LogIn, X } from '@lucide/vue'
import { Select, SelectContent, SelectTrigger, SelectValue, SelectGroup } from '@/components/ui/select'
import { Button } from '@/components/ui/button'
import SelectItem from '@/components/ui/CustomSelectItem.vue'
import { Input } from '@/components/ui/input'

interface Props {
    modelValue?: string
    options?: string[]
    triggerClass?: string
    avatar?: boolean
    showSignIn?: boolean
}

interface Emits {
    (e: 'update:modelValue', value: string): void
    (e: 'update:options', value: string[]): void
    (e: 'item-added', value: string): void
    (e: 'item-removed', value: string): void
    (e: 'sign-in'): void
}

const props = withDefaults(defineProps<Props>(), {
    modelValue: '',
    options: () => []
})

const emit = defineEmits<Emits>()

const internalOptions = ref([...props.options])
const selectedValue = ref(props.modelValue)
const newOption = ref('')
const initial = computed(() => (selectedValue.value || '?').charAt(0).toUpperCase())

watch(() => props.modelValue, (newVal) => {
    if (newVal !== selectedValue.value) {
        selectedValue.value = newVal || ''
    }
})

watch(() => props.options, (newOptions) => {
    if (JSON.stringify(newOptions) !== JSON.stringify(internalOptions.value)) {
        internalOptions.value = [...newOptions]
    }
}, { deep: true })

watch(selectedValue, (newVal) => {
    emit('update:modelValue', newVal)
})

const addOption = () => {
    const trimmed = newOption.value.trim()
    if (trimmed && !internalOptions.value.includes(trimmed)) {
        emit('item-added', trimmed)
        newOption.value = ''
    }
}

const removeOption = (index: number) => {
    const removedItem = internalOptions.value[index]
    const newOptions = [...internalOptions.value]
    newOptions.splice(index, 1)
    internalOptions.value = newOptions
    emit('update:options', newOptions)
    emit('item-removed', removedItem)

    if (selectedValue.value === removedItem) {
        selectedValue.value = ''
    }
}
</script>

<template>
  <div
    v-if="internalOptions.length === 0"
    class="flex gap-2"
  >
    <Input
      v-model="newOption"
      placeholder="Add credentials"
      @keydown.enter="addOption"
    />
    <Button
      type="button"
      :disabled="!newOption.trim()"
      @click="addOption"
    >
      Add
    </Button>
  </div>

  <Select
    v-else
    v-model="selectedValue"
  >
    <SelectTrigger :class="cn('select-none', triggerClass)">
      <span
        v-if="avatar"
        class="font-display flex size-7 shrink-0 items-center justify-center rounded-full bg-spark text-[13px] font-bold text-background"
      >{{ initial }}</span>
      <SelectValue
        class="min-w-0 flex-1 truncate text-left"
        placeholder="Select Account"
      />
    </SelectTrigger>
    <SelectContent align="end">
      <SelectGroup>
        <template
          v-for="(option, index) in internalOptions"
          :key="index"
        >
          <div class="relative flex items-center">
            <SelectItem :value="option">
              <div class="flex w-full items-center justify-between">
                <span class="truncate">{{ option }}</span>
              </div>
            </SelectItem>

            <button
              type="button"
              class="absolute right-2 p-1 rounded hover:bg-destructive z-10 group cursor-pointer"
              :title="`Remove ${option}`"
              @click.stop.prevent="() => removeOption(index)"
            >
              <X class="h-3 w-3 text-muted-foreground group-hover:text-destructive-foreground" />
            </button>
          </div>
        </template>
      </SelectGroup>
      <div
        v-if="showSignIn"
        class="p-2 pb-0"
      >
        <Button
          size="sm"
          type="button"
          variant="outline"
          class="w-full cursor-pointer"
          @click="emit('sign-in')"
        >
          <LogIn class="size-3.5" />
          Sign in with Google
        </Button>
      </div>
      <div class="p-2 flex gap-2">
        <Input
          v-model="newOption"
          class="h-8 px-1"
          placeholder="Credentials"
          @keydown.enter="addOption"
        />
        <Button
          size="sm"
          type="button"
          :disabled="!newOption.trim()"
          @click="addOption"
        >
          Add
        </Button>
      </div>
    </SelectContent>
  </Select>
</template>