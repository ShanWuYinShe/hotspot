/**
 * PIN 安全管理服务
 * 使用更强的哈希算法和安全实践
 */

interface PinInfo {
  hash: string
  salt: string
  iterations: number
  createdAt: string
  version: number
  attempts: number
  lastAttempt?: string
  lockedUntil?: string
}

export const usePinSecurity = () => {
  const STORAGE_KEY = 'hotspot_pin_info'
  const CURRENT_VERSION = 1
  const DEFAULT_ITERATIONS = 100000 // PBKDF2 迭代次数
  const MAX_ATTEMPTS = 5 // 最大尝试次数
  const LOCKOUT_DURATION = 15 * 60 * 1000 // 15分钟锁定时间

  // 生成随机盐
  const generateSalt = (): string => {
    const saltBuffer = crypto.getRandomValues(new Uint8Array(16))
    const saltArray = Array.from(saltBuffer)
    return saltArray.map(b => b.toString(16).padStart(2, '0')).join('')
  }

  // 使用 PBKDF2 哈希 PIN
  const hashPinWithPBKDF2 = async (
    pin: string, 
    salt: string, 
    iterations: number = DEFAULT_ITERATIONS
  ): Promise<string> => {
    const encoder = new TextEncoder()
    
    // 导入 PIN 作为密钥材料
    const keyMaterial = await crypto.subtle.importKey(
      'raw',
      encoder.encode(pin),
      { name: 'PBKDF2' },
      false,
      ['deriveBits']
    )
    
    // 使用 PBKDF2 派生密钥
    const derivedBits = await crypto.subtle.deriveBits(
      {
        name: 'PBKDF2',
        salt: encoder.encode(salt + 'hotspot_pin_v1'), // 添加应用特定的盐
        iterations,
        hash: 'SHA-256',
      },
      keyMaterial,
      256 // 32 bytes
    )
    
    const hashArray = Array.from(new Uint8Array(derivedBits))
    return hashArray.map(b => b.toString(16).padStart(2, '0')).join('')
  }

  // 获取存储的 PIN 信息
  const getStoredPinInfo = (): PinInfo | null => {
    try {
      const stored = localStorage.getItem(STORAGE_KEY)
      return stored ? JSON.parse(stored) : null
    } catch {
      return null
    }
  }

  // 存储 PIN 信息
  const storePinInfo = (pinInfo: PinInfo): void => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(pinInfo))
    } catch (error) {
      console.error('Failed to store PIN info:', error)
      throw new Error('无法存储PIN信息')
    }
  }

  // 检查是否被锁定
  const isLocked = (): boolean => {
    const pinInfo = getStoredPinInfo()
    if (!pinInfo || !pinInfo.lockedUntil) return false
    
    const now = new Date().getTime()
    const lockUntil = new Date(pinInfo.lockedUntil).getTime()
    
    return now < lockUntil
  }

  // 获取剩余锁定时间（毫秒）
  const getRemainingLockTime = (): number => {
    const pinInfo = getStoredPinInfo()
    if (!pinInfo || !pinInfo.lockedUntil) return 0
    
    const now = new Date().getTime()
    const lockUntil = new Date(pinInfo.lockedUntil).getTime()
    
    return Math.max(0, lockUntil - now)
  }

  // 设置 PIN
  const setPin = async (pin: string): Promise<void> => {
    // 验证 PIN 格式
    if (!/^\d{4}$/.test(pin)) {
      throw new Error('PIN必须是4位数字')
    }

    // 检查 PIN 强度（避免过于简单的组合）
    const weakPatterns = [
      /^(\d)\1{3}$/, // 四个相同数字 (1111, 2222, etc.)
      /^1234$/, /^4321$/, // 连续数字
      /^0000$/, /^1111$/, /^2222$/, /^3333$/, /^4444$/, 
      /^5555$/, /^6666$/, /^7777$/, /^8888$/, /^9999$/, // 重复数字
    ]
    
    const isWeak = weakPatterns.some(pattern => pattern.test(pin))
    if (isWeak) {
      throw new Error('PIN过于简单，请选择更安全的组合')
    }

    try {
      const salt = generateSalt()
      const hash = await hashPinWithPBKDF2(pin, salt)
      
      const pinInfo: PinInfo = {
        hash,
        salt,
        iterations: DEFAULT_ITERATIONS,
        createdAt: new Date().toISOString(),
        version: CURRENT_VERSION,
        attempts: 0,
      }
      
      storePinInfo(pinInfo)
      
      if (import.meta.dev) {
        console.log('PIN set successfully')
      }
    } catch (error) {
      console.error('Failed to set PIN:', error)
      throw new Error('设置PIN失败')
    }
  }

  // 验证 PIN
  const verifyPin = async (pin: string): Promise<boolean> => {
    const pinInfo = getStoredPinInfo()
    if (!pinInfo) {
      throw new Error('未设置PIN')
    }

    // 检查是否被锁定
    if (isLocked()) {
      const remainingTime = Math.ceil(getRemainingLockTime() / 1000 / 60)
      throw new Error(`PIN已锁定，请等待${remainingTime}分钟后重试`)
    }

    // 验证 PIN 格式
    if (!/^\d{4}$/.test(pin)) {
      throw new Error('PIN必须是4位数字')
    }

    try {
      const hash = await hashPinWithPBKDF2(pin, pinInfo.salt, pinInfo.iterations)
      const isValid = hash === pinInfo.hash
      
      if (isValid) {
        // 验证成功，重置尝试次数
        pinInfo.attempts = 0
        delete pinInfo.lastAttempt
        delete pinInfo.lockedUntil
        storePinInfo(pinInfo)
        
        if (import.meta.dev) {
          console.log('PIN verification successful')
        }
        
        return true
      } else {
        // 验证失败，增加尝试次数
        pinInfo.attempts += 1
        pinInfo.lastAttempt = new Date().toISOString()
        
        if (pinInfo.attempts >= MAX_ATTEMPTS) {
          // 达到最大尝试次数，锁定账户
          pinInfo.lockedUntil = new Date(Date.now() + LOCKOUT_DURATION).toISOString()
          storePinInfo(pinInfo)
          
          const lockMinutes = LOCKOUT_DURATION / 1000 / 60
          throw new Error(`PIN错误次数过多，已锁定${lockMinutes}分钟`)
        } else {
          storePinInfo(pinInfo)
          const remainingAttempts = MAX_ATTEMPTS - pinInfo.attempts
          throw new Error(`PIN错误，还有${remainingAttempts}次尝试机会`)
        }
      }
    } catch (error) {
      if (error instanceof Error && error.message.includes('PIN')) {
        throw error // 重新抛出 PIN 相关的错误
      }
      
      console.error('PIN verification failed:', error)
      throw new Error('PIN验证失败')
    }
  }

  // 更改 PIN
  const changePin = async (oldPin: string, newPin: string): Promise<void> => {
    // 先验证旧 PIN
    const isOldPinValid = await verifyPin(oldPin)
    if (!isOldPinValid) {
      throw new Error('原PIN错误')
    }
    
    // 设置新 PIN
    await setPin(newPin)
  }

  // 检查是否已设置 PIN
  const hasPin = (): boolean => {
    return getStoredPinInfo() !== null
  }

  // 清除 PIN
  const clearPin = (): void => {
    try {
      localStorage.removeItem(STORAGE_KEY)
    } catch (error) {
      console.error('Failed to clear PIN:', error)
    }
  }

  // 获取 PIN 状态信息（不包含敏感数据）
  const getPinStatus = () => {
    const pinInfo = getStoredPinInfo()
    if (!pinInfo) {
      return {
        hasPin: false,
        isLocked: false,
        attempts: 0,
        remainingLockTime: 0,
      }
    }
    
    return {
      hasPin: true,
      isLocked: isLocked(),
      attempts: pinInfo.attempts,
      remainingLockTime: getRemainingLockTime(),
      createdAt: pinInfo.createdAt,
      lastAttempt: pinInfo.lastAttempt,
    }
  }

  // 重置尝试次数（管理员功能）
  const resetAttempts = (): void => {
    const pinInfo = getStoredPinInfo()
    if (pinInfo) {
      pinInfo.attempts = 0
      delete pinInfo.lastAttempt
      delete pinInfo.lockedUntil
      storePinInfo(pinInfo)
    }
  }

  // 生成 PIN 强度评分
  const evaluatePinStrength = (pin: string): {
    score: number
    feedback: string[]
  } => {
    const feedback: string[] = []
    let score = 0
    
    if (!/^\d{4}$/.test(pin)) {
      feedback.push('PIN必须是4位数字')
      return { score: 0, feedback }
    }
    
    // 检查重复数字
    const hasRepeating = /^(\d)\1{3}$/.test(pin)
    if (hasRepeating) {
      feedback.push('避免使用相同数字')
    } else {
      score += 25
    }
    
    // 检查连续数字
    const isSequential = /^(0123|1234|2345|3456|4567|5678|6789|9876|8765|7654|6543|5432|4321|3210)$/.test(pin)
    if (isSequential) {
      feedback.push('避免使用连续数字')
    } else {
      score += 25
    }
    
    // 检查常见模式
    const commonPatterns = ['0000', '1111', '1234', '4321', '1122', '2211']
    const isCommon = commonPatterns.includes(pin)
    if (isCommon) {
      feedback.push('避免使用常见组合')
    } else {
      score += 25
    }
    
    // 检查数字多样性
    const uniqueDigits = new Set(pin.split('')).size
    if (uniqueDigits >= 3) {
      score += 25
      feedback.push('数字多样性良好')
    } else if (uniqueDigits === 2) {
      feedback.push('建议使用更多不同数字')
    }
    
    if (score >= 75) {
      feedback.unshift('PIN强度：强')
    } else if (score >= 50) {
      feedback.unshift('PIN强度：中等')
    } else {
      feedback.unshift('PIN强度：弱')
    }
    
    return { score, feedback }
  }

  return {
    // 主要方法
    setPin,
    verifyPin,
    changePin,
    clearPin,
    
    // 查询方法
    hasPin,
    getPinStatus,
    isLocked,
    getRemainingLockTime,
    
    // 工具方法
    evaluatePinStrength,
    resetAttempts,
    
    // 常量
    MAX_ATTEMPTS,
    LOCKOUT_DURATION,
  }
}

