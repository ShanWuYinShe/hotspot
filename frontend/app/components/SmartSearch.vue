<template>
  <div class="relative">
    <!-- 搜索输入框 -->
    <UInput
      v-model="searchQuery"
      :placeholder="placeholder"
      :loading="loading"
      :disabled="disabled"
      class="w-full"
      :ui="inputUi"
      @input="handleInput"
      @keyup.enter="handleEnter"
      @keyup.escape="handleEscape"
      @focus="handleFocus"
      @blur="handleBlur"
    >
      <template #leading>
        <UIcon :name="leadingIcon" />
      </template>
      
      <template #trailing>
        <!-- 清除按钮 -->
        <UButton
          v-if="searchQuery && showClear"
          variant="ghost"
          size="xs"
          color="neutral"
          @click="clearSearch"
          class="hover:bg-neutral-100 dark:hover:bg-neutral-700 rounded-full"
        >
          <UIcon name="i-heroicons-x-mark" class="w-4 h-4" />
        </UButton>
        
        <!-- 搜索按钮 -->
        <UButton
          v-if="showSearchButton"
          variant="ghost"
          size="xs"
          :color="searchButtonColor"
          :loading="loading"
          :disabled="!canSearch"
          @click="performSearch"
          class="ml-1"
        >
          <UIcon name="i-heroicons-magnifying-glass" class="w-4 h-4" />
        </UButton>
      </template>
    </UInput>
    
    <!-- 搜索建议下拉框 -->
    <div
      v-if="showSuggestions && (suggestions.length > 0 || recentSearches.length > 0)"
      class="absolute top-full left-0 right-0 mt-1 bg-white dark:bg-neutral-800 border border-neutral-200 dark:border-neutral-700 rounded-lg shadow-lg z-50 max-h-80 overflow-y-auto"
    >
      <!-- 搜索建议 -->
      <div v-if="suggestions.length > 0">
        <div class="px-3 py-2 text-xs font-medium text-neutral-500 dark:text-neutral-400 border-b border-neutral-100 dark:border-neutral-700">
          搜索建议
        </div>
        <div
          v-for="(suggestion, index) in suggestions"
          :key="`suggestion-${index}`"
          class="px-3 py-2 hover:bg-neutral-50 dark:hover:bg-neutral-700 cursor-pointer flex items-center gap-3 transition-colors"
          @click="selectSuggestion(suggestion)"
        >
          <UIcon name="i-heroicons-magnifying-glass" class="w-4 h-4 text-neutral-400" />
          <span class="flex-1 truncate">{{ suggestion }}</span>
          <UIcon name="i-heroicons-arrow-up-left" class="w-3 h-3 text-neutral-400" />
        </div>
      </div>
      
      <!-- 最近搜索 -->
      <div v-if="recentSearches.length > 0 && !searchQuery">
        <div class="px-3 py-2 text-xs font-medium text-neutral-500 dark:text-neutral-400 border-b border-neutral-100 dark:border-neutral-700 flex items-center justify-between">
          <span>最近搜索</span>
          <UButton
            variant="ghost"
            size="xs"
            color="neutral"
            @click="clearRecentSearches"
          >
            <UIcon name="i-heroicons-trash" class="w-3 h-3" />
          </UButton>
        </div>
        <div
          v-for="(recent, index) in recentSearches"
          :key="`recent-${index}`"
          class="px-3 py-2 hover:bg-neutral-50 dark:hover:bg-neutral-700 cursor-pointer flex items-center gap-3 transition-colors"
          @click="selectSuggestion(recent)"
        >
          <UIcon name="i-heroicons-clock" class="w-4 h-4 text-neutral-400" />
          <span class="flex-1 truncate">{{ recent }}</span>
          <UButton
            variant="ghost"
            size="xs"
            color="neutral"
            @click.stop="removeRecentSearch(recent)"
          >
            <UIcon name="i-heroicons-x-mark" class="w-3 h-3" />
          </UButton>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
interface Props {
  placeholder?: string;
  loading?: boolean;
  disabled?: boolean;
  leadingIcon?: string;
  showClear?: boolean;
  showSearchButton?: boolean;
  searchButtonColor?: 'primary' | 'secondary' | 'success' | 'info' | 'warning' | 'error' | 'neutral';
  enableSuggestions?: boolean;
  enableRecentSearches?: boolean;
  maxRecentSearches?: number;
  minSearchLength?: number;
  searchDelay?: number;
  suggestions?: string[];
  storageKey?: string;
}

const props = withDefaults(defineProps<Props>(), {
  placeholder: '搜索...',
  leadingIcon: 'i-heroicons-magnifying-glass',
  showClear: true,
  showSearchButton: true,
  searchButtonColor: 'primary',
  enableSuggestions: true,
  enableRecentSearches: true,
  maxRecentSearches: 10,
  minSearchLength: 1,
  searchDelay: 300,
  suggestions: () => [],
  storageKey: 'smart-search-recent'
});

