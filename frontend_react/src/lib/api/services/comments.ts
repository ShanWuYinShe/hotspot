import { apiClient } from '@/lib/api/client';
import { ENDPOINTS } from '@/lib/api/endpoints';
import {
  CommentLikeItem,
  CommentListItem,
  CreateCommentRequest,
  PaginatedData,
  PaginationParams,
} from '@/types/api';

export const CommentService = {
  create: (data: CreateCommentRequest) => 
    apiClient.post<CommentListItem>(ENDPOINTS.COMMENTS.CREATE, data),
    
  delete: (commentId: string) => 
    apiClient.post(ENDPOINTS.COMMENTS.DELETE, { comment_id: commentId }),
    
  list: (momentId: string, params?: PaginationParams) =>
    apiClient.post<PaginatedData<CommentListItem>>(ENDPOINTS.COMMENTS.LIST, {
      moment_id: momentId,
      ...params
    }),
    
  replies: (commentId: string, params?: PaginationParams) =>
    apiClient.post<PaginatedData<CommentListItem>>(ENDPOINTS.COMMENTS.REPLIES, {
      comment_id: commentId,
      ...params
    }),
    
  like: (commentId: string) => 
    apiClient.post(ENDPOINTS.COMMENTS.LIKE, { comment_id: commentId }),
    
  unlike: (commentId: string) => 
    apiClient.post(ENDPOINTS.COMMENTS.UNLIKE, { comment_id: commentId }),

  likes: (commentId: string, params?: PaginationParams) =>
    apiClient.post<PaginatedData<CommentLikeItem>>(ENDPOINTS.COMMENTS.LIKES, {
      comment_id: commentId,
      ...params,
    }),
};
