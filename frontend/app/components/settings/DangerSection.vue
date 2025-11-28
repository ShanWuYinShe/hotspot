<template>
  <div class="space-y-4 md:space-y-6" role="group" aria-labelledby="danger-heading">
    <UCard :ui="cardDangerUi">
      <template #header>
        <div class="flex items-center gap-4 md:gap-5">
          <div class="p-4 md:p-5 rounded-xl bg-red-100 dark:bg-red-900/40 border border-red-200 dark:border-red-700 flex-shrink-0">
            <UIcon name="i-heroicons-exclamation-triangle" class="w-7 h-7 md:w-9 md:h-9 text-error-600 dark:text-error-400" />
          </div>
          <div class="min-w-0 flex-1">
            <h2 
              id="danger-heading" 
              tabindex="-1" 
              class="text-lg md:text-xl font-semibold text-neutral-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-error-500 rounded-sm"
            >
              危险操作
            </h2>
            <p class="text-xs md:text-sm text-neutral-600 dark:text-neutral-400">
              这些操作不可逆转，请谨慎操作
            </p>
          </div>
        </div>
      </template>

      <div class="space-y-3 md:space-y-4 p-1 pb-4 md:pb-6">
        <UAlert 
          color="error" 
          variant="soft" 
          title="警告" 
          description="以下操作将永久删除您的数据，且无法恢复。请确保您已备份重要信息。" 
        />
        
        <div class="space-y-3 md:space-y-4">
          <UCard :ui="dangerItemUi">
            <div class="flex flex-col sm:flex-row items-start gap-3 md:gap-4">
              <div class="p-4 md:p-5 rounded-lg bg-red-100 dark:bg-red-900/30 border border-red-200 dark:border-red-600 flex-shrink-0">
                <UIcon name="i-heroicons-trash" class="w-6 h-6 md:w-7 md:h-7 text-error-600 dark:text-error-400" />
              </div>
              <div class="flex-1 min-w-0">
                <h4 class="text-base md:text-lg font-medium text-neutral-900 dark:text-white">
                  删除账户
                </h4>
                <p class="text-xs md:text-sm text-neutral-700 dark:text-neutral-300 mt-1">
                  永久删除您的账户和所有相关数据，包括消息、群组、好友关系等。此操作不可撤销。
                </p>
                <UButton 
                  color="error" 
                  variant="solid" 
                  @click="handleDeleteAccount" 
                  class="mt-3 md:mt-4 transition-colors duration-200" 
                  size="sm"
                >
                  删除我的账户
                </UButton>
              </div>
            </div>
          </UCard>
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
  deleteAccount: [];
}>();

// 事件处理
const handleDeleteAccount = () => emit('deleteAccount');

// UI 配置
const cardDangerUi = {
  root: 'p-4 md:p-6 rounded-xl border border-error-200 dark:border-error-800 bg-white dark:bg-neutral-800',
  body: 'p-0'
};

const dangerItemUi = {
  root: 'p-4 md:p-5 lg:p-6 border-2 border-error-200 dark:border-error-800 rounded-lg bg-error-50 dark:bg-error-900/10',
  body: 'p-0'
};
</script>