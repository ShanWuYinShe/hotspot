import { BaseService } from './base.service'
import { MOMENTS } from '~/config/api'
import type {
  PaginatedData,
  MomentSearchItem,
  CreateMomentRequest,
  CreateMomentData,
  EditMomentRequest,
  FavoriteMomentRequest,
  UnfavoriteMomentRequest,
  GetFavoritesRequest,
  SearchMomentsByKeywordRequest,
  SearchMomentsByTagRequest,
  SearchMomentsByLocationRequest,
  GetUserMomentsRequest,
  GetMyMomentsRequest,
  MomentOpenTheDoorRequest,
  DeleteMomentRequest,
  LikeMomentRequest,
  UnlikeMomentRequest,
  CheckMomentLikedRequest,
  CheckLikedData,
  CheckMomentFavoritedRequest,
  CheckFavoritedData,
  GetMomentLikesRequest,
  MomentLikeItem,
  CommentListItem,
  GetCommentsRequest
} from '~/types/api'

class MomentsService extends BaseService {
  // ============ 搜索 API (public) ============

  async searchByKeyword(request: SearchMomentsByKeywordRequest) {
    return this.publicPost<PaginatedData<MomentSearchItem>>(MOMENTS.SEARCH_BY_KEYWORD, request)
  }

  async searchByTag(request: SearchMomentsByTagRequest) {
    return this.publicPost<PaginatedData<MomentSearchItem>>(MOMENTS.SEARCH_BY_TAG, request)
  }

  async searchByLocation(request: SearchMomentsByLocationRequest) {
    return this.publicPost<PaginatedData<MomentSearchItem>>(MOMENTS.SEARCH_BY_LOCATION, request)
  }

  async getComments(request: GetCommentsRequest) {
    return this.publicPost<PaginatedData<CommentListItem>>(MOMENTS.GET_COMMENTS_LIST, request)
  }

  // 获取指定用户瞬间列表（公开 API）
  async getUserMoments(request: GetUserMomentsRequest) {
    return this.publicPost<PaginatedData<MomentSearchItem>>(MOMENTS.GET_USER_MOMENTS_LIST, request)
  }

  async getLikes(request: GetMomentLikesRequest) {
    return this.publicPost<PaginatedData<MomentLikeItem>>(MOMENTS.GET_LIKES_LIST, request)
  }

  // 获取瞬间详情（通过 ID 获取指定瞬间的详情）
  async openTheDoor(request: MomentOpenTheDoorRequest) {
    return this.publicPost<MomentSearchItem>(MOMENTS.OPEN_THE_DOOR, request)
  }

  // ============ 操作 API (protected) ============

  async create(request: CreateMomentRequest) {
    return this.post<CreateMomentData>(MOMENTS.CREATE, request)
  }

  async update(request: EditMomentRequest) {
    return this.post<void>(MOMENTS.UPDATE, request)
  }

  async deleteMoment(request: DeleteMomentRequest) {
    return this.post<void>(MOMENTS.DELETE, request)
  }

  async likeMoment(request: LikeMomentRequest) {
    return this.post<void>(MOMENTS.LIKE_MOMENT, request)
  }

  async unlikeMoment(request: UnlikeMomentRequest) {
    return this.post<void>(MOMENTS.UNLIKE_MOMENT, request)
  }

  async favoriteMoment(request: FavoriteMomentRequest) {
    return this.post<void>(MOMENTS.FAVORITE_MOMENT, request)
  }

  async unfavoriteMoment(request: UnfavoriteMomentRequest) {
    return this.post<void>(MOMENTS.UNFAVORITE_MOMENT, request)
  }

  async checkLiked(request: CheckMomentLikedRequest) {
    return this.post<CheckLikedData>(MOMENTS.CHECK_LIKED, request)
  }

  async checkFavorited(request: CheckMomentFavoritedRequest) {
    return this.post<CheckFavoritedData>(MOMENTS.CHECK_FAVORITED, request)
  }

  // 获取当前用户瞬间列表（/me/ API）
  async getMyMoments(request: GetMyMomentsRequest) {
    return this.post<PaginatedData<MomentSearchItem>>(MOMENTS.GET_MY_LIST, request)
  }

  async getFavorites(request: GetFavoritesRequest) {
    return this.post<PaginatedData<MomentSearchItem>>(MOMENTS.GET_FAVORITES_LIST, request)
  }
}

export const momentsService = new MomentsService()
export function useMomentsService() { return momentsService }
