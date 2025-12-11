import { BaseService } from './base.service'
import { COMMENTS } from '~/config/api'
import type {
  PaginatedData,
  CreateCommentRequest,
  CreateCommentData,
  DeleteCommentRequest,
  GetCommentsRequest,
  CommentListItem,
  GetRepliesRequest,
  LikeCommentRequest,
  UnlikeCommentRequest,
  GetCommentLikesRequest,
  CommentLikeItem
} from '~/types/api'

class CommentsService extends BaseService {
  async add(request: CreateCommentRequest) {
    return this.post<CreateCommentData>(COMMENTS.ADD, request)
  }

  async remove(request: DeleteCommentRequest) {
    return this.post<void>(COMMENTS.REMOVE, request)
  }

  async getList(request: GetCommentsRequest) {
    return this.publicPost<PaginatedData<CommentListItem>>(COMMENTS.GET_LIST, request)
  }

  async getReplies(request: GetRepliesRequest) {
    return this.publicPost<PaginatedData<CommentListItem>>(COMMENTS.GET_REPLIES, request)
  }

  async likeComment(request: LikeCommentRequest) {
    return this.post<void>(COMMENTS.LIKE_COMMENT, request)
  }

  async unlikeComment(request: UnlikeCommentRequest) {
    return this.post<void>(COMMENTS.UNLIKE_COMMENT, request)
  }

  async getLikes(request: GetCommentLikesRequest) {
    return this.publicPost<PaginatedData<CommentLikeItem>>(COMMENTS.GET_LIKES_LIST, request)
  }
}

export const commentsService = new CommentsService()
export function useCommentsService() { return commentsService }
