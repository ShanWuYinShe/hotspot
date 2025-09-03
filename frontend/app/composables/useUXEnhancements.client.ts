/**
 * UX 增强功能 Composable - 客户端版本
 * 提供常用的用户体验增强功能，包括防抖、节流、复制到剪贴板、设备检测等
 */

import { ref, computed, nextTick } from 'vue'

export const useUXEnhancements = () => {
  /**
   * 防抖函数
   * @param func 要防抖的函数
   * @param delay 延迟时间（毫秒）
   * @returns 防抖后的函数
   */
  const debounce = <T extends (...args: any[]) => any>(
    func: T,
    delay: number
  ): ((...args: Parameters<T>) => void) => {
    let timeoutId: ReturnType<typeof setTimeout>
    return (...args: Parameters<T>) => {
      clearTimeout(timeoutId)
      timeoutId = setTimeout(() => func.apply(null, args), delay)
    }
  }

  /**
   * 节流函数
   * @param func 要节流的函数
   * @param delay 延迟时间（毫秒）
   * @returns 节流后的函数
   */
  const throttle = <T extends (...args: any[]) => any>(
    func: T,
    delay: number
  ): ((...args: Parameters<T>) => void) => {
    let lastCall = 0
    return (...args: Parameters<T>) => {
      const now = Date.now()
      if (now - lastCall >= delay) {
        lastCall = now
        func.apply(null, args)
      }
    }
  }

  /**
   * 复制文本到剪贴板
   * @param text 要复制的文本
   * @returns Promise<boolean> 是否复制成功
   */
  const copyToClipboard = async (text: string): Promise<boolean> => {
    try {
      if (navigator.clipboard && window.isSecureContext) {
        await navigator.clipboard.writeText(text)
        return true
      } else {
        // 降级方案
        const textArea = document.createElement('textarea')
        textArea.value = text
        textArea.style.position = 'fixed'
        textArea.style.left = '-999999px'
        textArea.style.top = '-999999px'
        document.body.appendChild(textArea)
        textArea.focus()
        textArea.select()
        const success = document.execCommand('copy')
        textArea.remove()
        return success
      }
    } catch (error) {
      console.error('Failed to copy text:', error)
      return false
    }
  }

  /**
   * 平滑滚动到指定元素
   * @param elementId 元素ID
   * @param offset 偏移量（像素）
   */
  const scrollToElement = (elementId: string, offset = 0) => {
    const element = document.getElementById(elementId)
    if (element) {
      const elementPosition = element.getBoundingClientRect().top + window.pageYOffset
      const offsetPosition = elementPosition - offset

      window.scrollTo({
        top: offsetPosition,
        behavior: 'smooth'
      })
    }
  }

  /**
   * 格式化文件大小
   * @param bytes 字节数
   * @param decimals 小数位数
   * @returns 格式化后的文件大小字符串
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
   * @param num 数字
   * @returns 格式化后的数字字符串
   */
  const formatNumber = (num: number): string => {
    return num.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ',')
  }

  /**
   * 截断文本
   * @param text 原始文本
   * @param maxLength 最大长度
   * @param suffix 后缀
   * @returns 截断后的文本
   */
  const truncateText = (text: string, maxLength: number, suffix = '...'): string => {
    if (text.length <= maxLength) return text
    return text.slice(0, maxLength - suffix.length) + suffix
  }

  /**
   * 检查是否为触摸设备
   */
  const isTouchDevice = computed(() => {
    return 'ontouchstart' in window || navigator.maxTouchPoints > 0
  })

  /**
   * 获取浏览器信息
   */
  const getBrowserInfo = () => {
    const userAgent = navigator.userAgent
    const isChrome = /Chrome/.test(userAgent) && /Google Inc/.test(navigator.vendor)
    const isFirefox = /Firefox/.test(userAgent)
    const isSafari = /Safari/.test(userAgent) && /Apple Computer/.test(navigator.vendor)
    const isEdge = /Edg/.test(userAgent)

    return {
      isChrome,
      isFirefox,
      isSafari,
      isEdge,
      userAgent
    }
  }

  /**
   * 延迟执行
   * @param ms 延迟时间（毫秒）
   * @returns Promise
   */
  const delay = (ms: number): Promise<void> => {
    return new Promise(resolve => setTimeout(resolve, ms))
  }

  /**
   * 等待元素出现在DOM中
   * @param selector CSS选择器
   * @param timeout 超时时间（毫秒）
   * @returns Promise<Element | null>
   */
  const waitForElement = (selector: string, timeout = 5000): Promise<Element | null> => {
    return new Promise((resolve) => {
      const element = document.querySelector(selector)
      if (element) {
        resolve(element)
        return
      }

      const observer = new MutationObserver(() => {
        const element = document.querySelector(selector)
        if (element) {
          observer.disconnect()
          resolve(element)
        }
      })

      observer.observe(document.body, {
        childList: true,
        subtree: true
      })

      setTimeout(() => {
        observer.disconnect()
        resolve(null)
      }, timeout)
    })
  }

  /**
   * 创建可取消的Promise
   * @param promise 原始Promise
   * @returns 包含cancel方法的Promise
   */
  const makeCancelable = <T>(promise: Promise<T>) => {
    let isCanceled = false

    const wrappedPromise = new Promise<T>((resolve, reject) => {
      promise
        .then(value => {
          if (!isCanceled) resolve(value)
        })
        .catch(error => {
          if (!isCanceled) reject(error)
        })
    })

    return {
      promise: wrappedPromise,
      cancel: () => {
        isCanceled = true
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

