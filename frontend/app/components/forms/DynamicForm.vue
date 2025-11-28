<template>
  <UForm
    :schema="schema"
    :state="formState"
    :validate-on="props.validateOn"
    class="space-y-4"
    @submit="handleSubmit"
    @error="handleError"
  >
    <div v-for="field in fields" :key="field.name" class="space-y-2">
      <!-- 字段标签 -->
      <UFormField
        :label="field.label"
        :description="field.description"
        :required="field.required"
        :error="getFieldError(field.name)"
        :name="field.name"
      >
        <!-- 文本输入 -->
        <UInput
          v-if="field.type === 'text' || field.type === 'email' || field.type === 'password'"
          v-model="formState[field.name]"
          :type="field.type"
          :placeholder="field.placeholder"
          :disabled="field.disabled || loading"
          :icon="field.icon"
        />
        
        <!-- 文本域 -->
        <UTextarea
          v-else-if="field.type === 'textarea'"
          v-model="formState[field.name]"
          :placeholder="field.placeholder"
          :disabled="field.disabled || loading"
          :rows="field.rows || 3"
        />
        
        <!-- 选择器 -->
        <USelectMenu
          v-else-if="field.type === 'select'"
          v-model="formState[field.name]"
          :options="field.options"
          :placeholder="field.placeholder"
          :disabled="field.disabled || loading"
          :multiple="field.multiple"
        />
        
        <!-- 自动完成输入 -->
        <UInputMenu
          v-else-if="field.type === 'autocomplete'"
          v-model="formState[field.name]"
          :items="field.options"
          :placeholder="field.placeholder"
          :disabled="field.disabled || loading"
          :multiple="field.multiple"
        />
        
        <!-- 标签输入 -->
        <UInputTags
          v-else-if="field.type === 'tags'"
          v-model="formState[field.name]"
          :placeholder="field.placeholder"
          :disabled="field.disabled || loading"
          :max="field.max"
        />
        
        <!-- PIN 码输入 -->
        <UPinInput
          v-else-if="field.type === 'pin'"
          v-model="formState[field.name]"
          :length="field.length || 6"
          :type="field.pinType || 'text'"
          :mask="field.mask || false"
          :otp="field.otp || false"
          :disabled="field.disabled || loading"
        />
        
        <!-- 开关 -->
        <USwitch
          v-else-if="field.type === 'switch'"
          v-model="formState[field.name]"
          :disabled="field.disabled || loading"
        />
        
        <!-- 复选框 -->
        <UCheckbox
          v-else-if="field.type === 'checkbox'"
          v-model="formState[field.name]"
          :label="field.label"
          :disabled="field.disabled || loading"
        />
        
        <!-- 文件上传 -->
        <UFileUpload
          v-else-if="field.type === 'file'"
          v-model="formState[field.name]"
          :accept="field.accept"
          :multiple="field.multiple"
          :disabled="field.disabled || loading"
          :description="field.description"
        />
        
        <!-- 日期选择器 -->
        <UInput
          v-else-if="field.type === 'date'"
          v-model="formState[field.name]"
          type="date"
          :placeholder="field.placeholder"
          :disabled="field.disabled || loading"
        />
        
        <!-- 数字输入 -->
        <UInputNumber
          v-else-if="field.type === 'number'"
          v-model="formState[field.name]"
          :placeholder="field.placeholder"
          :disabled="field.disabled || loading"
          :min="field.min"
          :max="field.max"
          :step="field.step"
        />
        
        <!-- 滑块输入 -->
        <div v-else-if="field.type === 'slider'" class="space-y-2">
          <USlider
            v-model="formState[field.name]"
            :min="field.min || 0"
            :max="field.max || 100"
            :step="field.step || 1"
            :disabled="field.disabled || loading"
            :tooltip="field.tooltip !== false"
          />
          <div class="flex justify-between text-xs text-neutral-500 dark:text-neutral-400">
            <span>{{ field.min || 0 }}{{ field.unit || '' }}</span>
            <span class="font-medium">{{ formState[field.name] }}{{ field.unit || '' }}</span>
            <span>{{ field.max || 100 }}{{ field.unit || '' }}</span>
          </div>
        </div>
      </UFormField>
    </div>
    
    <!-- 表单操作按钮 -->
    <div class="flex items-center justify-end gap-3 pt-4">
      <UButton
        v-if="showCancel"
        variant="ghost"
        :disabled="loading"
        @click="handleCancel"
      >
        取消
      </UButton>
      <UButton
        type="submit"
        :loading="loading"
        :disabled="!isFormValid"
      >
        {{ submitLabel }}
      </UButton>
    </div>
  </UForm>
