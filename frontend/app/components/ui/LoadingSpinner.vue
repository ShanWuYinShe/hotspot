<template>
  <div :class="containerClasses">
    <div :class="spinnerClasses">
      <div class="spinner-circle"></div>
      <div class="spinner-circle"></div>
      <div class="spinner-circle"></div>
    </div>
    <p v-if="text" :class="textClasses">{{ text }}</p>
  </div>
</template>

<script setup lang="ts">
interface Props {
  size?: 'sm' | 'md' | 'lg';
  text?: string;
  center?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  size: 'md',
  center: true,
});

// 容器样式
const containerClasses = computed(() => [
  'loading-spinner-container',
  {
    'flex flex-col items-center justify-center gap-3': props.center,
    'inline-flex': !props.center,
  }
]);

// 加载器样式
const spinnerClasses = computed(() => [
  'loading-spinner',
  `loading-spinner-${props.size}`,
]);

// 文本样式
const textClasses = computed(() => [
  'text-muted text-center',
  {
    'text-sm': props.size === 'sm',
    'text-base': props.size === 'md',
    'text-lg': props.size === 'lg',
  }
]);
</script>

<style scoped>
.loading-spinner {
  display: inline-flex;
  gap: 0.5rem;
  align-items: flex-end;
}

.spinner-circle {
  background-color: currentColor;
  border-radius: 50%;
  animation: loading-bounce 1.4s infinite ease-in-out both;
}

.spinner-circle:nth-child(1) {
  animation-delay: -0.32s;
}

.spinner-circle:nth-child(2) {
  animation-delay: -0.16s;
}

/* 大小变体 */
.loading-spinner-sm .spinner-circle {
  width: 0.5rem;
  height: 0.5rem;
}

.loading-spinner-md .spinner-circle {
  width: 0.75rem;
  height: 0.75rem;
}

.loading-spinner-lg .spinner-circle {
  width: 1rem;
  height: 1rem;
}

/* 弹跳动画 */
@keyframes loading-bounce {
  0%, 80%, 100% {
    transform: scale(0);
    opacity: 0.5;
  }
  40% {
    transform: scale(1);
    opacity: 1;
  }
}
</style>



