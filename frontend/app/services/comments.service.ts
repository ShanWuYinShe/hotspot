import { BaseService } from './base.service'
import { COMMENTS } from '~/config/api'
import type {
  PaginatedData,
  CreateCommentRequest,
  CreateCommentData,
  DeleteCommentRequest,
  CommentsRequest,
  CommentListItem,
  CommentRepliesRequest,
  LikeCommentRequest,
  UnlikeCommentRequest,
  CommentLikesRequest,
  CommentLikeItem
} from '~/types/api'

class CommentsService extends BaseService {
  async create(request: CreateCommentRequest) {
    return this.post<CreateCommentData>(COMMENTS.CREATE, request)
  }

  async delete(request: DeleteCommentRequest) {
    return this.post<void>(COMMENTS.DELETE, request)
  }

  async getList(request: CommentsRequest) {
    return this.publicPost<PaginatedData<CommentListItem>>(COMMENTS.LIST, request)
  }

  async getReplies(request: CommentRepliesRequest) {
    return this.publicPost<PaginatedData<CommentListItem>>(COMMENTS.REPLIES, request)
  }

  async like(request: LikeCommentRequest) {
    return this.post<void>(COMMENTS.LIKE, request)
  }

  async unlike(request: UnlikeCommentRequest) {
    return this.post<void>(COMMENTS.UNLIKE, request)
  }

  async getLikes(request: CommentLikesRequest) {
    return this.publicPost<PaginatedData<CommentLikeItem>>(COMMENTS.LIKES, request)
  }
}

export const commentsService = new CommentsService()
export function useCommentsService() { return commentsService }
