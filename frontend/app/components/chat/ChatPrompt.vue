<template>
  <div class="border-t border-default bg-card-elevated p-4">
    <!-- 错误提示 -->
    <div v-if="error" class="mb-2 p-2 bg-error-light text-error text-sm rounded-lg flex items-center gap-2">
      <UIcon name="i-heroicons-exclamation-triangle" class="icon-sm shrink-0" />
      <span>{{ error.message }}</span>
    </div>

    <!-- 输入区域 -->
    <div class="flex items-end gap-2">
      <!-- 左侧工具栏插槽 -->
      <div v-if="$slots.leading" class="shrink-0">
        <slot name="leading" />
      </div>

      <!-- 输入框 -->
      <div class="flex-1 relative">
        <UTextarea
          :model-value="modelValue"
          :placeholder="placeholder"
          :disabled="disabled"
          :rows="rows"
          :autoresize="autoresize"
          class="w-full"
          @update:model-value="emit('update:modelValue', $event)"
          @keydown.enter.exact.prevent="handleSubmit"
          @keydown.enter.shift.stop
        />
      </div>

      <!-- 右侧按钮插槽 (ChatPromptSubmit) -->
      <div v-if="$slots.default" class="shrink-0">
        <slot />
      </div>

      <!-- 默认发送按钮 -->
      <UButton
        v-else
        color="primary"
        :disabled="!canSubmit"
        :loading="loading"
        icon="i-heroicons-paper-airplane"
        @click="handleSubmit"
      />
    </div>

    <!-- 底部提示 -->
    <div v-if="$slots.footer" class="mt-2 text-xs text-secondary-content">
      <slot name="footer" />
    </div>
  </div>
</template>

<script setup lang="ts">
interface Props {
  modelValue?: string;
  placeholder?: string;
  disabled?: boolean;
  loading?: boolean;
  error?: Error | null;
  rows?: number;
  autoresize?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: '',
  placeholder: '输入消息...',
  disabled: false,
  loading: false,
  rows: 1,
  autoresize: true,
});

const emit = defineEmits<{
  'update:modelValue': [value: string];
  'submit': [];
}>();

// 是否可以提交
const canSubmit = computed(() => {
  return props.modelValue?.trim().length > 0 && !props.disabled && !props.loading;
});

// 提交处理
const handleSubmit = () => {
  if (canSubmit.value) {
    emit('submit');
  }
};
</script>

