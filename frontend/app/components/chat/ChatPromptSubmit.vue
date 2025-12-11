<template>
  <div class="flex items-center gap-1">
    <!-- 发送中状态：显示停止按钮 -->
    <UButton
      v-if="status === 'streaming' || status === 'submitted'"
      color="neutral"
      variant="ghost"
      icon="i-heroicons-stop"
      size="sm"
      @click="emit('stop')"
    />
    
    <!-- 错误状态：显示重试按钮 -->
    <UButton
      v-else-if="status === 'error'"
      color="warning"
      variant="ghost"
      icon="i-heroicons-arrow-path"
      size="sm"
      @click="emit('reload')"
    />
    
    <!-- 默认状态：显示发送按钮 -->
    <UButton
      v-else
      color="primary"
      icon="i-heroicons-paper-airplane"
      size="sm"
      :disabled="disabled"
      @click="emit('submit')"
    />
  </div>
</template>

<script setup lang="ts">
interface Props {
  status?: 'idle' | 'streaming' | 'submitted' | 'ready' | 'error';
  disabled?: boolean;
}

withDefaults(defineProps<Props>(), {
  status: 'idle',
  disabled: false,
});

const emit = defineEmits<{
  submit: [];
  stop: [];
  reload: [];
}>();
</script>

