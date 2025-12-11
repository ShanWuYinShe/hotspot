/**
 * 生活瞬间状态存储 - 纯状态管理
 * 
 * ✅ 职责：管理生活瞬间相关的运行时状态
 * ❌ 不包含：API 调用（使用 service）
 * 
 * 使用 pinia-plugin-persistedstate 自动处理水合
 */

import { defineStore } from 'pinia'
import type { MomentSearchItem } from '~/types/api'

/**
 * 客户端动态数据类型
 * 扩展 MomentSearchItem（API 搜索结果），添加客户端状态字段
 */
export interface Moment extends MomentSearchItem {
  isLiking?: boolean
  isDeleting?: boolean
  optimisticLike?: boolean
  localImages?: string[]
  is_favorited?: boolean
  liked_by_me?: boolean
}

export interface MomentComment {
  comment_id: string
  moment_id: string
  user_id: number
  user_name: string
  user_avatar?: string
  content: string
  created_at: string
  updated_at: string
}

export interface MomentDraft {
  description: string
  images: string[]
  tags: string[]
  latitude: number | null
  longitude: number | null
  location_name: string
}

const DEFAULT_DRAFT: MomentDraft = {
  description: '',
  images: [],
  tags: [],
  latitude: null,
  longitude: null,
  location_name: ''
}

