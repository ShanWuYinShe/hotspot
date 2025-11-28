<template>
  <component
    :is="componentTag"
    :to="to"
    :href="href"
    :class="cardClasses"
    :style="cardStyles"
    @click="handleClick"
    @mouseenter="handleMouseEnter"
    @mouseleave="handleMouseLeave"
  >
    <slot />
  </component>
</template>

<script setup lang="ts">
interface Props {
  to?: string;
  href?: string;
  hover?: boolean;
  active?: boolean;
  disabled?: boolean;
  glow?: boolean;
  bordered?: boolean;
  shadow?: 'none' | 'sm' | 'md' | 'lg' | 'xl';
  padding?: 'none' | 'sm' | 'md' | 'lg' | 'xl';
}

interface Emits {
  click: [event: MouseEvent];
}

const props = withDefaults(defineProps<Props>(), {
  hover: true,
  active: true,
  glow: false,
  bordered: true,
  shadow: 'md',
  padding: 'md',
});

const emit = defineEmits<Emits>();

// 组件标签（根据是否有to/href决定）
const componentTag = computed(() => {
  if (props.to) return resolveComponent('NuxtLink');
  if (props.href) return 'a';
  return 'div';
});

// 卡片样式类
const cardClasses = computed(() => {
  const classes = [
    'interactive-card',
    'rounded-xl',
    'bg-white dark:bg-neutral-800',
    'transition-all duration-200',
  ];

  // 边框
  if (props.bordered) {
    classes.push('border border-neutral-200 dark:border-neutral-700');
  }

  // 阴影
  if (props.shadow !== 'none') {
    const shadowMap = {
      sm: 'shadow-sm',
      md: 'shadow-md',
      lg: 'shadow-lg',
      xl: 'shadow-xl',
    };
    classes.push(shadowMap[props.shadow]);
  }

  // 内边距
  if (props.padding !== 'none') {
    const paddingMap = {
      sm: 'p-3',
      md: 'p-4',
      lg: 'p-6',
      xl: 'p-8',
    };
    classes.push(paddingMap[props.padding]);
  }

  // 悬停效果
  if (props.hover && !props.disabled) {
    classes.push('hover:shadow-lg hover:-translate-y-1');
  }

  // 禁用状态
  if (props.disabled) {
    classes.push('opacity-60 cursor-not-allowed');
  } else {
    classes.push('cursor-pointer');
  }

  // 发光效果
  if (props.glow && !props.disabled) {
    classes.push('hover:ring-2 hover:ring-primary-500/50 hover:ring-offset-2');
  }

  return classes;
});

// 卡片内联样式（用于动画）
const cardStyles = ref({});

// 点击处理
const handleClick = (event: MouseEvent) => {
  if (props.disabled) {
    event.preventDefault();
    return;
  }
  
  if (props.active) {
    // 点击波纹效果
    createRipple(event);
  }
  
  emit('click', event);
};

// 鼠标进入
const handleMouseEnter = () => {
  if (props.disabled) return;
  // 可以添加额外的鼠标进入效果
};

// 鼠标离开
const handleMouseLeave = () => {
  if (props.disabled) return;
  // 可以添加额外的鼠标离开效果
};

// 创建点击波纹效果
const createRipple = (event: MouseEvent) => {
  const card = event.currentTarget as HTMLElement;
  const ripple = document.createElement('span');
  const diameter = Math.max(card.clientWidth, card.clientHeight);
  const radius = diameter / 2;

  const rect = card.getBoundingClientRect();
  const x = event.clientX - rect.left - radius;
  const y = event.clientY - rect.top - radius;

  ripple.style.width = ripple.style.height = `${diameter}px`;
  ripple.style.left = `${x}px`;
  ripple.style.top = `${y}px`;
  ripple.classList.add('ripple');

  const existingRipple = card.querySelector('.ripple');
  if (existingRipple) {
    existingRipple.remove();
  }

  card.appendChild(ripple);

  setTimeout(() => {
    ripple.remove();
  }, 600);
};
</script>

<style scoped>
.interactive-card {
  position: relative;
  overflow: hidden;
}

.interactive-card::before {
  content: '';
  position: absolute;
  top: 0;
  left: -100%;
  width: 100%;
  height: 100%;
  background: linear-gradient(
    90deg,
    transparent,
    rgba(255, 255, 255, 0.1),
    transparent
  );
  transition: left 0.5s;
}

.interactive-card:hover::before {
  left: 100%;
}

.dark .interactive-card::before {
  background: linear-gradient(
    90deg,
    transparent,
    rgba(255, 255, 255, 0.05),
    transparent
  );
}

/* 点击波纹效果 */
:deep(.ripple) {
  position: absolute;
  border-radius: 50%;
  transform: scale(0);
  animation: ripple-animation 0.6s ease-out;
  background-color: rgba(255, 255, 255, 0.6);
  pointer-events: none;
}

.dark :deep(.ripple) {
  background-color: rgba(255, 255, 255, 0.2);
}

@keyframes ripple-animation {
  to {
    transform: scale(4);
    opacity: 0;
  }
}

/* 禁用动画（用于减少动画偏好） */
@media (prefers-reduced-motion: reduce) {
  .interactive-card {
    transition: none !important;
  }

  .interactive-card::before {
    transition: none;
  }

  .interactive-card:hover {
    transform: none !important;
  }

  :deep(.ripple) {
    animation: none;
    opacity: 0;
  }
}
</style>



