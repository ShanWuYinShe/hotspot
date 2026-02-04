<template>
  <div class="flex items-center justify-between gap-4 py-3 first:pt-0 last:pb-0">
    <div class="flex items-start gap-3 min-w-0 flex-1">
      <div class="p-2 rounded-lg shrink-0 bg-neutral-100 dark:bg-neutral-800">
        <UIcon :name="icon" class="size-5 text-neutral-600 dark:text-neutral-300" />
      </div>
      <div class="min-w-0 flex-1">
        <div class="text-sm font-medium text-neutral-900 dark:text-white">{{ title }}</div>
        <div v-if="description" class="text-sm text-neutral-500 dark:text-neutral-400 mt-0.5">{{ description }}</div>
      </div>
    </div>
    <USwitch v-model="model" color="primary" />
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
