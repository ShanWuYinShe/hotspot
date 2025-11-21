/**
 * 排行榜服务
 * 提供创作者和生活瞬间排行榜查询功能
 * 
 * 符合 API 规范 3.15 节
 */

import { BaseService } from './base.service'
import type {
  GetCreatorLeaderboardRequest,
  GetCreatorLeaderboardData,
  GetMomentLeaderboardRequest,
  GetMomentLeaderboardData,
  GetMyCreatorRankRequest,
  MyCreatorRankData,
  GetMyMomentRankRequest,
  MyMomentRankData
} from '~/types/api'

/**
 * 排行榜服务类
 */
export class LeaderboardService extends BaseService {
  constructor() {
    super('/api/v1/public/leaderboard')
  }

  /**
   * 获取创作者排行榜
   * 3.15.1. POST /api/v1/public/leaderboard/get_creators - 获取创作者排行榜
   */
  async getCreators(request: GetCreatorLeaderboardRequest): Promise<GetCreatorLeaderboardData> {
    return this.publicPost<GetCreatorLeaderboardData>('/get_creators', request)
  }

  /**
   * 获取生活瞬间排行榜
   * 3.15.2. POST /api/v1/public/leaderboard/get_moments - 获取生活瞬间排行榜
   */
  async getMoments(request: GetMomentLeaderboardRequest): Promise<GetMomentLeaderboardData> {
    return this.publicPost<GetMomentLeaderboardData>('/get_moments', request)
  }

  /**
   * 获取我的创作者排名
   * 3.15.3. POST /api/v1/protected/leaderboard/my_creator_rank - 获取我的创作者排名
   */
  async getMyCreatorRank(request: GetMyCreatorRankRequest): Promise<MyCreatorRankData> {
    return this.post<MyCreatorRankData>('/my_creator_rank', request)
  }

  /**
   * 获取我的生活瞬间排名
   * 3.15.4. POST /api/v1/protected/leaderboard/my_moment_rank - 获取我的生活瞬间排名
   */
  async getMyMomentRank(request: GetMyMomentRankRequest): Promise<MyMomentRankData> {
    return this.post<MyMomentRankData>('/my_moment_rank', request)
  }
}

/**
 * 使用排行榜服务的 Composable
 */
export function useLeaderboardService() {
  return new LeaderboardService()
}

