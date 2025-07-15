/**
 * 微信服务
 * 提供微信相关功能的API封装
 * 使用 Composition API 格式，提高代码复用性
 */

import { useApiService } from './api.factory'

// 微信JSSDK配置参数
export interface WxJssdkConfig {
  appId: string
  timestamp: number
  nonceStr: string
  signature: string
  jsApiList: string[]
}

// 微信分享参数
export interface WxShareOptions {
  title: string
  desc: string
  link: string
  imgUrl: string
  success?: () => void
  cancel?: () => void
  fail?: (error: any) => void
}

/**
 * 微信服务 Composition API
 * 统一管理API服务实例，避免重复调用
 */
export function useWechatService() {
  const apiService = useApiService()

  /**
   * 获取微信JSSDK配置
   */
  const getWxJssdkConfig = async (url: string): Promise<WxJssdkConfig> => {
    const result = await apiService.safeRequest<WxJssdkConfig>('/wechat/jssdk/config', { url })
    if (!result) throw new Error('获取微信JSSDK配置失败')
    return result
  }

  /**
   * 初始化微信JSSDK
   */
  const initWxJssdk = async (jsApiList: string[] = [
    'updateAppMessageShareData',
    'updateTimelineShareData',
    'onMenuShareTimeline',
    'onMenuShareAppMessage'
  ]): Promise<boolean> => {
    if (typeof window === 'undefined' || !window.wx) {
      console.error('微信JSSDK未加载')
      return false
    }

    try {
      // 获取当前URL，去除hash部分
      const url = window.location.href.split('#')[0]
      
      // 获取JSSDK配置
      const config = await getWxJssdkConfig(url)
      config.jsApiList = jsApiList

      // 初始化JSSDK
      return new Promise((resolve, reject) => {
        window.wx.config({
          ...config,
          debug: false
        })

        window.wx.ready(() => {
          console.log('微信JSSDK初始化成功')
          resolve(true)
        })

        window.wx.error((res: any) => {
          console.error('微信JSSDK初始化失败:', res)
          reject(new Error(`微信JSSDK初始化失败: ${res.errMsg || '未知错误'}`))
        })
      })
    } catch (error) {
      console.error('获取微信JSSDK配置失败:', error)
      return false
    }
  }

  /**
   * 分享到微信好友
   */
  const shareToWxFriend = (options: WxShareOptions): Promise<boolean> => {
    return new Promise((resolve, reject) => {
      if (typeof window === 'undefined' || !window.wx) {
        reject(new Error('微信JSSDK未加载'))
        return
      }

      window.wx.updateAppMessageShareData({
        title: options.title,
        desc: options.desc,
        link: options.link,
        imgUrl: options.imgUrl,
        success: () => {
          console.log('分享到微信好友成功')
          options.success?.()
          resolve(true)
        },
        cancel: () => {
          console.log('取消分享到微信好友')
          options.cancel?.()
          resolve(false)
        },
        fail: (error: any) => {
          console.error('分享到微信好友失败:', error)
          options.fail?.(error)
          reject(new Error(`分享失败: ${error.errMsg || '未知错误'}`))
        }
      })
    })
  }

  /**
   * 分享到微信朋友圈
   */
  const shareToWxTimeline = (options: WxShareOptions): Promise<boolean> => {
    return new Promise((resolve, reject) => {
      if (typeof window === 'undefined' || !window.wx) {
        reject(new Error('微信JSSDK未加载'))
        return
      }

      window.wx.updateTimelineShareData({
        title: options.title,
        link: options.link,
        imgUrl: options.imgUrl,
        success: () => {
          console.log('分享到微信朋友圈成功')
          options.success?.()
          resolve(true)
        },
        cancel: () => {
          console.log('取消分享到微信朋友圈')
          options.cancel?.()
          resolve(false)
        },
        fail: (error: any) => {
          console.error('分享到微信朋友圈失败:', error)
          options.fail?.(error)
          reject(new Error(`分享失败: ${error.errMsg || '未知错误'}`))
        }
      })
    })
  }

  /**
   * 设置微信分享配置
   */
  const setWxShareConfig = async (options: WxShareOptions): Promise<void> => {
    try {
      // 分享给朋友
      await shareToWxFriend(options)
      // 分享到朋友圈
      await shareToWxTimeline(options)
    } catch (error) {
      console.error('设置微信分享配置失败:', error)
      throw error
    }
  }

  return {
    getWxJssdkConfig,
    initWxJssdk,
    shareToWxFriend,
    shareToWxTimeline,
    setWxShareConfig
  }
}

declare global {
  interface Window {
    wx?: any
  }
}
