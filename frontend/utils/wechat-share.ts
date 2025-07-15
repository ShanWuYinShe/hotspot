/**
 * 微信分享工具函数
 * 从 composables/useChat.ts 移动到 utils
 */

import { wechatService, type WxShareOptions } from '~/services/wechat.service'

/**
 * 处理二维码分享功能，打开微信分享对话框
 */
export const handleQRCodeShare = (
  content: string, 
  qrCodeUrl: string, 
  setShowWechatShareDialog: (show: boolean) => void
) => {
  setShowWechatShareDialog(true)
}

/**
 * 创建微信分享选项
 */
export const createShareOptions = (
  title: string | (() => string),
  desc: string | (() => string),
  link: string | (() => string),
  imgUrl: string | (() => string)
): WxShareOptions => {
  return {
    title: typeof title === 'function' ? title() : title,
    desc: typeof desc === 'function' ? desc() : desc,
    link: typeof link === 'function' ? link() : link,
    imgUrl: typeof imgUrl === 'function' ? imgUrl() : imgUrl
  }
}

/**
 * 为群组聊天创建分享选项
 */
export const createGroupShareOptions = (
  groupInfo: any,
  groupInviteLink: string,
  defaultImage: string = '/images/default-group.png'
): WxShareOptions => {
  return createShareOptions(
    () => `加入群组：${groupInfo.name}`,
    () => groupInfo.description || '一起来聊天吧！',
    () => groupInviteLink,
    () => groupInfo.avatar || defaultImage
  )
}

/**
 * 为用户资料创建分享选项
 */
export const createUserShareOptions = (
  userInfo: any,
  userProfileLink: string,
  defaultImage: string = '/images/default-avatar.png'
): WxShareOptions => {
  return createShareOptions(
    () => `查看用户：${userInfo.nickname}`,
    () => `来看看 ${userInfo.nickname} 的资料`,
    () => userProfileLink,
    () => userInfo.avatar || defaultImage
  )
}

/**
 * 为位置创建分享选项
 */
export const createLocationShareOptions = (
  locationInfo: any,
  locationLink: string,
  defaultImage: string = '/images/default-location.png'
): WxShareOptions => {
  return createShareOptions(
    () => `位置分享：${locationInfo.name}`,
    () => `我在 ${locationInfo.name}，快来找我吧！`,
    () => locationLink,
    () => locationInfo.image || defaultImage
  )
}
