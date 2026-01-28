import { BaseService } from './base.service'
import { COMMENTS } from '~/config/api'
import type {
  PaginatedData,
  CreateCommentRequest,
  CommentListItem,
  DeleteCommentRequest,
  CommentsRequest,
  CommentRepliesRequest,
  LikeCommentRequest,
  UnlikeCommentRequest,
  CommentLikesRequest,
  CommentLikeItem
} from '~/types/api'

class CommentsService extends BaseService {
  async create(request: CreateCommentRequest) {
    return this.post<CommentListItem>(COMMENTS.CREATE, request)
  }

  async delete(request: DeleteCommentRequest) {
    return this.post<void>(COMMENTS.DELETE, request)
  }

  async getList(request: CommentsRequest) {
    return this.post<PaginatedData<CommentListItem>>(COMMENTS.LIST, request)
  }

  async getReplies(request: CommentRepliesRequest) {
    return this.post<PaginatedData<CommentListItem>>(COMMENTS.REPLIES, request)
  }

  async like(request: LikeCommentRequest) {
    return this.post<void>(COMMENTS.LIKE, request)
  }

  async unlike(request: UnlikeCommentRequest) {
    return this.post<void>(COMMENTS.UNLIKE, request)
  }

  async getLikes(request: CommentLikesRequest) {
    return this.post<PaginatedData<CommentLikeItem>>(COMMENTS.LIKES, request)
  }
}

export const commentsService = new CommentsService()
export function useCommentsService() { return commentsService }
