/**
 * 密码安全工具函数
 * 
 * 密码安全策略：
 * - **不做客户端哈希**：避免把“可重放的哈希值”当作密码替身
 * - **传输安全**：依赖 HTTPS/TLS
 * - **服务端存储**：服务端进行慢哈希（bcrypt/argon2id）
 */

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
  const level = levels[Math.min(score, 4)] || 'weak'

  return { score, level, feedback }
}



