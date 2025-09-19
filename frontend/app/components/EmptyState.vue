<template>
  <div class="flex flex-col items-center justify-center py-8 md:py-12 lg:py-16 space-y-4 md:space-y-6">
    <!-- 图标或插图 -->
    <div class="relative">
      <!-- 自定义插图插槽 -->
      <div v-if="$slots.illustration" class="mb-4">
        <slot name="illustration" />
      </div>
      <!-- 默认图标 -->
      <div v-else class="w-16 h-16 md:w-20 md:h-20 lg:w-24 lg:h-24 rounded-full bg-neutral-100 dark:bg-neutral-800 flex items-center justify-center">
        <UIcon 
          :name="icon" 
          class="w-8 h-8 md:w-10 md:h-10 lg:w-12 lg:h-12 text-neutral-400 dark:text-neutral-500" 
        />
        <!-- 装饰性元素 -->
        <div class="absolute -top-1 -right-1 w-4 h-4 md:w-5 md:h-5 bg-primary-100 dark:bg-primary-900/50 rounded-full"></div>
        <div class="absolute -bottom-2 -left-2 w-3 h-3 md:w-4 md:h-4 bg-secondary-100 dark:bg-secondary-900/50 rounded-full"></div>
      </div>
    </div>
    
    <!-- 标题和描述 -->
    <div class="text-center space-y-2 max-w-sm mx-auto px-4">
      <h3 class="text-base md:text-lg lg:text-xl font-semibold text-neutral-800 dark:text-neutral-200">
        {{ title }}
      </h3>
      <p class="text-sm md:text-base text-neutral-600 dark:text-neutral-400 leading-relaxed">
        {{ description }}
      </p>
    </div>
    
    <!-- 操作按钮 -->
    <div v-if="actionText" class="pt-2">
      <UButton 
        :variant="actionVariant"
        :color="actionColor"
        :size="actionSize"
        @click="$emit('action')"
        class="transition-colors duration-200"
      >
        <template v-if="actionIcon" #leading>
          <UIcon :name="actionIcon" class="w-4 h-4" />
        </template>
        {{ actionText }}
      </UButton>
    </div>
    
    <!-- 自定义内容插槽 -->
    <div v-if="$slots.default" class="pt-2">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
interface Props {
  icon: string;
  title: string;
  description: string;
  actionText?: string;
  actionIcon?: string;
  actionVariant?: 'solid' | 'outline' | 'soft' | 'ghost';
  actionColor?: 'primary' | 'secondary' | 'success' | 'info' | 'warning' | 'error' | 'neutral';
  actionSize?: 'xs' | 'sm' | 'md' | 'lg' | 'xl';
}

withDefaults(defineProps<Props>(), {
  actionVariant: 'soft',
  actionColor: 'primary',
  actionSize: 'sm'
});

defineEmits<{
  action: [];
}>();
</script>