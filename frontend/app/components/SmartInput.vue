<template>
  <div class="relative">
    <!-- 输入框 -->
    <UInput
      :model-value="modelValue"
      :placeholder="placeholder"
      :disabled="disabled"
      :loading="loading"
      :type="inputType"
      :ui="computedUi"
      v-bind="$attrs"
      @input="handleInput"
      @focus="handleFocus"
      @blur="handleBlur"
      @keydown="handleKeydown"
      ref="inputRef"
    >
      <template v-if="leadingIcon" #leading>
        <UIcon :name="leadingIcon" class="text-neutral-500 dark:text-neutral-400 w-5 h-5" />
      </template>
      
      <template #trailing>
        <!-- 清除按钮 -->
        <UButton
          v-if="showClear && modelValue && !disabled"
          variant="ghost"
          size="xs"
          color="neutral"
          @click="clearInput"
          class="hover:bg-neutral-100 dark:hover:bg-neutral-700 rounded-full mr-1"
        >
          <UIcon name="i-heroicons-x-mark" class="w-4 h-4" />
        </UButton>
        
        <!-- 密码显示切换 -->
        <UButton
          v-if="type === 'password'"
          variant="ghost"
          size="xs"
          color="neutral"
          @click="togglePasswordVisibility"
          class="hover:bg-neutral-100 dark:hover:bg-neutral-700 rounded-full mr-1"
        >
          <UIcon :name="showPassword ? 'i-heroicons-eye-slash' : 'i-heroicons-eye'" class="w-4 h-4" />
        </UButton>
        
        <!-- 自定义尾部图标 -->
        <UIcon v-if="trailingIcon" :name="trailingIcon" class="text-neutral-500 dark:text-neutral-400 w-5 h-5" />
      </template>
    </UInput>
    
    <!-- 验证状态指示器 -->
    <div v-if="showValidation && (isValid !== null || validationMessage)" class="absolute right-3 top-1/2 transform -translate-y-1/2 pointer-events-none">
      <UIcon 
        v-if="isValid === true"
        name="i-heroicons-check-circle" 
        class="w-5 h-5 text-green-500" 
      />
      <UIcon 
        v-else-if="isValid === false"
        name="i-heroicons-x-circle" 
        class="w-5 h-5 text-red-500" 
      />
      <UIcon 
        v-else-if="validating"
        name="i-heroicons-arrow-path" 
        class="w-5 h-5 text-blue-500 animate-spin" 
      />
    </div>
    
    <!-- 字符计数 -->
    <div 
      v-if="showCharCount && maxLength"
      class="absolute right-3 bottom-1 text-xs"
      :class="characterCountClass"
    >
      {{ characterCount }}/{{ maxLength }}
    </div>
    
    <!-- 验证消息 -->
    <p 
      v-if="validationMessage && showValidationMessage"
      class="mt-1 text-sm"
      :class="validationMessageClass"
    >
      {{ validationMessage }}
    </p>
    
    <!-- 帮助文本 -->
    <p v-if="helpText" class="mt-1 text-sm text-neutral-500 dark:text-neutral-400">
      {{ helpText }}
    </p>
    
    <!-- 建议列表 -->
    <div
      v-if="showSuggestions && suggestions.length > 0"
      class="absolute top-full left-0 right-0 mt-1 bg-white dark:bg-neutral-800 border border-neutral-200 dark:border-neutral-700 rounded-lg shadow-lg z-50 max-h-48 overflow-y-auto"
    >
      <div
        v-for="(suggestion, index) in suggestions"
        :key="index"
        class="px-3 py-2 hover:bg-neutral-50 dark:hover:bg-neutral-700 cursor-pointer transition-colors"
        :class="{ 'bg-neutral-50 dark:bg-neutral-700': index === selectedSuggestionIndex }"
        @click="selectSuggestion(suggestion)"
      >
        {{ suggestion }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
interface Props {
  modelValue?: string | number;
  type?: 'text' | 'email' | 'password' | 'number' | 'tel' | 'url';
  placeholder?: string;
  disabled?: boolean;
  loading?: boolean;
  leadingIcon?: string;
  trailingIcon?: string;
  showClear?: boolean;
  showValidation?: boolean;
  showValidationMessage?: boolean;
  showCharCount?: boolean;
  maxLength?: number;
  helpText?: string;
  validationRules?: ((value: string) => string | null)[];
  suggestions?: string[];
  enableSuggestions?: boolean;
  debounceMs?: number;
  variant?: 'default' | 'success' | 'error' | 'warning';
}

const props = withDefaults(defineProps<Props>(), {
  type: 'text',
  showClear: true,
  showValidation: true,
  showValidationMessage: true,
  showCharCount: false,
  enableSuggestions: false,
  debounceMs: 300,
  variant: 'default',
  suggestions: () => []
});

const emit = defineEmits<{
  'update:modelValue': [value: string | number];
  'validation': [isValid: boolean, message?: string];
  'suggestion-select': [suggestion: string];
}>();

// 响应式状态
const inputRef = ref();
const showPassword = ref(false);
const isFocused = ref(false);
const isValid = ref<boolean | null>(null);
const validationMessage = ref('');
const validating = ref(false);
const showSuggestions = ref(false);
const selectedSuggestionIndex = ref(-1);

// 计算属性
const inputType = computed(() => {
  if (props.type === 'password') {
    return showPassword.value ? 'text' : 'password';
  }
  return props.type;
});

const characterCount = computed(() => {
  return String(props.modelValue || '').length;
});

const characterCountClass = computed(() => {
  if (!props.maxLength) return 'text-neutral-500';
  
  const ratio = characterCount.value / props.maxLength;
  if (ratio >= 1) return 'text-red-500';
  if (ratio >= 0.8) return 'text-yellow-500';
  return 'text-neutral-500';
});

const validationMessageClass = computed(() => {
  if (isValid.value === false) return 'text-red-500';
  if (isValid.value === true) return 'text-green-500';
  return 'text-neutral-500';
});

const computedUi = computed(() => {
  let baseClasses = 'transition-all duration-200 px-12 py-3';
  
  // 根据验证状态调整样式
  if (props.variant === 'success' || isValid.value === true) {
    baseClasses += ' border-green-500 focus:border-green-500 focus:ring-green-500';
  } else if (props.variant === 'error' || isValid.value === false) {
    baseClasses += ' border-red-500 focus:border-red-500 focus:ring-red-500';
  } else if (props.variant === 'warning') {
    baseClasses += ' border-yellow-500 focus:border-yellow-500 focus:ring-yellow-500';
  } else {
    baseClasses += ' border-neutral-200 dark:border-neutral-700 focus:border-primary-500 focus:ring-primary-500';
  }
  
  return {
    base: baseClasses
  };
});

// 防抖验证
let validationTimeout: NodeJS.Timeout;

// 事件处理
const handleInput = (event: Event) => {
  const target = event.target as HTMLInputElement;
  const value = props.type === 'number' ? Number(target.value) : target.value;
  
  emit('update:modelValue', value);
  
  // 防抖验证
  if (props.validationRules && props.validationRules.length > 0) {
    clearTimeout(validationTimeout);
    validating.value = true;
    
    validationTimeout = setTimeout(() => {
      validateInput(String(value));
    }, props.debounceMs);
  }
  
  // 建议功能
  if (props.enableSuggestions && String(value).length > 0) {
    showSuggestions.value = true;
    selectedSuggestionIndex.value = -1;
  } else {
    showSuggestions.value = false;
  }
};

const handleFocus = () => {
  isFocused.value = true;
  if (props.enableSuggestions && props.suggestions.length > 0) {
    showSuggestions.value = true;
  }
};

const handleBlur = () => {
  isFocused.value = false;
  // 延迟隐藏建议，允许点击
  setTimeout(() => {
    showSuggestions.value = false;
  }, 200);
};

const handleKeydown = (event: KeyboardEvent) => {
  if (!showSuggestions.value || props.suggestions.length === 0) return;
  
  switch (event.key) {
    case 'ArrowDown':
      event.preventDefault();
      selectedSuggestionIndex.value = Math.min(
        selectedSuggestionIndex.value + 1,
        props.suggestions.length - 1
      );
      break;
    case 'ArrowUp':
      event.preventDefault();
      selectedSuggestionIndex.value = Math.max(
        selectedSuggestionIndex.value - 1,
        -1
      );
      break;
    case 'Enter':
      if (selectedSuggestionIndex.value >= 0) {
        event.preventDefault();
        const suggestion = props.suggestions?.[selectedSuggestionIndex.value];
        if (suggestion) {
          selectSuggestion(suggestion);
        }
      }
      break;
    case 'Escape':
      showSuggestions.value = false;
      selectedSuggestionIndex.value = -1;
      break;
  }
};

// 验证输入
const validateInput = (value: string) => {
  if (!props.validationRules || props.validationRules.length === 0) {
    validating.value = false;
    return;
  }
  
  for (const rule of props.validationRules) {
    const result = rule(value);
    if (result) {
      isValid.value = false;
      validationMessage.value = result;
      validating.value = false;
      emit('validation', false, result);
      return;
    }
  }
  
  isValid.value = true;
  validationMessage.value = '';
  validating.value = false;
  emit('validation', true);
};

// 其他操作
const clearInput = () => {
  emit('update:modelValue', '');
  isValid.value = null;
  validationMessage.value = '';
  showSuggestions.value = false;
};

const togglePasswordVisibility = () => {
  showPassword.value = !showPassword.value;
};

const selectSuggestion = (suggestion: string) => {
  emit('update:modelValue', suggestion);
  emit('suggestion-select', suggestion);
  showSuggestions.value = false;
  selectedSuggestionIndex.value = -1;
};

// 暴露方法
const focus = () => {
  inputRef.value?.focus();
};

const blur = () => {
  inputRef.value?.blur();
};

const validate = () => {
  if (props.modelValue !== undefined) {
    validateInput(String(props.modelValue));
  }
  return isValid.value;
};

defineExpose({
  focus,
  blur,
  validate,
  isValid: readonly(isValid),
  validationMessage: readonly(validationMessage)
});

// 监听外部值变化进行验证
watch(() => props.modelValue, (newValue) => {
  if (newValue !== undefined && props.validationRules && props.validationRules.length > 0) {
    validateInput(String(newValue));
  }
}, { immediate: true });
</script>
</template>
</invoke>
