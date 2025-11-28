/**
 * 动态 Store - 改进版本
 * 使用 readonly，但不持久化（动态数据应从服务器获取）
 */

import { defineStore } from 'pinia'
import type { MomentResponse, PaginatedData } from '~/types/api'

export interface Moment extends MomentResponse {
  // 扩展字段
  isLiking?: boolean
  isDeleting?: boolean
  optimisticLike?: boolean
  localImages?: string[] // 本地上传的图片预览
  is_favorited?: boolean // 是否已收藏
}

export interface MomentComment {
  comment_id: string
  moment_id: string
  user_id: string
  user_name: string
  user_avatar?: string
  content: string
  created_at: string
  updated_at: string
}

export const useMomentsStore = defineStore('moments', () => {
  // ============ 内部状态 (私有) ============
  const _moments = ref<Map<string, Moment>>(new Map())
  const _comments = ref<Map<string, MomentComment[]>>(new Map())
  
  // 不同类型的动态列表
  const _feeds = ref({
    nearby: [] as Moment[],
    following: [] as Moment[],
    mine: [] as Moment[],
    trending: [] as Moment[]
  })
  
  // 临时状态（不需要持久化）
  const _loading = ref({
    nearby: false,
    following: false,
    mine: false,
    trending: false,
    creating: false,
    comments: false
  })
  
  // 分页状态（不需要持久化）
  const _pagination = ref({
    nearby: { hasMore: false, cursor: undefined as string | undefined },
    following: { hasMore: false, cursor: undefined as string | undefined },
    mine: { hasMore: false, cursor: undefined as string | undefined },
    trending: { hasMore: false, cursor: undefined as string | undefined }
  })

  // ============ 公开状态 ============
  // 注意：不使用 readonly() 包装，因为 Pinia 需要在 hydration 时能够修改这些状态
  // 外部应该通过 actions 修改状态，而不是直接修改
  const moments = _moments
  const comments = _comments
  const feeds = _feeds
  const loading = _loading
  const pagination = _pagination

  // ============ 计算属性 ============
  const momentsList = computed(() => Array.from(_moments.value.values()))
  
  const nearbyMoments = computed(() => _feeds.value.nearby)
  const followingMoments = computed(() => _feeds.value.following)
  const myMoments = computed(() => _feeds.value.mine)
  const trendingMoments = computed(() => _feeds.value.trending)
  
  const myMomentsCount = computed(() => _feeds.value.mine.length)

  // ============ Actions ============
  
  /**
   * 设置附近动态数据（由调用者提供）
   */
  const setNearbyMoments = (moments: Moment[], hasMore: boolean, cursor?: string, loadMore = false) => {
    const newMoments = moments.map(item => ({ ...item } as Moment))
    
    if (loadMore) {
      _feeds.value.nearby.push(...newMoments)
    } else {
      _feeds.value.nearby = newMoments
    }
    
    // 更新到 moments Map
    newMoments.forEach(moment => {
      _moments.value.set(moment.moment_id, moment)
    })
    
    // 更新分页信息
    _pagination.value.nearby = {
      hasMore,
      cursor: cursor || ''
    }
    
    return newMoments
  }

  /**
   * 设置关注动态数据（由调用者提供）
   */
  const setFollowingMoments = (moments: Moment[], hasMore: boolean, cursor?: string, loadMore = false) => {
    const newMoments = moments.map(item => ({ ...item } as Moment))
    
    if (loadMore) {
      _feeds.value.following.push(...newMoments)
    } else {
      _feeds.value.following = newMoments
    }
    
    // 更新到 moments Map
    newMoments.forEach(moment => {
      _moments.value.set(moment.moment_id, moment)
    })
    
    // 更新分页信息
    _pagination.value.following = {
      hasMore,
      cursor: cursor || ''
    }
    
    return newMoments
  }

  /**
   * 设置我的动态数据（由调用者提供）
   */
  const setMyMoments = (moments: Moment[], hasMore: boolean, cursor?: string, loadMore = false) => {
    const newMoments = moments.map(item => ({ ...item } as Moment))
    
    if (loadMore) {
      _feeds.value.mine.push(...newMoments)
    } else {
      _feeds.value.mine = newMoments
    }
    
    // 更新到 moments Map
    newMoments.forEach(moment => {
      _moments.value.set(moment.moment_id, moment)
    })
    
    // 更新分页信息
    _pagination.value.mine = {
      hasMore,
      cursor: cursor || ''
    }
    
    return newMoments
  }

  /**
   * 设置热门动态数据（由调用者提供）
   */
  const setTrendingMoments = (moments: Moment[], hasMore: boolean, cursor?: string, loadMore = false) => {
    const newMoments = moments.map(item => ({ ...item } as Moment))
    
    if (loadMore) {
      _feeds.value.trending.push(...newMoments)
    } else {
      _feeds.value.trending = newMoments
    }
    
    // 更新到 moments Map
    newMoments.forEach(moment => {
      _moments.value.set(moment.moment_id, moment)
    })
    
    // 更新分页信息
    _pagination.value.trending = {
      hasMore,
      cursor: cursor || ''
    }
    
    return newMoments
  }

  /**
   * 添加新创建的动态（由调用者提供）
   */
  const addCreatedMoment = (moment: Moment) => {
    const momentData = { ...moment } as Moment
    _moments.value.set(moment.moment_id, momentData)
    _feeds.value.mine.unshift(momentData)
    return momentData
  }

  /**
   * 设置点赞状态（由调用者提供）
   */
  const setLikeStatus = (momentId: string, isLiked: boolean, isLiking = false) => {
    const moment = _moments.value.get(momentId)
    if (moment) {
      moment.isLiking = isLiking
      moment.liked_by_me = isLiked
      moment.likes_count += isLiked ? 1 : -1
    }
  }

  /**
   * 回滚点赞状态
   */
  const rollbackLikeStatus = (momentId: string, wasLiked: boolean) => {
    const moment = _moments.value.get(momentId)
    if (moment) {
      moment.liked_by_me = wasLiked
      moment.likes_count += wasLiked ? 1 : -1
      moment.isLiking = false
      moment.optimisticLike = undefined
    }
  }

  /**
   * 设置删除状态（由调用者提供）
   */
  const setDeleteStatus = (momentId: string, isDeleting: boolean) => {
    const moment = _moments.value.get(momentId)
    if (moment) {
      moment.isDeleting = isDeleting
    }
  }

  /**
   * 删除动态数据
   */
  const removeMoment = (momentId: string) => {
    // 从所有列表中移除
    _moments.value.delete(momentId)
    _comments.value.delete(momentId)
    
    Object.keys(_feeds.value).forEach(feedType => {
      _feeds.value[feedType as keyof typeof _feeds.value] = _feeds.value[feedType as keyof typeof _feeds.value]
        .filter(m => m.moment_id !== momentId)
    })
  }

  /**
   * 设置动态详情（由调用者提供）
   */
  const setMomentDetails = (momentId: string, moment: Moment) => {
    const momentData = { ...moment } as Moment
    _moments.value.set(momentId, momentData)
    return momentData
  }

  /**
   * 设置搜索结果（由调用者提供）
   */
  const setSearchResults = (moments: Moment[]) => {
    const searchResults = moments.map(item => ({ ...item } as Moment))
    
    // 更新到 moments Map
    searchResults.forEach(moment => {
      _moments.value.set(moment.moment_id, moment)
    })
    
    return searchResults
  }

  /**
   * 设置动态评论（由调用者提供）
   */
  const setMomentComments = (momentId: string, comments: MomentComment[], loadMore = false) => {
    if (loadMore) {
      const existingComments = _comments.value.get(momentId) || []
      _comments.value.set(momentId, [...existingComments, ...comments])
    } else {
      _comments.value.set(momentId, comments)
    }
    
    return comments
  }

  /**
   * 添加新评论（由调用者提供）
   */
  const addNewComment = (momentId: string, newComment: MomentComment) => {
    // 添加到本地评论列表
    const existingComments = _comments.value.get(momentId) || []
    _comments.value.set(momentId, [...existingComments, newComment])
    
    // 更新动态的评论数
    const moment = _moments.value.get(momentId)
    if (moment) {
      moment.comments_count = (moment.comments_count || 0) + 1
    }
  }

  /**
   * 删除评论数据
   */
  const removeComment = (momentId: string, commentId: string) => {
    // 从本地评论列表中移除
    const existingComments = _comments.value.get(momentId) || []
    const updatedComments = existingComments.filter(c => c.comment_id !== commentId)
    _comments.value.set(momentId, updatedComments)
    
    // 更新动态的评论数
    const moment = _moments.value.get(momentId)
    if (moment) {
      moment.comments_count = Math.max((moment.comments_count || 0) - 1, 0)
    }
  }

  /**
   * 设置收藏状态（由调用者提供）
   */
  const setFavoriteStatus = (momentId: string, isFavorited: boolean) => {
    const moment = _moments.value.get(momentId)
    if (moment) {
      moment.is_favorited = isFavorited
    }
  }

  /**
   * 获取动态
   */
  const getMoment = (momentId: string): Moment | undefined => {
    return _moments.value.get(momentId)
  }

  /**
   * 获取动态评论
   */
  const getMomentComments = (momentId: string): MomentComment[] => {
    return _comments.value.get(momentId) || []
  }

  /**
   * 点赞生活瞬间（调用服务并更新本地状态）
   */
  const likeMoment = async (momentId: string): Promise<void> => {
    const { useMomentsService } = await import('~/services/moments.service')
    const momentsService = useMomentsService()
    await momentsService.likeMoment(momentId)
    setLikeStatus(momentId, true)
  }

  /**
   * 取消点赞生活瞬间（调用服务并更新本地状态）
   */
  const unlikeMoment = async (momentId: string): Promise<void> => {
    const { useMomentsService } = await import('~/services/moments.service')
    const momentsService = useMomentsService()
    await momentsService.unlikeMoment(momentId)
    setLikeStatus(momentId, false)
  }

  /**
   * 清空指定类型的动态数据
   */
  const clearFeed = (type: 'nearby' | 'following' | 'mine' | 'trending') => {
    switch (type) {
      case 'nearby':
        _feeds.value.nearby = []
        _pagination.value.nearby = { hasMore: false, cursor: undefined }
        break
      case 'following':
        _feeds.value.following = []
        _pagination.value.following = { hasMore: false, cursor: undefined }
        break
      case 'mine':
        _feeds.value.mine = []
        _pagination.value.mine = { hasMore: false, cursor: undefined }
        break
      case 'trending':
        _feeds.value.trending = []
        _pagination.value.trending = { hasMore: false, cursor: undefined }
        break
    }
  }

  return {
    // ============ 只读状态 ============
    moments,
    comments,
    feeds,
    loading,
    pagination,
    
    // ============ 计算属性 ============
    momentsList,
    nearbyMoments,
    followingMoments,
    myMoments,
    trendingMoments,
    myMomentsCount,
    
    // ============ 方法 ============
    setNearbyMoments,
    setFollowingMoments,
    setMyMoments,
    setTrendingMoments,
    addCreatedMoment,
    setLikeStatus,
    rollbackLikeStatus,
    setDeleteStatus,
    removeMoment,
    setMomentDetails,
    setSearchResults,
    setMomentComments,
    addNewComment,
    removeComment,
    setFavoriteStatus,
    getMoment,
    getMomentComments,
    clearFeed,
    likeMoment,
    unlikeMoment
  }
})
// 注意：这个 store 不使用 persist，因为动态数据应该从服务器获取
