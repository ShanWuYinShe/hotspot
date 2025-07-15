/**
 * 统一服务导出
 * 提供所有API服务的统一访问入口
 * 使用 Composition API 格式
 */

// 导出所有 Composition API 服务
export { useUserService } from './user.service'
export { useGroupService } from './group.service' 
export { useMomentService } from './moment.service'
export { useMessageService } from './message.service'
export { useCheckinService } from './checkin.service'
export { useLocationService } from './location.service'
export { useWechatService } from './wechat.service'
export { useWebSocketNotificationService } from './websocket-notification.service'

// 导出核心服务
export { useApiService } from './api.factory'

// Composition API 使用示例：
// import { useUserService } from '~/services'
// const { searchUsersByLocation, searchUsersByName } = useUserService()
// const result = await searchUsersByLocation(params)