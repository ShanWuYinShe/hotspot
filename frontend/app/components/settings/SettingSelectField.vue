<template>
  <UFormField 
    :label="label" 
    :description="description" 
    class="space-y-2" 
    :ui="{ label: 'font-medium text-neutral-900 dark:text-white' }"
  >
    <div class="space-y-2">
      <!-- 搜索框 -->
      <UInput
        v-if="searchable"
        v-model="query"
        :placeholder="searchPlaceholder"
        size="sm"
        :aria-label="searchPlaceholder"
        class="w-full"
        :ui="{ 
          base: 'bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200 dark:border-neutral-700 focus:ring-2 focus:ring-primary-500 focus:border-primary-500 transition-colors duration-200 rounded-md px-10 py-2',
          leading: 'pl-3',
          trailing: 'pr-3'
        }"
      >
        <template #leading>
          <UIcon name="i-heroicons-magnifying-glass" class="text-neutral-500 dark:text-neutral-400 w-4 h-4 ml-0.5" />
        </template>
      </UInput>
      
      <!-- 选择器 -->
      <USelect
        v-model="internalModel"
        :options="filteredOptions"
        :placeholder="placeholder"
        :size="size"
        class="w-full"
        :disabled="disabled || loading"
        :multiple="multiple"
        :loading="loading"
        :color="error ? 'error' : 'primary'"
        :ui="{ 
          base: 'bg-white dark:bg-neutral-800 border border-neutral-200 dark:border-neutral-700 focus:ring-2 focus:ring-primary-500 focus:border-primary-500 transition-colors duration-200 rounded-md shadow-sm px-4 py-2.5'
        }"
      >
        <!-- 自定义选项显示 -->
        <template v-if="showDescriptions" #item="{ item }">
          <div v-if="item && typeof item === 'object' && 'label' in item" class="flex flex-col text-left">
            <span class="font-medium">{{ item.label }}</span>
            <span v-if="'description' in item && item.description" class="text-xs text-neutral-500 dark:text-neutral-400">
              {{ item.description }}
            </span>
          </div>
        </template>
      </USelect>
      
      <!-- 无结果提示 -->
      <div v-if="!loading && filteredOptions.length === 0 && query" class="text-xs text-neutral-500 dark:text-neutral-400 px-1 pt-1">
        {{ noResultsText }}
      </div>
      
      <!-- 错误提示 -->
      <div v-if="error" class="text-xs text-error-600 dark:text-error-400 flex items-center gap-1">
        <UIcon name="i-heroicons-exclamation-triangle" class="w-3.5 h-3.5" />
        <span>{{ error }}</span>
      </div>
      
      <!-- 提示信息 -->
      <div v-else-if="hint" class="text-xs text-neutral-500 dark:text-neutral-400">
        {{ hint }}
      </div>
    </div>
  </UFormField>
</template>

<script setup lang="ts">
interface OptionItem { 
  label: string; 
  value: string | number; 
  description?: string; 
}

interface Props {
  label: string;
  modelValue: string | number | (string | number)[] | null;
  options: OptionItem[];
  placeholder?: string;
  description?: string;
  size?: 'xs' | 'sm' | 'md' | 'lg';
  disabled?: boolean;
  searchable?: boolean;
  searchPlaceholder?: string;
  loading?: boolean;
  error?: string | null;
  hint?: string | null;
  multiple?: boolean;
  showDescriptions?: boolean;
  noResultsText?: string;
}

const props = withDefaults(defineProps<Props>(), {
  size: 'md',
  searchPlaceholder: '搜索...',
  noResultsText: '无匹配结果',
  showDescriptions: true
});

const emit = defineEmits<{ 
  'update:modelValue': [val: any] 
}>();

// 双向绑定
const internalModel = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v)
});

// 搜索功能
const query = ref('');

const filteredOptions = computed(() => {
  if (!props.searchable || !query.value.trim()) {
    return props.options;
  }
  
  const searchTerm = query.value.trim().toLowerCase();
  return props.options.filter(option => 
    option.label.toLowerCase().includes(searchTerm) ||
    (option.description && option.description.toLowerCase().includes(searchTerm))
  );
});

// 重置搜索
const resetSearch = () => {
  query.value = '';
};

defineExpose({ resetSearch });
</script>