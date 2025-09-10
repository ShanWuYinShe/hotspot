<template>
  <UModal v-model:open="isOpen" :prevent-close="true">
    <UCard>
      <template #header>
        <div class="flex items-center justify-between">
          <h3 class="text-lg font-semibold">
            {{ isSetup ? '设置设备PIN' : '输入设备PIN' }}
          </h3>
        </div>
      </template>

      <div class="space-y-4">
        <p class="text-sm text-gray-600 dark:text-gray-400">
          {{ isSetup 
            ? '请设置4位数字PIN以保护账户切换功能' 
            : '请输入4位数字PIN以解锁账户切换功能' 
          }}
        </p>

        <div class="flex justify-center space-x-2">
          <input
            v-for="(digit, index) in pinDigits"
            :key="index"
            :ref="el => pinInputs[index] = el"
            v-model="pinDigits[index]"
            type="password"
            inputmode="numeric"
            pattern="[0-9]"
            maxlength="1"
            class="w-12 h-12 text-center text-xl font-mono border-2 rounded-lg focus:border-primary-500 focus:ring-2 focus:ring-primary-200 dark:bg-gray-800 dark:border-gray-600"
            :class="{
              'border-red-500': hasError,
              'border-green-500': isComplete && !hasError
            }"
            @input="handleInput(index, $event)"
            @keydown="handleKeydown(index, $event)"
            @paste="handlePaste"
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
    </UCard>
  </UModal>
</template>

<script setup lang="ts">
interface Props {
  isSetup?: boolean
  open?: boolean
}

interface Emits {
  (e: 'update:open', value: boolean): void
  (e: 'confirm', pin: string): void
  (e: 'cancel'): void
}

const props = withDefaults(defineProps<Props>(), {
  isSetup: false,
  open: false
})

const emit = defineEmits<Emits>()

// 响应式状态
const isOpen = computed({
  get: () => props.open,
  set: (value) => emit('update:open', value)
})

const pinDigits = ref(['', '', '', ''])
const pinInputs = ref<HTMLInputElement[]>([])
const hasError = ref(false)
const errorMessage = ref('')
const isProcessing = ref(false)

// 计算属性
const isComplete = computed(() => 
  pinDigits.value.every(digit => digit.length === 1 && /^\d$/.test(digit))
)

const pinValue = computed(() => pinDigits.value.join(''))

// 监听器
watch(isOpen, (newValue) => {
  if (newValue) {
    resetForm()
    nextTick(() => {
      pinInputs.value[0]?.focus()
    })
  }
})

watch(pinDigits, () => {
  if (hasError.value) {
    hasError.value = false
    errorMessage.value = ''
  }
}, { deep: true })

// 方法
function resetForm() {
  pinDigits.value = ['', '', '', '']
  hasError.value = false
  errorMessage.value = ''
  isProcessing.value = false
}

function handleInput(index: number, event: Event) {
  const target = event.target as HTMLInputElement
  const value = target.value.replace(/\D/g, '') // 只保留数字
  
  if (value.length > 1) {
    // 处理粘贴或快速输入多个字符的情况
    const digits = value.slice(0, 4).split('')
    for (let i = 0; i < Math.min(digits.length, 4); i++) {
      if (index + i < 4) {
        pinDigits.value[index + i] = digits[i]
      }
    }
    
    // 聚焦到下一个空的输入框或最后一个
    const nextIndex = Math.min(index + digits.length, 3)
    pinInputs.value[nextIndex]?.focus()
  } else if (value.length === 1) {
    pinDigits.value[index] = value
    
    // 自动聚焦到下一个输入框
    if (index < 3) {
      pinInputs.value[index + 1]?.focus()
    }
  } else {
    pinDigits.value[index] = ''
  }
}

function handleKeydown(index: number, event: KeyboardEvent) {
  if (event.key === 'Backspace' && !pinDigits.value[index] && index > 0) {
    // 如果当前输入框为空且按下退格键，聚焦到前一个输入框
    pinInputs.value[index - 1]?.focus()
  } else if (event.key === 'ArrowLeft' && index > 0) {
    pinInputs.value[index - 1]?.focus()
  } else if (event.key === 'ArrowRight' && index < 3) {
    pinInputs.value[index + 1]?.focus()
  } else if (event.key === 'Enter' && isComplete.value) {
    handleConfirm()
  }
}

function handlePaste(event: ClipboardEvent) {
  event.preventDefault()
  const pastedData = event.clipboardData?.getData('text') || ''
  const digits = pastedData.replace(/\D/g, '').slice(0, 4).split('')
  
  for (let i = 0; i < Math.min(digits.length, 4); i++) {
    pinDigits.value[i] = digits[i]
  }
  
  // 聚焦到最后一个填充的输入框
  const lastIndex = Math.min(digits.length - 1, 3)
  pinInputs.value[lastIndex]?.focus()
}

async function handleConfirm() {
  if (!isComplete.value || isProcessing.value) return
  
  isProcessing.value = true
  
  try {
    emit('confirm', pinValue.value)
  } catch (error) {
    hasError.value = true
    errorMessage.value = '处理失败，请重试'
    isProcessing.value = false
  }
}

function handleCancel() {
  if (isProcessing.value) return
  emit('cancel')
}

function showError(message: string) {
  hasError.value = true
  errorMessage.value = message
  isProcessing.value = false
  
  // 清空输入并聚焦到第一个输入框
  pinDigits.value = ['', '', '', '']
  nextTick(() => {
    pinInputs.value[0]?.focus()
  })
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

