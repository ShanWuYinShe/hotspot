import { BaseService } from './base.service'
import { FEED } from '~/config/api'
import type {
  PaginatedData,
  FeedMomentsRequest,
  MomentItem,
  FeedGroupsRequest,
  FeedGroupItem,
  FeedUsersRequest,
  FeedUserItem,
  FeedTrendingRequest,
  FeedTrendingData,
  FeedFeedbackRequest
} from '~/types/api'

class FeedService extends BaseService {
  // 获取瞬间信息流
  async getMoments(request: FeedMomentsRequest = {}) {
    return this.post<PaginatedData<MomentItem>>(FEED.MOMENTS, request)
  }

  // 获取群组推荐
  async getGroups(request: FeedGroupsRequest = {}) {
    return this.post<PaginatedData<FeedGroupItem>>(FEED.GROUPS, request)
  }

  // 获取用户推荐
  async getUsers(request: FeedUsersRequest = {}) {
    return this.post<PaginatedData<FeedUserItem>>(FEED.USERS, request)
  }

  // 获取热门内容
  async getTrending(request: FeedTrendingRequest = {}) {
    return this.post<FeedTrendingData>(FEED.TRENDING, request)
  }

  // 反馈信息流内容
  async feedback(request: FeedFeedbackRequest) {
    return this.post<void>(FEED.FEEDBACK, request)
  }
}

export const feedService = new FeedService()
export function useFeedService() { return feedService }
