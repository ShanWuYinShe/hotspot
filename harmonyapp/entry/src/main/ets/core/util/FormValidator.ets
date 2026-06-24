/**
 * 轻量表单验证工具（对齐前端 frontend/src/lib/utils/form-validation.ts）。
 *
 * 用法：
 * 1. 页面内定义字段规则数组（文案逐字对齐前端 auth-form-content / create-group-modal 等）。
 * 2. 维护 @State touched: Record<string, boolean>，onBlur 时整体替换 `this.touched = { ...this.touched, [field]: true }` 触发重渲染。
 * 3. 用 getter 方法在 build 时计算错误：`getXxxError(): string | undefined { return this.touched['xxx'] ? validateField(this.xxx, XXX_RULES) : undefined }`。
 * 4. 模板内 `if (error) { Text(error)... }` 渲染字段下方红字。
 */

export interface FieldRule {
  required?: boolean;
  requiredMessage?: string;
  minLength?: number;
  minLengthMessage?: string;
  maxLength?: number;
  maxLengthMessage?: string;
  pattern?: RegExp;
  patternMessage?: string;
  custom?: (value: string) => string | undefined;
}

/**
 * 按优先级依次校验（required → minLength → maxLength → pattern → custom），返回首个错误消息；
 * 全部通过返回 undefined。
 */
export function validateField(value: string, rules: FieldRule[]): string | undefined {
  for (let i = 0; i < rules.length; i++) {
    const rule: FieldRule = rules[i];
    if (rule.required && !value.trim()) {
      return rule.requiredMessage ?? '此字段不能为空';
    }
    const minLen: number | undefined = rule.minLength;
    if (minLen !== undefined && value.length > 0 && value.length < minLen) {
      return rule.minLengthMessage ?? `最少 ${minLen} 字符`;
    }
    const maxLen: number | undefined = rule.maxLength;
    if (maxLen !== undefined && value.length > maxLen) {
      return rule.maxLengthMessage ?? `最多 ${maxLen} 字符`;
    }
    const pattern: RegExp | undefined = rule.pattern;
    if (pattern !== undefined && value.length > 0 && !pattern.test(value)) {
      return rule.patternMessage ?? '格式不正确';
    }
    const customFn: ((value: string) => string | undefined) | undefined = rule.custom;
    if (customFn !== undefined) {
      const error: string | undefined = customFn(value);
      if (error) {
        return error;
      }
    }
  }
  return undefined;
}
