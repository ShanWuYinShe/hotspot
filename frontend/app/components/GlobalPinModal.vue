<template>
  <!-- 全局 PIN 模态框管理器 - 用于解密本地token数据 -->
</template>

<script setup lang="ts">
// 使用 useOverlay 管理 PIN 输入模态框
const overlay = useOverlay()
const authStore = useAuthStore()

// PIN码用于解密本地token数据，在需要访问加密数据时主动触发
const pinInputModal = overlay.create(defineAsyncComponent(() => import('~/components/PinInputModalContent.vue')))

// 当需要解密本地token数据时调用此方法
const requestPinForDecryption = async (): Promise<string> => {
  return new Promise((resolve, reject) => {
    pinInputModal.open({ 
      isSetup: false,
      onConfirm: (pin: string) => {
        pinInputModal.close()
        resolve(pin)
      },
      onCancel: () => {
        pinInputModal.close()
        reject(new Error('用户取消了PIN输入'))
      }
    })
  })
}

// 当需要设置PIN码时调用此方法
const requestPinForSetup = async (): Promise<string> => {
  return new Promise((resolve, reject) => {
    pinInputModal.open({ 
      isSetup: true,
      onConfirm: (pin: string) => {
        pinInputModal.close()
        resolve(pin)
      },
      onCancel: () => {
        pinInputModal.close()
        reject(new Error('用户取消了PIN设置'))
      }
    })
  })
}

// 暴露方法供其他组件使用
defineExpose({
  requestPinForDecryption,
  requestPinForSetup
})
</script>
