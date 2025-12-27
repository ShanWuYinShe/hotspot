<template>
  <UModal
    :title="isSetup ? '设置设备PIN' : '输入设备PIN'"
    :dismissible="false"
  >
    <template #body>
      <UAlert
        class="mb-4"
        color="neutral"
        variant="soft"
        icon="i-lucide-lock-keyhole"
        :title="isSetup
          ? '请设置4位数字PIN以保护账户切换功能'
          : '请输入4位数字PIN以解锁账户切换功能'
        "
      />

      <section class="flex justify-center mb-4">
        <UPinInput
          v-model="pinValue"
          :length="4"
          :mask="true"
          :error="hasError"
          placeholder="●"
          @complete="handlePinComplete"
        />
      </section>

      <UAlert
        v-if="hasError"
        color="error"
        variant="subtle"
        :title="errorMessage"
        class="mb-4"
      />

      <p
        v-if="isSetup"
        class="text-xs text-center"
      >
        <UBadge color="neutral" variant="soft" size="xs">
          建议使用好记的数字组合，如生日或纪念日
        </UBadge>
      </p>
    </template>

    <template #footer>
      <UButton
        variant="ghost"
        :disabled="isProcessing"
        @click="handleCancel"
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
  (e: 'close', value?: string): void
}

const props = withDefaults(defineProps<Props>(), {
  isSetup: false
})

const emit = defineEmits<Emits>()

const pinValue = ref<string[]>(['', '', '', ''])
const hasError = ref(false)
const errorMessage = ref('')
const isProcessing = ref(false)

const isComplete = computed(() =>
  pinValue.value.length === 4 && pinValue.value.every(digit => /^\d$/.test(digit))
)

const pinString = computed(() => pinValue.value.join(''))

onMounted(() => resetForm())

watch(pinValue, () => {
  if (hasError.value) {
    hasError.value = false
    errorMessage.value = ''
  }
})

const handlePinComplete = (pins: string[]) => {
  pinValue.value = pins
}

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
    emit('close', pinString.value)
  } catch (error) {
    hasError.value = true
    errorMessage.value = '处理失败，请重试'
    isProcessing.value = false
  }
}

function handleCancel() {
  if (isProcessing.value) return
  emit('cancel')
  emit('close')
}

function showError(message: string) {
  hasError.value = true
  errorMessage.value = message
  isProcessing.value = false
  resetForm()
}

defineExpose({ showError, resetForm })
</script>
