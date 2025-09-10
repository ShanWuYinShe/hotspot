<template>
  <UModal>
    <template #header>
      <div class="flex items-center gap-2">
        <UIcon name="i-heroicons-arrow-path" class="w-5 h-5 text-warning-500" />
        <h3 class="text-lg font-semibold">{{ t('userMenu.switchModal.title') }}</h3>
      </div>
    </template>

    <template #body>
      <div class="space-y-4">
        <p class="text-neutral-600 dark:text-neutral-400">
          {{ t('userMenu.switchModal.description', { name: account?.nickname }) }}
        </p>

        <UAlert 
          v-if="account?.is_temp"
          icon="i-heroicons-exclamation-triangle"
          color="warning"
          variant="soft"
          :title="t('userMenu.switchModal.tempWarningTitle')"
          :description="t('userMenu.switchModal.tempWarningDescription')"
        />

        <div class="flex justify-end gap-3 pt-4">
          <UButton 
            @click="handleClose" 
            variant="ghost"
            color="neutral"
          >
            {{ t('userMenu.switchModal.cancel') }}
          </UButton>
          <UButton 
            @click="handleConfirm" 
            :loading="isSwitching"
            color="warning"
            icon="i-heroicons-arrow-path"
          >
            {{ t('userMenu.switchModal.confirm') }}
          </UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
interface Props {
  account?: any
  onConfirm?: () => Promise<void>
}

const props = defineProps<Props>()
const emit = defineEmits<{
  close: []
}>()

const { t } = useI18n()

// useOverlay 会自动管理 Modal 状态，无需手动绑定 isOpen

// 响应式状态
const isSwitching = ref(false)

// 处理关闭
const handleClose = () => {
  emit('close')
}

// 处理确认切换
const handleConfirm = async () => {
  if (!props.onConfirm) {
    console.error('onConfirm callback not provided')
    return
  }

  try {
    isSwitching.value = true
    await props.onConfirm()
    handleClose()
  } catch (error) {
    // 错误已在父组件处理，这里不需要额外处理
    console.error('切换账户失败:', error)
  } finally {
    isSwitching.value = false
  }
}
</script>

