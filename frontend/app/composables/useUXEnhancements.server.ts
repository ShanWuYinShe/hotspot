/**
 * UX 增强功能 Composable - 服务端版本
 * 提供服务端安全的占位实现
 */

import { ref, computed } from 'vue'

export const useUXEnhancements = () => {
  /**
   * 防抖函数 - 服务端占位
   */
  const debounce = <T extends (...args: any[]) => any>(
    func: T,
    delay: number
  ): ((...args: Parameters<T>) => void) => {
    return (...args: Parameters<T>) => {
      // 服务端直接执行，不防抖
      func.apply(null, args)
    }
  }

  /**
   * 节流函数 - 服务端占位
   */
  const throttle = <T extends (...args: any[]) => any>(
    func: T,
    delay: number
  ): ((...args: Parameters<T>) => void) => {
    return (...args: Parameters<T>) => {
      // 服务端直接执行，不节流
      func.apply(null, args)
    }
  }

  /**
   * 复制文本到剪贴板 - 服务端占位
   */
  const copyToClipboard = async (text: string): Promise<boolean> => {
    console.warn('copyToClipboard is not available on server side')
    return false
  }

  /**
   * 平滑滚动到指定元素 - 服务端占位
   */
  const scrollToElement = (elementId: string, offset = 0) => {
    console.warn('scrollToElement is not available on server side')
  }

  /**
   * 格式化文件大小
   */
  const formatFileSize = (bytes: number, decimals = 2): string => {
    if (bytes === 0) return '0 Bytes'

    const k = 1024
    const dm = decimals < 0 ? 0 : decimals
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB']

    const i = Math.floor(Math.log(bytes) / Math.log(k))

    return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i]
  }

  /**
   * 格式化数字（添加千分位分隔符）
   */
  const formatNumber = (num: number): string => {
    return num.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ',')
  }

  /**
   * 截断文本
   */
  const truncateText = (text: string, maxLength: number, suffix = '...'): string => {
    if (text.length <= maxLength) return text
    return text.slice(0, maxLength - suffix.length) + suffix
  }

  /**
   * 检查是否为触摸设备 - 服务端默认值
   */
  const isTouchDevice = computed(() => false)

  /**
   * 获取浏览器信息 - 服务端占位
   */
  const getBrowserInfo = () => {
    return {
      isChrome: false,
      isFirefox: false,
      isSafari: false,
      isEdge: false,
      userAgent: 'server'
    }
  }

  /**
   * 延迟执行
   */
  const delay = (ms: number): Promise<void> => {
    return new Promise(resolve => setTimeout(resolve, ms))
  }

  /**
   * 等待元素出现在DOM中 - 服务端占位
   */
  const waitForElement = (selector: string, timeout = 5000): Promise<Element | null> => {
    console.warn('waitForElement is not available on server side')
    return Promise.resolve(null)
  }

  /**
   * 创建可取消的Promise
   */
  const makeCancelable = <T>(promise: Promise<T>) => {
    return {
      promise,
      cancel: () => {
        console.warn('makeCancelable.cancel is not available on server side')
      }
    }
  }

  return {
    // 工具函数
    debounce,
    throttle,
    copyToClipboard,
    scrollToElement,
    formatFileSize,
    formatNumber,
    truncateText,
    delay,
    waitForElement,
    makeCancelable,
    
    // 设备检测
    isTouchDevice,
    getBrowserInfo
  }
}

