/**
 * 密码安全工具函数
 * 
 * 根据 api_spec.mdc 2.10 密码安全策略：
 * - 客户端哈希：SHA256(password)
 * - 字段命名：password_hash
 * - 算法固定：SHA-256
 * - 传输安全：HTTPS
 */

/**
 * 计算密码哈希
 * 
 * @param password 原始密码
 * @returns SHA256 哈希值（十六进制字符串）
 * 
 * @example
 * const hash = await hashPassword('myPassword123')
 * // 返回: "ef92b778bafe771e89245b89ecbc08a44a4e166c06659911881f383d4473e94f"
 */
export async function hashPassword(password: string): Promise<string> {
  // 使用浏览器原生 Web Crypto API
  const encoder = new TextEncoder()
  const data = encoder.encode(password)
  
  // 计算 SHA-256 哈希
  const hashBuffer = await crypto.subtle.digest('SHA-256', data)
  
  // 转换为十六进制字符串
  const hashArray = Array.from(new Uint8Array(hashBuffer))
  const hashHex = hashArray.map(b => b.toString(16).padStart(2, '0')).join('')
  
  return hashHex
}

/**
 * 验证密码强度
 * 
 * @param password 原始密码
 * @returns 强度等级和提示信息
 */
export interface PasswordStrength {
  score: number // 0-4 (弱-强)
  level: 'weak' | 'fair' | 'good' | 'strong' | 'very_strong'
  feedback: string[]
}

export function checkPasswordStrength(password: string): PasswordStrength {
  const feedback: string[] = []
  let score = 0

  // 长度检查
  if (password.length >= 8) score++
  if (password.length >= 12) score++
  if (password.length < 6) {
    feedback.push('密码至少需要6个字符')
  }

  // 复杂度检查
  if (/[a-z]/.test(password) && /[A-Z]/.test(password)) {
    score++
    feedback.push('✓ 包含大小写字母')
  } else {
    feedback.push('建议使用大小写字母组合')
  }

  if (/\d/.test(password)) {
    score++
    feedback.push('✓ 包含数字')
  } else {
    feedback.push('建议包含数字')
  }

  if (/[^a-zA-Z0-9]/.test(password)) {
    score++
    feedback.push('✓ 包含特殊字符')
  } else {
    feedback.push('建议包含特殊字符')
  }

  // 确定强度等级
  const levels: PasswordStrength['level'][] = ['weak', 'fair', 'good', 'strong', 'very_strong']
  const level = levels[Math.min(score, 4)]

  return { score, level, feedback }
}

