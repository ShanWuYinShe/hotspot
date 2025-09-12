/**
 * 设备密钥管理服务
 * 提供稳定且安全的设备密钥生成和管理
 */

interface DeviceKeyInfo {
  key: string
  fingerprint: string
  createdAt: string
  version: number
}

export const useDeviceKey = () => {
  const STORAGE_KEY = 'hotspot_device_key'
  const FINGERPRINT_KEY = 'hotspot_device_fingerprint'
  const CURRENT_VERSION = 2 // 版本号，用于密钥格式升级

  // 生成浏览器指纹
  const generateFingerprint = async (): Promise<string> => {
    const components = [
      navigator.userAgent,
      navigator.language,
      navigator.languages?.join(',') || '',
      screen.width,
      screen.height,
      screen.colorDepth,
      new Date().getTimezoneOffset(),
      navigator.hardwareConcurrency || 0,
      (navigator as any).deviceMemory || 0,
      navigator.maxTouchPoints || 0,
    ]

    // 添加一些可选的浏览器特性
    try {
      const canvas = document.createElement('canvas')
      const ctx = canvas.getContext('2d')
      if (ctx) {
        ctx.textBaseline = 'top'
        ctx.font = '14px Arial'
        ctx.fillText('Device fingerprint', 2, 2)
        components.push(canvas.toDataURL())
      }
    } catch {
      // Canvas fingerprinting 可能被阻止，忽略错误
    }

    const fingerprint = components.join('|')
    
    // 使用 SHA-256 生成指纹哈希
    const encoder = new TextEncoder()
    const data = encoder.encode(fingerprint)
    const hashBuffer = await crypto.subtle.digest('SHA-256', data)
    const hashArray = Array.from(new Uint8Array(hashBuffer))
    
    return hashArray.map(b => b.toString(16).padStart(2, '0')).join('')
  }

  // 生成强随机密钥
  const generateSecureKey = async (): Promise<string> => {
    const keyBuffer = crypto.getRandomValues(new Uint8Array(32))
    const keyArray = Array.from(keyBuffer)
    return keyArray.map(b => b.toString(16).padStart(2, '0')).join('')
  }

  // 获取存储的设备密钥信息
  const getStoredKeyInfo = (): DeviceKeyInfo | null => {
    try {
      const stored = localStorage.getItem(STORAGE_KEY)
      return stored ? JSON.parse(stored) : null
    } catch {
      return null
    }
  }

  // 存储设备密钥信息
  const storeKeyInfo = (keyInfo: DeviceKeyInfo): void => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(keyInfo))
      localStorage.setItem(FINGERPRINT_KEY, keyInfo.fingerprint)
    } catch (error) {
      console.error('Failed to store device key:', error)
      throw new Error('无法存储设备密钥')
    }
  }

  // 验证指纹是否匹配
  const verifyFingerprint = async (storedFingerprint: string): Promise<boolean> => {
    try {
      const currentFingerprint = await generateFingerprint()
      return currentFingerprint === storedFingerprint
    } catch {
      return false
    }
  }

  // 获取或生成设备密钥
  const getDeviceKey = async (): Promise<string> => {
    const storedKeyInfo = getStoredKeyInfo()
    
    if (storedKeyInfo) {
      // 检查版本更新
      if (storedKeyInfo.version < CURRENT_VERSION) {
        if (import.meta.dev) {
          console.log('Device key version outdated, regenerating...')
        }
        return await regenerateDeviceKey()
      }
      
      // 验证指纹是否仍然匹配
      const fingerprintValid = await verifyFingerprint(storedKeyInfo.fingerprint)
      
      if (fingerprintValid) {
        return storedKeyInfo.key
      } else {
        if (import.meta.dev) {
          console.log('Device fingerprint changed, regenerating key...')
        }
        return await regenerateDeviceKey()
      }
    }
    
    // 首次生成密钥
    return await generateDeviceKey()
  }

  // 生成新的设备密钥
  const generateDeviceKey = async (): Promise<string> => {
    try {
      const fingerprint = await generateFingerprint()
      const secureKey = await generateSecureKey()
      
      // 结合指纹和随机密钥生成最终密钥
      const combinedData = fingerprint + secureKey
      const encoder = new TextEncoder()
      const data = encoder.encode(combinedData)
      const hashBuffer = await crypto.subtle.digest('SHA-256', data)
      const hashArray = Array.from(new Uint8Array(hashBuffer))
      const finalKey = hashArray.map(b => b.toString(16).padStart(2, '0')).join('')
      
      const keyInfo: DeviceKeyInfo = {
        key: finalKey,
        fingerprint,
        createdAt: new Date().toISOString(),
        version: CURRENT_VERSION,
      }
      
      storeKeyInfo(keyInfo)
      
      if (import.meta.dev) {
        console.log('New device key generated')
      }
      
      return finalKey
    } catch (error) {
      console.error('Failed to generate device key:', error)
      throw new Error('无法生成设备密钥')
    }
  }

  // 重新生成设备密钥
  const regenerateDeviceKey = async (): Promise<string> => {
    // 清除旧的密钥
    clearDeviceKey()
    // 生成新的密钥
    return await generateDeviceKey()
  }

  // 清除设备密钥
  const clearDeviceKey = (): void => {
    try {
      localStorage.removeItem(STORAGE_KEY)
      localStorage.removeItem(FINGERPRINT_KEY)
    } catch (error) {
      console.error('Failed to clear device key:', error)
    }
  }

  // 检查设备密钥是否存在
  const hasDeviceKey = (): boolean => {
    return getStoredKeyInfo() !== null
  }

  // 获取设备密钥信息（不包含实际密钥）
  const getDeviceKeyInfo = (): Omit<DeviceKeyInfo, 'key'> | null => {
    const keyInfo = getStoredKeyInfo()
    if (!keyInfo) return null
    
    return {
      fingerprint: keyInfo.fingerprint,
      createdAt: keyInfo.createdAt,
      version: keyInfo.version,
    }
  }

  // 验证设备密钥的有效性
  const validateDeviceKey = async (): Promise<boolean> => {
    const storedKeyInfo = getStoredKeyInfo()
    if (!storedKeyInfo) return false
    
    // 检查版本
    if (storedKeyInfo.version < CURRENT_VERSION) return false
    
    // 检查指纹
    return await verifyFingerprint(storedKeyInfo.fingerprint)
  }

  // 使用 PBKDF2 增强密钥安全性
  const deriveKeyWithPBKDF2 = async (
    baseKey: string, 
    salt: string, 
    iterations: number = 100000
  ): Promise<string> => {
    const encoder = new TextEncoder()
    const keyMaterial = await crypto.subtle.importKey(
      'raw',
      encoder.encode(baseKey),
      { name: 'PBKDF2' },
      false,
      ['deriveBits']
    )
    
    const derivedBits = await crypto.subtle.deriveBits(
      {
        name: 'PBKDF2',
        salt: encoder.encode(salt),
        iterations,
        hash: 'SHA-256',
      },
      keyMaterial,
      256 // 32 bytes
    )
    
    const derivedArray = Array.from(new Uint8Array(derivedBits))
    return derivedArray.map(b => b.toString(16).padStart(2, '0')).join('')
  }

  // 获取用于加密的强化密钥
  const getEncryptionKey = async (purpose: string = 'default'): Promise<string> => {
    const baseKey = await getDeviceKey()
    const salt = `hotspot_${purpose}_salt_v${CURRENT_VERSION}`
    return await deriveKeyWithPBKDF2(baseKey, salt)
  }

  return {
    // 主要方法
    getDeviceKey,
    generateDeviceKey,
    regenerateDeviceKey,
    clearDeviceKey,
    
    // 查询方法
    hasDeviceKey,
    getDeviceKeyInfo,
    validateDeviceKey,
    
    // 高级方法
    getEncryptionKey,
    deriveKeyWithPBKDF2,
    
    // 工具方法
    generateFingerprint,
    verifyFingerprint,
  }
}


