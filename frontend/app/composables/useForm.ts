/**
 * 通用表单处理 - 统一表单状态和提交逻辑
 */

interface FormOptions<T> {
  initialValues?: Partial<T>
  validate?: (values: T) => Record<string, string> | null
  onSuccess?: (result: any) => void
  onError?: (error: any) => void
}

export function useForm<T extends Record<string, any>>(
  submitFunction: (values: T) => Promise<any>,
  options: FormOptions<T> = {}
) {
  const { initialValues = {}, validate, onSuccess, onError } = options

  // 状态管理
  const values = reactive<T>({ ...initialValues } as T)
  const errors = ref<Record<string, string>>({})
  const submitting = ref(false)
  const submitted = ref(false)

  const toast = useSmartToast()

  // 设置字段值
  const setValue = (field: keyof T, value: any) => {
    values[field] = value
    // 清除该字段的错误
    if (errors.value[field as string]) {
      delete errors.value[field as string]
    }
  }

  // 验证表单
  const validateForm = () => {
    if (!validate) return true
    
    const validationErrors = validate(values)
    if (validationErrors) {
      errors.value = validationErrors
      return false
    }
    
    errors.value = {}
    return true
  }

  // 提交表单
  const submit = async () => {
    submitted.value = true
    
    if (!validateForm()) {
      toast.error('请检查表单输入')
      return false
    }

    submitting.value = true
    try {
      const result = await submitFunction(values)
      onSuccess?.(result)
      toast.success('操作成功')
      return result
    } catch (error: any) {
      const errorMessage = error.message || '操作失败'
      onError?.(error)
      toast.error(errorMessage)
      return false
    } finally {
      submitting.value = false
    }
  }

  // 重置表单
  const reset = () => {
    Object.assign(values, initialValues)
    errors.value = {}
    submitted.value = false
  }

  // 计算属性
  const isValid = computed(() => Object.keys(errors.value).length === 0)
  const hasErrors = computed(() => Object.keys(errors.value).length > 0)

  return {
    values,
    errors: readonly(errors),
    submitting: readonly(submitting),
    submitted: readonly(submitted),
    isValid,
    hasErrors,
    setValue,
    submit,
    reset,
    validateForm
  }
}
