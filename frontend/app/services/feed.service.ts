import { BaseService } from './base.service'
import { FEED } from '~/config/api'
import type {
  PaginatedData,
  GetFeedMomentsRequest,
  FeedMomentItem,
  GetFeedGroupsRequest,
  FeedGroupItem,
  GetFeedUsersRequest,
  FeedUserItem,
  GetFeedTrendingRequest,
  FeedTrendingData,
  FeedFeedbackRequest
} from '~/types/api'

class FeedService extends BaseService {
  // 获取瞬间信息流
  async getMoments(request: GetFeedMomentsRequest = {}) {
    return this.post<PaginatedData<FeedMomentItem>>(FEED.MOMENTS, request)
  }

  // 获取群组推荐
  async getGroups(request: GetFeedGroupsRequest = {}) {
    return this.post<PaginatedData<FeedGroupItem>>(FEED.GROUPS, request)
  }

  // 获取用户推荐
  async getUsers(request: GetFeedUsersRequest = {}) {
    return this.post<PaginatedData<FeedUserItem>>(FEED.USERS, request)
  }

  // 获取热门内容
  async getTrending(request: GetFeedTrendingRequest = {}) {
    return this.post<FeedTrendingData>(FEED.TRENDING, request)
  }

  // 反馈信息流内容
  async feedback(request: FeedFeedbackRequest) {
    return this.post<void>(FEED.FEEDBACK, request)
  }
}

export const feedService = new FeedService()
export function useFeedService() { return feedService }
