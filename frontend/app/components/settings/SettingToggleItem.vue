<template>
  <UCard variant="soft">
    <div class="flex items-center justify-between gap-4">
      <div class="flex items-center gap-3 min-w-0 flex-1">
        <UBadge color="neutral" variant="soft" size="lg" square :icon="icon" />
        <div class="min-w-0 flex-1">
          <div class="text-sm font-medium truncate">{{ title }}</div>
          <UBadge v-if="description" color="neutral" variant="soft" size="xs" class="mt-1 w-fit max-w-full truncate">
            {{ description }}
          </UBadge>
        </div>
      </div>
      <USwitch v-model="model" />
    </div>
  </UCard>
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
