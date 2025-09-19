<template>
  <div class="space-y-4 md:space-y-6" role="group" aria-labelledby="about-heading">
    <UCard :ui="cardDefaultUi">
      <template #header>
        <div class="flex items-center gap-4 md:gap-5">
          <div class="p-4 md:p-5 rounded-xl bg-primary-100 dark:bg-primary-900/40 flex-shrink-0">
            <UIcon name="i-heroicons-information-circle" class="w-7 h-7 md:w-9 md:h-9 text-primary-600 dark:text-primary-400" />
          </div>
          <div class="min-w-0 flex-1">
            <h2 
              id="about-heading" 
              tabindex="-1" 
              class="text-lg md:text-xl font-semibold text-neutral-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-primary-500 rounded-sm"
            >
              关于应用
            </h2>
            <p class="text-xs md:text-sm text-neutral-600 dark:text-neutral-400">
              应用信息和支持选项
            </p>
          </div>
        </div>
      </template>

      <div class="space-y-3 md:space-y-4 p-1 pb-4 md:pb-6">
        <!-- 应用信息 -->
        <div class="text-center py-4 md:py-6">
          <div class="w-16 h-16 md:w-20 md:h-20 lg:w-24 lg:h-24 bg-primary-100 dark:bg-primary-900/40 rounded-2xl flex items-center justify-center mx-auto mb-3 md:mb-4">
            <UIcon name="i-heroicons-map-pin" class="w-8 h-8 md:w-10 md:h-10 lg:w-12 lg:h-12 text-primary-600 dark:text-primary-400" />
          </div>
          <h3 class="text-lg md:text-xl lg:text-2xl font-bold text-neutral-900 dark:text-white mb-1 md:mb-2">
            HotSpot
          </h3>
          <p class="text-xs md:text-sm text-neutral-600 dark:text-neutral-400 mb-3 md:mb-4">
            基于位置的社交平台
          </p>
          <UBadge label="v1.0.0" color="primary" variant="soft" />
        </div>

        <USeparator />

        <!-- 支持选项 -->
        <div class="space-y-2 md:space-y-3">
          <UButton 
            variant="ghost" 
            block 
            justify="start" 
            @click="handleClearCache" 
            class="h-12 md:h-14 lg:h-16"
          >
            <div class="p-3 md:p-4 rounded-lg bg-neutral-100 dark:bg-neutral-800 mr-3 md:mr-4 flex-shrink-0">
              <UIcon name="i-heroicons-trash" class="w-5 h-5 md:w-6 md:h-6" />
            </div>
            <div class="text-left min-w-0 flex-1">
              <div class="font-medium text-sm md:text-base">清除缓存</div>
              <div class="text-xs md:text-sm text-neutral-600">清除应用缓存数据</div>
            </div>
          </UButton>
          
          <UButton 
            variant="ghost" 
            block 
            justify="start" 
            @click="handleShowHelp" 
            class="h-12 md:h-14 lg:h-16"
          >
            <div class="p-3 md:p-4 rounded-lg bg-neutral-100 dark:bg-neutral-800 mr-3 md:mr-4 flex-shrink-0">
              <UIcon name="i-heroicons-question-mark-circle" class="w-5 h-5 md:w-6 md:h-6" />
            </div>
            <div class="text-left min-w-0 flex-1">
              <div class="font-medium text-sm md:text-base">帮助与支持</div>
              <div class="text-xs md:text-sm text-neutral-600">获取使用帮助</div>
            </div>
          </UButton>
          
          <UButton 
            variant="ghost" 
            block 
            justify="start" 
            @click="handleShowPrivacyPolicy" 
            class="h-12 md:h-14 lg:h-16"
          >
            <div class="p-3 md:p-4 rounded-lg bg-neutral-100 dark:bg-neutral-800 mr-3 md:mr-4 flex-shrink-0">
              <UIcon name="i-heroicons-document-text" class="w-5 h-5 md:w-6 md:h-6" />
            </div>
            <div class="text-left min-w-0 flex-1">
              <div class="font-medium text-sm md:text-base">隐私政策</div>
              <div class="text-xs md:text-sm text-neutral-600">查看隐私政策</div>
            </div>
          </UButton>
          
          <UButton 
            variant="ghost" 
            block 
            justify="start" 
            @click="handleShowTerms" 
            class="h-12 md:h-14 lg:h-16"
          >
            <div class="p-3 md:p-4 rounded-lg bg-neutral-100 dark:bg-neutral-800 mr-3 md:mr-4 flex-shrink-0">
              <UIcon name="i-heroicons-clipboard-document-list" class="w-5 h-5 md:w-6 md:h-6" />
            </div>
            <div class="text-left min-w-0 flex-1">
              <div class="font-medium text-sm md:text-base">服务条款</div>
              <div class="text-xs md:text-sm text-neutral-600">查看服务条款</div>
            </div>
          </UButton>
        </div>
      </div>
    </UCard>
  </div>
</template>

<script setup lang="ts">
interface Props {
}

defineProps<Props>();

const emit = defineEmits<{
  clearCache: [];
  showHelp: [];
  showPrivacyPolicy: [];
  showTerms: [];
}>();

// 事件处理
const handleClearCache = () => emit('clearCache');
const handleShowHelp = () => emit('showHelp');
const handleShowPrivacyPolicy = () => emit('showPrivacyPolicy');
const handleShowTerms = () => emit('showTerms');

// UI 配置
const cardDefaultUi = {
  root: 'p-4 md:p-6 rounded-xl border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-800',
  body: 'p-0'
};
</script>