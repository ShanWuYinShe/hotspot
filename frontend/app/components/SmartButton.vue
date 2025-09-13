<template>
  <UButton
    :variant="computedVariant"
    :color="computedColor"
    :size="size"
    :loading="isLoading"
    :disabled="isDisabled"
    :class="computedClass"
    v-bind="$attrs"
    @click="handleClick"
  >
    <template v-if="leadingIcon && !isLoading" #leading>
      <UIcon :name="leadingIcon" :class="iconClass" />
    </template>
    
    <template v-if="trailingIcon && !isLoading" #trailing>
      <UIcon :name="trailingIcon" :class="iconClass" />
    </template>
    
    <slot />
    
    <!-- 成功状态图标 -->
    <template v-if="showSuccessIcon" #leading>
      <UIcon name="i-heroicons-check" :class="iconClass" />
    </template>
  </UButton>
</template>

<script setup lang="ts">
interface Props {
  variant?: 'solid' | 'outline' | 'soft' | 'ghost' | 'link';
  color?: 'primary' | 'secondary' | 'success' | 'info' | 'warning' | 'error' | 'neutral';
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl';
  loading?: boolean;
  disabled?: boolean;
  leadingIcon?: string;
  trailingIcon?: string;
  
  // 智能功能
  smartLoading?: boolean; // 点击后自动显示加载状态
  loadingDuration?: number; // 自动加载持续时间
  showSuccessState?: boolean; // 显示成功状态
  successDuration?: number; // 成功状态持续时间
  confirmRequired?: boolean; // 需要二次确认
  confirmMessage?: string; // 确认消息
  debounce?: number; // 防抖延迟
  throttle?: number; // 节流延迟
  
  // 样式增强
  gradient?: boolean; // 渐变背景
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'solid',
  color: 'primary',
  size: 'md',
  smartLoading: false,
  loadingDuration: 2000,
  showSuccessState: false,
  successDuration: 1500,
  confirmRequired: false,
  confirmMessage: '确定要执行此操作吗？',
  debounce: 0,
  throttle: 0,
  gradient: false,
  shadow: true,
  pulse: false,
  bounce: false,
  scale: true
});

const emit = defineEmits<{
  click: [event: MouseEvent];
  success: [];
  error: [error: any];
}>();

// 响应式状态
const isLoading = ref(false);
const isSuccess = ref(false);
const showSuccessIcon = ref(false);
const lastClickTime = ref(0);

// 计算属性
const isDisabled = computed(() => {
  return props.disabled || isLoading.value;
});

const computedVariant = computed(() => {
  if (isSuccess.value) return 'soft';
  return props.variant;
});

const computedColor = computed(() => {
  if (isSuccess.value) return 'success';
  return props.color;
});

const iconClass = computed(() => {
  const classes = [];
  
  switch (props.size) {
    case 'xs':
      classes.push('w-3 h-3');
      break;
    case 'sm':
      classes.push('w-4 h-4');
      break;
    case 'md':
      classes.push('w-4 h-4');
      break;
    case 'lg':
      classes.push('w-5 h-5');
      break;
    case 'xl':
      classes.push('w-6 h-6');
      break;
  }
  
  return classes.join(' ');
});

const computedClass = computed(() => {
  const classes: string[] = [];
  
  // 基础效果已在 app.config.ts 的 button 配置中定义
  // 只保留必要的增强样式
  
  // 渐变背景（可选增强）
  if (props.gradient && props.variant === 'solid') {
    switch (props.color) {
      case 'primary':
        classes.push('bg-gradient-to-r from-primary-500 to-blue-600');
        break;
      case 'success':
        classes.push('bg-gradient-to-r from-green-500 to-emerald-600');
        break;
      case 'error':
        classes.push('bg-gradient-to-r from-red-500 to-rose-600');
        break;
      case 'warning':
        classes.push('bg-gradient-to-r from-yellow-500 to-amber-600');
        break;
    }
  }
  
  // 成功状态样式
  if (isSuccess.value) {
    classes.push('ring-2 ring-green-200 dark:ring-green-800');
  }
  
  return classes.join(' ');
});

// 防抖和节流
let debounceTimeout: NodeJS.Timeout;
let isThrottled = false;

// 点击处理
const handleClick = async (event: MouseEvent) => {
  // 防抖处理
  if (props.debounce > 0) {
    clearTimeout(debounceTimeout);
    debounceTimeout = setTimeout(() => {
      executeClick(event);
    }, props.debounce);
    return;
  }
  
  // 节流处理
  if (props.throttle > 0) {
    const now = Date.now();
    if (isThrottled || (now - lastClickTime.value < props.throttle)) {
      return;
    }
    isThrottled = true;
    lastClickTime.value = now;
    
    setTimeout(() => {
      isThrottled = false;
    }, props.throttle);
  }
  
  await executeClick(event);
};

// 执行点击
const executeClick = async (event: MouseEvent) => {
  try {
    // 二次确认
    if (props.confirmRequired) {
      const confirmed = confirm(props.confirmMessage);
      if (!confirmed) return;
    }
    
    // 智能加载状态
    if (props.smartLoading) {
      isLoading.value = true;
      
      // 自动结束加载
      if (props.loadingDuration > 0) {
        setTimeout(() => {
          isLoading.value = false;
          
          // 显示成功状态
          if (props.showSuccessState) {
            showSuccessState();
          }
        }, props.loadingDuration);
      }
    }
    
    emit('click', event);
  } catch (error) {
    isLoading.value = false;
    emit('error', error);
  }
};

// 显示成功状态
const showSuccessState = () => {
  isSuccess.value = true;
  showSuccessIcon.value = true;
  
  setTimeout(() => {
    isSuccess.value = false;
    showSuccessIcon.value = false;
  }, props.successDuration);
  
  emit('success');
};

// 手动控制加载状态
const setLoading = (loading: boolean) => {
  isLoading.value = loading;
};

// 手动触发成功状态
const triggerSuccess = () => {
  isLoading.value = false;
  showSuccessState();
};

// 手动触发错误状态
const triggerError = (error?: any) => {
  isLoading.value = false;
  emit('error', error);
};

// 暴露方法
defineExpose({
  setLoading,
  triggerSuccess,
  triggerError,
  isLoading: readonly(isLoading),
  isSuccess: readonly(isSuccess)
});

// 清理定时器
onUnmounted(() => {
  if (debounceTimeout) {
    clearTimeout(debounceTimeout);
  }
});
</script>
