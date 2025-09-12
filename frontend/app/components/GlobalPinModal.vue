<template>
  <!-- 全局 PIN 模态框管理器 - 无需渲染内容，仅负责逻辑管理 -->
</template>

<script setup lang="ts">
// 使用 useOverlay 管理 PIN 输入模态框
const overlay = useOverlay()
const authStore = useAuthStore()

// 懒加载 PIN 输入模态框内容组件
const PinInputModalContent = defineAsyncComponent(() => import('~/components/PinInputModalContent.vue'))

// 创建 PIN 输入模态框实例
const pinInputModal = overlay.create(PinInputModalContent)

// 监听 authStore 的 PIN 模态框状态
watch(() => authStore.pinModalState.isOpen, (isOpen) => {
  if (isOpen) {
    // 打开模态框并传递参数
    pinInputModal.open({
      isSetup: authStore.pinModalState.isSetup
    })
    
    // 设置事件处理器
    pinInputModal.patch({
      onConfirm: (pin: string) => {
        authStore.handlePinConfirm(pin)
      },
      onCancel: () => {
        authStore.handlePinCancel()
      }
    })
  } else {
    // 关闭模态框
    pinInputModal.close()
  }
}, { immediate: true })
</script>
