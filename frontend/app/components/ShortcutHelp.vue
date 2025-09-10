<template>
  <UModal 
    :title="t('shortcuts.help.title')"
    size="lg"
  >
    <template #header>
      <div class="flex items-center gap-2">
        <UIcon name="i-heroicons-command-line" class="w-5 h-5 text-primary-500" />
        <span>{{ t('shortcuts.help.title') }}</span>
      </div>
    </template>

    <template #body>
      <div class="space-y-6">
        <div 
          v-for="(shortcuts, category) in shortcutsByCategory" 
          :key="category"
          class="space-y-3"
        >
          <h3 class="text-lg font-semibold text-neutral-900 dark:text-white border-b border-neutral-200 dark:border-neutral-700 pb-2">
            {{ category }}
          </h3>
          
          <div class="space-y-2">
            <div 
              v-for="shortcut in shortcuts" 
              :key="shortcut.key"
              class="flex items-center justify-between py-2 px-3 rounded-lg hover:bg-neutral-50 dark:hover:bg-neutral-800 transition-colors"
            >
              <span class="text-neutral-700 dark:text-neutral-300">
                {{ shortcut.description }}
              </span>
              
              <div class="flex items-center gap-1">
                <UKbd 
                  v-for="key in formatShortcut(shortcut.key)" 
                  :key="key"
                  size="sm"
                >
                  {{ key }}
                </UKbd>
              </div>
            </div>
          </div>
        </div>

        <div v-if="Object.keys(shortcutsByCategory).length === 0" class="text-center py-8">
          <UIcon name="i-heroicons-command-line" class="w-12 h-12 text-neutral-400 mx-auto mb-3" />
          <p class="text-neutral-500">{{ t('shortcuts.help.empty') }}</p>
        </div>
      </div>
    </template>

    <template #footer>
      <div class="flex justify-between items-center">
        <div class="text-sm text-neutral-500">
          {{ t('shortcuts.help.tip') }}
        </div>
        <UButton @click="handleClose" color="primary">
          {{ t('common.close') }}
        </UButton>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
const { t } = useI18n()
const { getShortcutsByCategory, formatShortcut, isHelpVisible } = useGlobalShortcuts()

// 处理关闭
const handleClose = () => {
  const { hideHelp } = useGlobalShortcuts()
  hideHelp()
}

const shortcutsByCategory = getShortcutsByCategory
</script>
