/**
 * 通用表单验证 Composable
 * 提供常用的表单验证函数和类型定义
 */

export type FormErrors = { path: string; message: string }[]

/**
 * 验证必填字段
 * @param value 字段值
 * @param label 字段标签
 * @returns 错误信息或 undefined
 */
export function validateRequired(value: string, label: string): string | undefined {
  return (value || "").trim() ? undefined : `${label}为必填项`
}

/**
 * 验证邮箱格式
 * @param value 邮箱值
 * @returns 错误信息或 undefined
 */
export function validateEmail(value: string): string | undefined {
  if (!value?.trim()) return undefined
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
  return emailRegex.test(value) ? undefined : "邮箱格式不正确"
}

/**
 * 验证手机号格式
 * @param value 手机号值
 * @returns 错误信息或 undefined
 */
export function validatePhone(value: string): string | undefined {
  if (!value?.trim()) return undefined
  const phoneRegex = /^1[3-9]\d{9}$/
  return phoneRegex.test(value) ? undefined : "手机号格式不正确"
}

// validatePassword 已移至 ~/utils/api-validation.ts 统一管理

/**
 * 验证字符串长度
 * @param value 字符串值
 * @param maxLength 最大长度
 * @param fieldName 字段名称
 * @returns 错误信息或 undefined
 */
export function validateLength(value: string, maxLength: number, fieldName: string = "字段"): string | undefined {
  if (!value?.trim()) return undefined
  if (value.length > maxLength) return `${fieldName}不能超过${maxLength}个字符`
  return undefined
}

/**
 * 创建表单验证器
 * @param validationRules 验证规则对象
 * @returns 验证函数
 */
export function createFormValidator(validationRules: Record<string, Array<(value: any) => string | undefined>>) {
  return (state: any): FormErrors => {
    const errors: FormErrors = []
    
    for (const [field, rules] of Object.entries(validationRules)) {
      for (const rule of rules) {
        const error = rule(state[field])
        if (error) {
          errors.push({ path: field, message: error })
          break // 只显示第一个错误
        }
      }
    }
    
    return errors
  }
}

/**
 * 通用表单验证 composable
 */
export function useFormValidation() {
  return {
    validateRequired,
    validateEmail,
    validatePhone,
    validateLength,
    createFormValidator
  }
}
