import { BaseService } from './base.service'
import { MOMENTS, USER } from '~/config/api'
import type {
  PaginatedData,
  MomentItem,
  CreateMomentRequest,
  CreateMomentData,
  UpdateMomentRequest,
  FavoriteMomentRequest,
  UnfavoriteMomentRequest,
  MyFavoritesRequest,
  SearchMomentsByKeywordRequest,
  SearchMomentsByTagRequest,
  SearchMomentsByLocationRequest,
  UserMomentsRequest,
  MyMomentsRequest,
  MomentOpenTheDoorRequest,
  DeleteMomentRequest,
  LikeMomentRequest,
  UnlikeMomentRequest,
  CheckLikedRequest,
  CheckLikedData,
  CheckFavoritedRequest,
  CheckFavoritedData,
  MomentLikesRequest,
  MomentLikeItem,
  CommentListItem,
  MomentCommentsRequest
} from '~/types/api'

class MomentsService extends BaseService {
  // ============ 搜索 API (public) ============

  async searchByKeyword(request: SearchMomentsByKeywordRequest) {
    return this.publicPost<PaginatedData<MomentItem>>(MOMENTS.SEARCH_BY_KEYWORD, request)
  }

  async searchByTag(request: SearchMomentsByTagRequest) {
    return this.publicPost<PaginatedData<MomentItem>>(MOMENTS.SEARCH_BY_TAG, request)
  }

  async searchByLocation(request: SearchMomentsByLocationRequest) {
    return this.publicPost<PaginatedData<MomentItem>>(MOMENTS.SEARCH_BY_LOCATION, request)
  }

  async getComments(request: MomentCommentsRequest) {
    return this.publicPost<PaginatedData<CommentListItem>>(MOMENTS.COMMENTS, request)
  }

  // 获取指定用户瞬间列表（公开 API）
  async getUserMoments(request: UserMomentsRequest) {
    return this.publicPost<PaginatedData<MomentItem>>(USER.MOMENTS, request)
  }

  async getLikes(request: MomentLikesRequest) {
    return this.publicPost<PaginatedData<MomentLikeItem>>(MOMENTS.LIKES, request)
  }

  // 获取瞬间详情（通过 ID 获取指定瞬间的详情）
  async openTheDoor(request: MomentOpenTheDoorRequest) {
    return this.publicPost<MomentItem>(MOMENTS.OPEN_THE_DOOR, request)
  }

  // ============ 操作 API (protected) ============

  async create(request: CreateMomentRequest) {
    return this.post<MomentItem>(MOMENTS.CREATE, request)
  }

  async update(request: UpdateMomentRequest) {
    return this.post<MomentItem>(MOMENTS.UPDATE, request)
  }

  async deleteMoment(request: DeleteMomentRequest) {
    return this.post<void>(MOMENTS.DELETE, request)
  }

  async likeMoment(request: LikeMomentRequest) {
    return this.post<void>(MOMENTS.LIKE, request)
  }

  async unlikeMoment(request: UnlikeMomentRequest) {
    return this.post<void>(MOMENTS.UNLIKE, request)
  }

  async favoriteMoment(request: FavoriteMomentRequest) {
    return this.post<void>(MOMENTS.FAVORITE, request)
  }

  async unfavoriteMoment(request: UnfavoriteMomentRequest) {
    return this.post<void>(MOMENTS.UNFAVORITE, request)
  }

  async checkLiked(request: CheckLikedRequest) {
    return this.post<CheckLikedData>(MOMENTS.LIKED, request)
  }

  async checkFavorited(request: CheckFavoritedRequest) {
    return this.post<CheckFavoritedData>(MOMENTS.FAVORITED, request)
  }

  // 获取当前用户瞬间列表（/me/ API）
  async getMyMoments(request: MyMomentsRequest) {
    return this.post<PaginatedData<MomentItem>>(MOMENTS.MY, request)
  }

  async getFavorites(request: MyFavoritesRequest) {
    return this.post<PaginatedData<MomentItem>>(MOMENTS.MY_FAVORITES, request)
  }
}

export const momentsService = new MomentsService()
export function useMomentsService() { return momentsService }
