<template>
  <UCard :ui="cardUi">
    <div class="flex items-center justify-between gap-3 md:gap-4">
      <div class="flex items-center gap-2 md:gap-3 min-w-0 flex-1">
        <div :class="iconWrapperClass">
          <UIcon :name="icon" :class="iconClass" />
        </div>
        <div class="min-w-0 flex-1">
          <div class="font-medium text-sm md:text-base text-neutral-900 dark:text-white">
            {{ title }}
          </div>
          <div v-if="description" class="text-xs md:text-sm text-neutral-600 dark:text-neutral-400 mt-0.5">
            {{ description }}
          </div>
        </div>
      </div>
      <div class="flex-shrink-0">
        <UTooltip :text="model ? `关闭 ${title}` : `开启 ${title}`">
          <USwitch 
            v-model="model" 
            size="md"
            :color="switchColor"
          />
        </UTooltip>
      </div>
    </div>
  </UCard>
</template>

<script setup lang="ts">
interface Props {
  title: string;
  icon: string;
  description?: string;
  modelValue: boolean;
  color?: 'blue' | 'green' | 'purple' | 'yellow' | 'neutral' | 'red' | 'pink';
}

const props = withDefaults(defineProps<Props>(), {
  color: 'neutral'
});

const emit = defineEmits<{
  'update:modelValue': [value: boolean];
}>();

// 双向绑定
const model = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
});


const switchColor = computed(() => {
  const colorMap: Record<string, 'neutral' | 'primary' | 'secondary' | 'success' | 'info' | 'warning' | 'error'> = {
    blue: 'primary',
    green: 'success',
    purple: 'primary',
    yellow: 'warning',
    neutral: 'primary',
    red: 'error',
    pink: 'primary'
  };
  return colorMap[props.color] || 'primary';
});

// 颜色配置
const colorConfig = {
  blue: {
    wrapper: 'bg-primary-100 dark:bg-blue-900/50 border border-blue-200 dark:border-blue-700',
    icon: 'text-primary-600 dark:text-blue-300'
  },
  green: {
    wrapper: 'bg-green-100 dark:bg-green-900/50 border border-green-200 dark:border-green-700',
    icon: 'text-green-600 dark:text-green-300'
  },
  purple: {
    wrapper: 'bg-purple-100 dark:bg-purple-900/50 border border-purple-200 dark:border-purple-700',
    icon: 'text-purple-600 dark:text-purple-300'
  },
  yellow: {
    wrapper: 'bg-yellow-100 dark:bg-yellow-900/50 border border-yellow-200 dark:border-yellow-700',
    icon: 'text-yellow-600 dark:text-yellow-300'
  },
  neutral: {
    wrapper: 'bg-neutral-100 dark:bg-neutral-700 border border-neutral-200 dark:border-neutral-600',
    icon: 'text-neutral-600 dark:text-neutral-400'
  },
  red: {
    wrapper: 'bg-red-100 dark:bg-red-900/50 border border-red-200 dark:border-red-700',
    icon: 'text-red-600 dark:text-red-300'
  },
  pink: {
    wrapper: 'bg-pink-100 dark:bg-pink-900/50 border border-pink-200 dark:border-pink-700',
    icon: 'text-pink-600 dark:text-pink-300'
  }
};

const resolvedColor = computed(() => colorConfig[props.color]);

const iconWrapperClass = computed(() => 
  `p-3 md:p-4 rounded-lg flex-shrink-0 ${resolvedColor.value.wrapper}`
);

const iconClass = computed(() => 
  `w-5 h-5 md:w-6 md:h-6 ${resolvedColor.value.icon}`
);

// UI 配置
const cardUi = {
  root: 'p-3 md:p-4 rounded-lg bg-white/70 dark:bg-neutral-800/50 backdrop-blur-sm border border-neutral-200 dark:border-neutral-700 hover:bg-white/90 dark:hover:bg-neutral-700/70 transition-all duration-200',
  body: 'p-0'
};
</script>