</template>

<script setup lang="ts">
import { z } from 'zod'
// 注释掉直到安装 @internationalized/date 包
// import { CalendarDate, DateFormatter, getLocalTimeZone, parseDate as parseISODate } from '@internationalized/date'

export interface FormField {
  name: string
  label: string
  type: 'text' | 'email' | 'password' | 'textarea' | 'select' | 'autocomplete' | 'tags' | 'pin' | 'switch' | 'checkbox' | 'file' | 'date' | 'number' | 'slider'
  placeholder?: string
  description?: string
  required?: boolean
  disabled?: boolean
  icon?: string
  
  // 文本域特定
  rows?: number
  
  // 选择器特定
  options?: Array<{ label: string; value: any }>
  multiple?: boolean
  
  // 文件上传特定
  accept?: string
  
  // 数字输入特定
  min?: number
  max?: number
  step?: number
  
  // 滑块特定
  tooltip?: boolean
  unit?: string
  
  // PIN 码输入特定
  length?: number
  pinType?: 'text' | 'number'
  mask?: boolean
  otp?: boolean
}

interface Props {
  fields: FormField[]
  initialData?: Record<string, any>
  schema?: z.ZodSchema
  loading?: boolean
  submitLabel?: string
  showCancel?: boolean
  validateOn?: Array<'blur' | 'change' | 'input'>
}

const props = withDefaults(defineProps<Props>(), {
  initialData: () => ({}),
  loading: false,
  submitLabel: '提交',
  showCancel: true,
  validateOn: () => ['blur', 'change', 'input']
})

const emit = defineEmits<{
  submit: [data: Record<string, any>]
  cancel: []
  change: [field: string, value: any]
}>()

// 表单状态
const formState = reactive({ ...props.initialData })
const formErrors = ref<Record<string, string>>({})

// 计算属性
const isFormValid = computed(() => {
  if (props.schema) {
    try {
      props.schema.parse(formState)
      return true
    } catch {
      return false
    }
  }
  
  // 基本验证：检查必填字段
  return props.fields.every(field => {
    if (!field.required) return true
    const value = formState[field.name]
    return value !== undefined && value !== null && value !== ''
  })
})

// 方法
const getFieldError = (fieldName: string) => {
  return formErrors.value[fieldName]
}


const handleSubmit = async () => {
  try {
    formErrors.value = {}
    
    if (props.schema) {
      const validatedData = props.schema.parse(formState)
      emit('submit', validatedData)
    } else {
      emit('submit', { ...formState })
    }
  } catch (error) {
    if (error instanceof z.ZodError) {
      error.errors.forEach(err => {
        const path = err.path.join('.')
        formErrors.value[path] = err.message
      })
    } else {
      // 处理其他类型的错误
      console.error('表单提交错误:', error)
    }
  }
}

const handleError = (event: any) => {
  console.error('表单验证错误:', event)
  // 可以在这里添加更多的错误处理逻辑
  if (event?.errors) {
    event.errors.forEach((err: any) => {
      if (err.path) {
        formErrors.value[err.path] = err.message
      }
    })
  }
}

const handleCancel = () => {
  emit('cancel')
}

// 简化版本，使用原生 HTML date input

// 监听字段变化
watch(formState, (newState, oldState) => {
  Object.keys(newState).forEach(key => {
    if (newState[key] !== oldState?.[key]) {
      emit('change', key, newState[key])
    }
  })
}, { deep: true })

// 初始化表单数据
watchEffect(() => {
  Object.assign(formState, props.initialData)
})
</script>
