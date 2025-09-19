<template>
  <UModal title="PIN验证" :prevent-close="true">
    <template #header>
      <div class="flex items-center justify-between">
        <h3 class="text-lg font-semibold">
          {{ isSetup ? '设置设备PIN' : '输入设备PIN' }}
        </h3>
      </div>
    </template>

    <template #body>
      <div class="space-y-4">
        <p class="text-sm text-gray-600 dark:text-gray-400">
          {{ isSetup 
            ? '请设置4位数字PIN以保护账户切换功能' 
            : '请输入4位数字PIN以解锁账户切换功能' 
          }}
        </p>

        <div class="flex justify-center">
          <UPinInput
            v-model="pinValue"
            :length="4"
            mask
            :error="hasError"
            placeholder="●"
            @complete="handlePinComplete"
          />
        </div>

        <div v-if="hasError" class="text-center">
          <p class="text-sm text-red-600 dark:text-red-400">
            {{ errorMessage }}
          </p>
        </div>

        <div v-if="isSetup" class="text-center">
          <p class="text-xs text-gray-500 dark:text-gray-400">
            建议使用好记的数字组合，如生日或纪念日
          </p>
        </div>
      </div>
    </template>

    <template #footer>
      <div class="flex justify-end space-x-3">
        <UButton 
          variant="ghost" 
          @click="handleCancel"
          :disabled="isProcessing"
        >
          取消
        </UButton>
        <UButton 
          :disabled="!isComplete || isProcessing"
          :loading="isProcessing"
          @click="handleConfirm"
        >
          {{ isSetup ? '设置' : '确认' }}
        </UButton>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
interface Props {
  isSetup?: boolean
}

interface Emits {
  (e: 'confirm', pin: string): void
  (e: 'cancel'): void
  (e: 'close'): void
}

const props = withDefaults(defineProps<Props>(), {
  isSetup: false
})

const emit = defineEmits<Emits>()

// useOverlay 会自动管理 Modal 状态，无需手动绑定 isOpen

const pinValue = ref<string[]>(['', '', '', ''])
const hasError = ref(false)
const errorMessage = ref('')
const isProcessing = ref(false)

// 计算属性
const isComplete = computed(() => 
  pinValue.value.length === 4 && pinValue.value.every(digit => /^\d$/.test(digit))
)

// 计算完整的PIN字符串
const pinString = computed(() => pinValue.value.join(''))

// 组件挂载时重置表单
onMounted(() => {
  resetForm()
})

// 监听PIN值变化，清除错误状态
watch(pinValue, () => {
  if (hasError.value) {
    hasError.value = false
    errorMessage.value = ''
  }
})

// 处理PIN输入完成
const handlePinComplete = (pins: string[]) => {
  pinValue.value = pins
}

// 方法
function resetForm() {
  pinValue.value = ['', '', '', '']
  hasError.value = false
  errorMessage.value = ''
  isProcessing.value = false
}


async function handleConfirm() {
  if (!isComplete.value || isProcessing.value) return
  
  isProcessing.value = true
  
  try {
    emit('confirm', pinString.value)
    emit('close') // 通知 useOverlay 关闭
  } catch (error) {
    hasError.value = true
    errorMessage.value = '处理失败，请重试'
    isProcessing.value = false
  }
}

function handleCancel() {
  if (isProcessing.value) return
  emit('cancel')
  emit('close') // 通知 useOverlay 关闭
}

function showError(message: string) {
  hasError.value = true
  errorMessage.value = message
  isProcessing.value = false
  
  // 清空输入
  resetForm()
}

// 暴露方法给父组件
defineExpose({
  showError,
  resetForm
})
</script>

<style scoped>
/* 确保输入框在所有浏览器中都有一致的样式 */
input[type="password"]::-ms-reveal {
  display: none;
}

input[type="password"]::-webkit-textfield-decoration-container {
  display: none;
}
</style>


