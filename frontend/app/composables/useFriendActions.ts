/**
 * 好友操作管理 - 统一处理所有好友相关操作
 * 简洁而不简陋：提供完整的好友功能，但保持API简洁
 */

import type { UserResponse } from '~/types/api'
import { useUserService } from '~/services/users.service'

export interface FriendRequest {
  relationship_id: string
  user_id: string
  nickname: string
  avatar_url?: string
  message?: string
  created_at: string
  processing?: boolean
}

export const useFriendActions = () => {
  const userService = useUserService()
  const toast = useToast()

  /**
   * 发送好友申请
   */
  const sendFriendRequest = async (user: UserResponse, message?: string) => {
    await executeFeature('friends', async () => {
      await userService.addFriend(user.user_id, message)
      
      toast.add({
        title: '好友申请已发送',
        description: `已向 ${user.nickname} 发送好友申请`,
        color: 'success'
      })
    })
  }

  /**
   * 接受好友申请
   */
  const acceptFriendRequest = async (request: FriendRequest) => {
    await executeFeature('friends', async () => {
      await userService.acceptFriend(request.relationship_id)
      
      toast.add({
        title: '已接受好友申请',
        description: `你和 ${request.nickname} 现在是好友了`,
        color: 'success'
      })
    })
  }

  /**
   * 拒绝好友申请
   */
  const rejectFriendRequest = async (request: FriendRequest) => {
    await executeFeature('friends', async () => {
      await userService.rejectFriend(request.relationship_id)
      
      toast.add({
        title: '已拒绝好友申请',
        description: `已拒绝 ${request.nickname} 的好友申请`,
        color: 'info'
      })
    })
  }

  /**
   * 删除好友 - 通过取消关注实现
   */
  const removeFriend = async (user: UserResponse) => {
    const confirmed = confirm(`确定要删除好友 ${user.nickname} 吗？`)
    if (!confirmed) return

    await executeFeature('friends', async () => {
      // 简化实现：通过取消关注来删除好友关系
      await userService.unfollowUser(user.user_id)
      
      toast.add({
        title: '已删除好友',
        description: `已删除好友 ${user.nickname}`,
        color: 'success'
      })
    })
  }

  /**
   * 关注用户
   */
  const followUser = async (user: UserResponse) => {
    await executeFeature('userFollow', async () => {
      await userService.followUser(user.user_id)
      
      toast.add({
        title: '关注成功',
        description: `已关注 ${user.nickname}`,
        color: 'success'
      })
    })
  }

  /**
   * 取消关注
   */
  const unfollowUser = async (user: UserResponse) => {
    await executeFeature('userFollow', async () => {
      await userService.unfollowUser(user.user_id)
      
      toast.add({
        title: '已取消关注',
        description: `已取消关注 ${user.nickname}`,
        color: 'info'
      })
    })
  }

  /**
   * 拉黑用户
   */
  const blockUser = async (user: UserResponse, reason?: string) => {
    const confirmed = confirm(`确定要拉黑用户 ${user.nickname} 吗？`)
    if (!confirmed) return

    await executeFeature('userBlock', async () => {
      await userService.blockUser(user.user_id, reason)
      
      toast.add({
        title: '已拉黑用户',
        description: `已拉黑 ${user.nickname}`,
        color: 'warning'
      })
    })
  }

  /**
   * 获取好友列表
   */
  const getFriendsList = async (options?: {
    limit?: number
    cursor?: string
    status?: 'pending' | 'accepted' | 'all'
  }) => {
    let result: any = null
    await executeFeature('friends', async () => {
      result = await userService.getFriendsList(options)
    })
    return result
  }

  /**
   * 获取好友申请列表
   */
  const getFriendRequests = async (type: 'received' | 'sent' = 'received') => {
    let result: any = null
    await executeFeature('friends', async () => {
      // 根据类型获取不同的好友申请
      if (type === 'received') {
        result = await userService.getFriendsList({ status: 'pending' })
      } else {
        // 发送的申请需要特殊处理，这里简化实现
        result = await userService.getFriendsList({ status: 'pending' })
      }
    })
    return result
  }

  /**
   * 批量操作好友申请
   */
  const batchHandleFriendRequests = async (
    requests: FriendRequest[],
    action: 'accept' | 'reject'
  ) => {
    const results = []
    
    for (const request of requests) {
      try {
        if (action === 'accept') {
          await acceptFriendRequest(request)
        } else {
          await rejectFriendRequest(request)
        }
        results.push({ success: true, request })
      } catch (error) {
        results.push({ success: false, request, error })
      }
    }
    
    const successCount = results.filter(r => r.success).length
    const failCount = results.length - successCount
    
    if (successCount > 0) {
      toast.add({
        title: `批量${action === 'accept' ? '接受' : '拒绝'}完成`,
        description: `成功处理 ${successCount} 个申请${failCount > 0 ? `，失败 ${failCount} 个` : ''}`,
        color: failCount > 0 ? 'warning' : 'success'
      })
    }
    
    return results
  }

  return {
    // 基础操作
    sendFriendRequest,
    acceptFriendRequest,
    rejectFriendRequest,
    removeFriend,
    
    // 关注操作
    followUser,
    unfollowUser,
    
    // 拉黑操作
    blockUser,
    
    // 数据获取
    getFriendsList,
    getFriendRequests,
    
    // 批量操作
    batchHandleFriendRequests
  }
}
