/**
 * 轻量表单验证工具
 *
 * 用法：
 * 1. 定义字段规则数组
 * 2. 用 $derived 派生错误: `let err = $derived(touched['field'] ? validateField(value, rules) : undefined)`
 * 3. onblur 标记 touched，模板显示内联错误
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
 * 按优先级依次校验，返回首个错误消息；全部通过返回 undefined
 */
export function validateField(value: string, rules: FieldRule[]): string | undefined {
	for (const rule of rules) {
		if (rule.required && !value.trim()) {
			return rule.requiredMessage ?? '此字段不能为空';
		}
		if (rule.minLength !== undefined && value.length > 0 && value.length < rule.minLength) {
			return rule.minLengthMessage ?? `最少 ${rule.minLength} 字符`;
		}
		if (rule.maxLength !== undefined && value.length > rule.maxLength) {
			return rule.maxLengthMessage ?? `最多 ${rule.maxLength} 字符`;
		}
		if (rule.pattern && value && !rule.pattern.test(value)) {
			return rule.patternMessage ?? '格式不正确';
		}
		if (rule.custom) {
			const error = rule.custom(value);
			if (error) return error;
		}
	}
	return undefined;
}
