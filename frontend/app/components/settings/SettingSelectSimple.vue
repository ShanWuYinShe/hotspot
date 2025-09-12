<template>
  <UFormField 
    :label="label" 
    :description="description" 
    class="space-y-1" 
    :ui="{ label: 'font-medium text-neutral-800 dark:text-neutral-200' }"
  >
    <USelect
      v-model="internalModel"
      :options="options"
      :placeholder="placeholder"
      :disabled="disabled"
      :size="size"
      class="w-full"
    />
  </UFormField>
</template>

<script setup lang="ts">
interface Option { 
  label: string; 
  value: string | number; 
}

interface Props {
  modelValue: string | number | null;
  options: Option[];
  label: string;
  description?: string;
  placeholder?: string;
  disabled?: boolean;
  size?: 'xs' | 'sm' | 'md' | 'lg';
}

const props = withDefaults(defineProps<Props>(), {
  size: 'md'
});

const emit = defineEmits<{ 
  'update:modelValue': [value: string | number | null] 
}>();

// 双向绑定
const internalModel = computed({
  get: () => props.modelValue ?? undefined,
  set: (value) => emit('update:modelValue', value ?? null)
});
</script>