export const useMomentsStore = defineStore('moments', () => {
  // ============ 内部状态 ============
  const _moments = ref<Map<string, Moment>>(new Map())
  const _comments = ref<Map<string, MomentComment[]>>(new Map())
  const _draft = ref<MomentDraft>({ ...DEFAULT_DRAFT })
  
  const _feeds = ref({
    nearby: [] as Moment[],
    following: [] as Moment[],
    mine: [] as Moment[],
    trending: [] as Moment[]
  })
  
  const _loading = ref({
    nearby: false,
    following: false,
    mine: false,
    trending: false,
    creating: false,
    comments: false
  })
  
  const _pagination = ref({
    nearby: { hasMore: false, cursor: undefined as string | undefined },
    following: { hasMore: false, cursor: undefined as string | undefined },
    mine: { hasMore: false, cursor: undefined as string | undefined },
    trending: { hasMore: false, cursor: undefined as string | undefined }
  })

  // ============ Getter ============

  const moments = computed(() => _moments.value)
  const comments = computed(() => _comments.value)
  const draft = computed((): MomentDraft => _draft.value)
  const hasDraft = computed(() => {
    return _draft.value.description.trim() !== '' || _draft.value.images.length > 0
  })
  
  const feeds = computed(() => _feeds.value)
  const loading = computed(() => _loading.value)
  const pagination = computed(() => _pagination.value)
  const momentsList = computed(() => Array.from(_moments.value.values()))
  const nearbyMoments = computed(() => _feeds.value.nearby)
  const followingMoments = computed(() => _feeds.value.following)
  const myMoments = computed(() => _feeds.value.mine)
  const trendingMoments = computed(() => _feeds.value.trending)
  const myMomentsCount = computed(() => _feeds.value.mine.length)

  // ============ Setter ============

  const setFeedMoments = (
    type: 'nearby' | 'following' | 'mine' | 'trending',
    momentsList: Moment[],
    hasMore: boolean,
    cursor?: string,
    loadMore = false
  ) => {
    const newMoments = momentsList.map(item => ({ ...item }))

    if (loadMore) {
      _feeds.value[type].push(...newMoments)
    } else {
      _feeds.value[type] = newMoments
    }

    newMoments.forEach(moment => _moments.value.set(moment.id, moment))
    _pagination.value[type] = { hasMore, cursor }

    return newMoments
  }

  const setNearbyMoments = (m: Moment[], h: boolean, c?: string, l = false) => setFeedMoments('nearby', m, h, c, l)
  const setFollowingMoments = (m: Moment[], h: boolean, c?: string, l = false) => setFeedMoments('following', m, h, c, l)
  const setMyMoments = (m: Moment[], h: boolean, c?: string, l = false) => setFeedMoments('mine', m, h, c, l)
  const setTrendingMoments = (m: Moment[], h: boolean, c?: string, l = false) => setFeedMoments('trending', m, h, c, l)

  const addCreatedMoment = (moment: Moment) => {
    const data = { ...moment }
    _moments.value.set(moment.id, data)
    _feeds.value.mine.unshift(data)
    return data
  }

  const setLikeStatus = (momentId: string, isLiked: boolean, isLiking = false) => {
    const moment = _moments.value.get(momentId)
    if (moment) {
      moment.isLiking = isLiking
      moment.liked_by_me = isLiked
      moment.likes_count += isLiked ? 1 : -1
    }
  }

  const rollbackLikeStatus = (momentId: string, wasLiked: boolean) => {
    const moment = _moments.value.get(momentId)
    if (moment) {
      moment.liked_by_me = wasLiked
      moment.likes_count += wasLiked ? 1 : -1
      moment.isLiking = false
      moment.optimisticLike = undefined
    }
  }

  const setDeleteStatus = (momentId: string, isDeleting: boolean) => {
    const moment = _moments.value.get(momentId)
    if (moment) moment.isDeleting = isDeleting
  }

  const removeMoment = (momentId: string) => {
    _moments.value.delete(momentId)
    _comments.value.delete(momentId)
    Object.keys(_feeds.value).forEach(feedType => {
      const key = feedType as keyof typeof _feeds.value
      _feeds.value[key] = _feeds.value[key].filter(m => m.id !== momentId)
    })
  }

  const setMomentDetails = (momentId: string, moment: Moment) => {
    const data = { ...moment }
    _moments.value.set(momentId, data)
    return data
  }

  const setSearchResults = (momentsList: Moment[]) => {
    const results = momentsList.map(item => ({ ...item }))
    results.forEach(moment => _moments.value.set(moment.id, moment))
    return results
  }

  const setMomentComments = (momentId: string, commentsList: MomentComment[], loadMore = false) => {
    if (loadMore) {
      const existing = _comments.value.get(momentId) || []
      _comments.value.set(momentId, [...existing, ...commentsList])
    } else {
      _comments.value.set(momentId, commentsList)
    }
    return commentsList
  }

  const addNewComment = (momentId: string, newComment: MomentComment) => {
    const existing = _comments.value.get(momentId) || []
    _comments.value.set(momentId, [...existing, newComment])
    const moment = _moments.value.get(momentId)
    if (moment) moment.comments_count = (moment.comments_count || 0) + 1
  }

  const removeComment = (momentId: string, commentId: string) => {
    const existing = _comments.value.get(momentId) || []
    _comments.value.set(momentId, existing.filter(c => c.comment_id !== commentId))
    const moment = _moments.value.get(momentId)
    if (moment) moment.comments_count = Math.max((moment.comments_count || 0) - 1, 0)
  }

  const setFavoriteStatus = (momentId: string, isFavorited: boolean) => {
    const moment = _moments.value.get(momentId)
    if (moment) moment.is_favorited = isFavorited
  }

  const getMoment = (momentId: string) => _moments.value.get(momentId)
  const getMomentComments = (momentId: string) => _comments.value.get(momentId) || []

  const setLoading = (type: keyof typeof _loading.value, isLoading: boolean) => {
    _loading.value[type] = isLoading
  }

  const clearFeed = (type: 'nearby' | 'following' | 'mine' | 'trending') => {
    _feeds.value[type] = []
    _pagination.value[type] = { hasMore: false, cursor: undefined }
  }

  const saveDraft = (draftData: Partial<MomentDraft>) => {
    _draft.value = { ..._draft.value, ...draftData }
  }

  const clearDraft = () => {
    _draft.value = { ...DEFAULT_DRAFT }
  }

  return {
    // 内部状态（用于持久化）
    _draft,

    // Getter
    moments,
    comments,
    draft,
    hasDraft,
    feeds,
    loading,
    pagination,
    momentsList,
    nearbyMoments,
    followingMoments,
    myMoments,
    trendingMoments,
    myMomentsCount,

    // Setter
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
    setLoading,
    clearFeed,
    saveDraft,
    clearDraft
  }
}, {
  // 仅持久化草稿
  persist: {
    key: 'moments-draft',
    pick: ['_draft']
  }
})
