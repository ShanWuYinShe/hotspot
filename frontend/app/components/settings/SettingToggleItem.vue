<template>
  <div class="flex items-center justify-between gap-4 p-3 rounded-lg bg-surface/50">
    <div class="flex items-center gap-3 min-w-0 flex-1">
      <div class="p-2 rounded-lg bg-interactive">
        <UIcon :name="icon" class="w-5 h-5 text-secondary-content" />
      </div>
      <div class="min-w-0 flex-1">
        <div class="font-medium text-sm">{{ title }}</div>
        <div v-if="description" class="text-xs text-secondary-content mt-0.5">{{ description }}</div>
      </div>
    </div>
    <USwitch v-model="model" />
  </div>
</template>

<script setup lang="ts">
interface Props {
  title: string
  icon: string
  description?: string
  modelValue: boolean
}

const props = defineProps<Props>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

const model = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})
</script>
