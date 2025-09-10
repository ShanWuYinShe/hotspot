<template>
  <UModal :title="title">
    <template #body>
      <p class="text-neutral-600 dark:text-neutral-300">{{ message }}</p>
    </template>
    
    <template #footer>
      <div class="flex justify-end space-x-3">
        <UButton variant="ghost" @click="handleCancel">{{ cancelText }}</UButton>
        <UButton :color="confirmColor" :loading="loading" @click="handleConfirm">{{ confirmText }}</UButton>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
// Props - 移除 open 属性，由 useOverlay 管理
interface Props {
  title?: string;
  message?: string;
  confirmText?: string;
  cancelText?: string;
  confirmColor?: 'primary' | 'error' | 'warning' | 'success';
  loading?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  title: '确认操作',
  message: '确定要执行此操作吗？',
  confirmText: '确认',
  cancelText: '取消',
  confirmColor: 'primary',
  loading: false
});

// Emits - 移除 update:open，由 useOverlay 管理
const emit = defineEmits<{
  'confirm': [];
  'cancel': [];
  'close': [];
}>();

// 处理函数
const handleConfirm = () => {
  emit('confirm');
  emit('close'); // 通知 useOverlay 关闭
};

const handleCancel = () => {
  emit('cancel');
  emit('close'); // 通知 useOverlay 关闭
};
</script>