const emit = defineEmits<{
  search: [query: string];
  input: [query: string];
  focus: [];
  blur: [];
  clear: [];
  suggestionSelect: [suggestion: string];
}>();

// 响应式状态
const searchQuery = ref('');
const showSuggestions = ref(false);
const recentSearches = ref<string[]>([]);

// 计算属性
const canSearch = computed(() => {
  return searchQuery.value.trim().length >= props.minSearchLength;
});

const inputUi = {
  base: 'bg-white dark:bg-neutral-800 shadow-sm border border-neutral-200 dark:border-neutral-700 focus:ring-2 focus:ring-primary-500 focus:border-primary-500 focus:shadow-lg focus:shadow-primary-500/25 transition-all duration-200 rounded-lg px-12 py-3'
};

// 防抖搜索
let searchTimeout: NodeJS.Timeout;

// 事件处理
const handleInput = (event: Event) => {
  const target = event.target as HTMLInputElement;
  const query = target.value;
  
  emit('input', query);
  
  // 防抖处理建议
  if (props.enableSuggestions && query.length >= props.minSearchLength) {
    clearTimeout(searchTimeout);
    searchTimeout = setTimeout(() => {
      // 这里可以触发建议搜索
      showSuggestions.value = true;
    }, props.searchDelay);
  } else {
    showSuggestions.value = false;
  }
};

const handleEnter = () => {
  if (canSearch.value) {
    performSearch();
  }
};

const handleEscape = () => {
  showSuggestions.value = false;
  searchQuery.value = '';
};

const handleFocus = () => {
  if (props.enableSuggestions || props.enableRecentSearches) {
    showSuggestions.value = true;
  }
  emit('focus');
};

const handleBlur = () => {
  // 延迟隐藏，允许点击建议
  setTimeout(() => {
    showSuggestions.value = false;
  }, 200);
  emit('blur');
};

// 搜索操作
const performSearch = () => {
  const query = searchQuery.value.trim();
  if (query.length >= props.minSearchLength) {
    addToRecentSearches(query);
    showSuggestions.value = false;
    emit('search', query);
  }
};

const clearSearch = () => {
  searchQuery.value = '';
  showSuggestions.value = false;
  emit('clear');
};

// 建议选择
const selectSuggestion = (suggestion: string) => {
  searchQuery.value = suggestion;
  showSuggestions.value = false;
  addToRecentSearches(suggestion);
  emit('suggestionSelect', suggestion);
  emit('search', suggestion);
};

// 最近搜索管理
const loadRecentSearches = () => {
  if (!props.enableRecentSearches) return;
  
  try {
    const stored = localStorage.getItem(props.storageKey);
    if (stored) {
      recentSearches.value = JSON.parse(stored);
    }
  } catch (error) {
    console.warn('Failed to load recent searches:', error);
  }
};

const saveRecentSearches = () => {
  if (!props.enableRecentSearches) return;
  
  try {
    localStorage.setItem(props.storageKey, JSON.stringify(recentSearches.value));
  } catch (error) {
    console.warn('Failed to save recent searches:', error);
  }
};

const addToRecentSearches = (query: string) => {
  if (!props.enableRecentSearches || !query.trim()) return;
  
  // 移除重复项
  recentSearches.value = recentSearches.value.filter(item => item !== query);
  
  // 添加到开头
  recentSearches.value.unshift(query);
  
  // 限制数量
  if (recentSearches.value.length > props.maxRecentSearches) {
    recentSearches.value = recentSearches.value.slice(0, props.maxRecentSearches);
  }
  
  saveRecentSearches();
};

const removeRecentSearch = (query: string) => {
  recentSearches.value = recentSearches.value.filter(item => item !== query);
  saveRecentSearches();
};

const clearRecentSearches = () => {
  recentSearches.value = [];
  saveRecentSearches();
};

// 暴露方法
const focus = () => {
  // 可以通过 ref 调用
};

const setQuery = (query: string) => {
  searchQuery.value = query;
};

defineExpose({
  focus,
  setQuery,
  clearSearch,
  query: readonly(searchQuery)
});

// 生命周期 - onMounted 确保只在客户端执行
onMounted(() => {
  loadRecentSearches();
});

// 监听点击外部关闭建议
onMounted(() => {
  const handleClickOutside = (event: Event) => {
    const target = event.target as Element;
    if (!target.closest('.relative')) {
      showSuggestions.value = false;
    }
  };
  
  document.addEventListener('click', handleClickOutside);
  
  onUnmounted(() => {
    document.removeEventListener('click', handleClickOutside);
  });
});
</script>
