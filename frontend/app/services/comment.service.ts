/**
 * 评论服务 - Composition API 风格
 * 提供评论互动相关的所有 API 调用功能
 * 
 * 基于 backend_doc.md 5.2.6 评论互动API
 */

import { 
  CHAT_ENDPOINTS,
  buildChatUrl 
} from "~/config/api";
import { 
  validateUserId,
  validateContent,
  validatePaginationParams 
} from "~/utils/api-validation";
import type { 
  ApiResponse,
  PaginatedResponse,
  AddCommentRequest,
  AddCommentResponse,
  ReplyCommentRequest,
  ReplyCommentResponse,
  DeleteCommentRequest,
  DeleteCommentResponse,
  GetCommentsRequest,
  GetCommentsResponse,
  CommentResponse
} from "~/types/api";

export const useCommentService = () => {
  const api = useApi();

  /**
   * 添加评论
   */
  const addComment = async (
    momentId: string,
    content: string
  ): Promise<AddCommentResponse> => {
    // 发送前验证参数
    const validMomentId = validateUserId(momentId); // 通用ID验证
    const validContent = validateContent(content, 1, 500);

    const request: AddCommentRequest = {
      moment_id: validMomentId,
      content: validContent
    };

    const response = await api.post<ApiResponse<AddCommentResponse>>(
      '/api/v1/protected/comments/add', // 使用评论专用端点
      request,
      { auth: true }
    );

    return response.data!;
  };

  /**
   * 回复评论
   */
  const replyComment = async (
    commentId: string,
    content: string
  ): Promise<ReplyCommentResponse> => {
    // 发送前验证参数
    const validCommentId = validateUserId(commentId); // 通用ID验证
    const validContent = validateContent(content, 1, 500);

    const request: ReplyCommentRequest = {
      parent_comment_id: validCommentId,
      content: validContent
    };

    const response = await api.post<ApiResponse<ReplyCommentResponse>>(
      '/api/v1/protected/comments/reply', // 使用评论专用端点
      request,
      { auth: true }
    );

    return response.data!;
  };

  /**
   * 删除评论
   */
  const deleteComment = async (
    commentId: string
  ): Promise<DeleteCommentResponse> => {
    const validCommentId = validateUserId(commentId); // 通用ID验证

    const request: DeleteCommentRequest = {
      comment_id: validCommentId
    };

    const response = await api.post<ApiResponse<DeleteCommentResponse>>(
      '/api/v1/protected/comments/delete', // 使用评论专用端点
      request,
      { auth: true }
    );

    return response.data!;
  };

  /**
   * 获取评论列表
   */
  const getComments = async (
    momentId: string,
    cursor?: string,
    limit: number = 20
  ): Promise<PaginatedResponse<CommentResponse>> => {
    // 发送前验证参数
    const validMomentId = validateUserId(momentId); // 通用ID验证
    const validLimit = validatePaginationParams(cursor, limit).limit;

    const request: GetCommentsRequest = {
      moment_id: validMomentId,
      cursor,
      limit: validLimit
    };

    const response = await api.post<ApiResponse<PaginatedResponse<CommentResponse>>>(
      '/api/v1/public/comments/list', // 公开端点，可以查看评论
      request
    );

    return response.data!;
  };

  /**
   * 获取用户的评论历史
   */
  const getUserComments = async (
    userId?: string,
    cursor?: string,
    limit: number = 20
  ): Promise<PaginatedResponse<CommentResponse>> => {
    // 发送前验证参数
    const validUserId = userId ? validateUserId(userId) : undefined;
    const validLimit = validatePaginationParams(cursor, limit).limit;

    const request: GetCommentsRequest = {
      user_id: validUserId, // 如果不提供user_id，则获取当前用户的评论
      cursor,
      limit: validLimit
    };

    const response = await api.post<ApiResponse<PaginatedResponse<CommentResponse>>>(
      '/api/v1/protected/comments/list', // 需要认证的端点
      request,
      { auth: true }
    );

    return response.data!;
  };

  /**
   * 点赞评论（扩展功能）
   */
  const likeComment = async (commentId: string): Promise<{ success: boolean; message: string }> => {
    // 这个功能在当前API规范中没有定义，作为扩展功能预留
    const validCommentId = validateUserId(commentId);
    
    // 暂时返回成功响应，实际需要后端支持
    return {
      success: true,
      message: '点赞成功'
    };
  };

  /**
   * 取消点赞评论（扩展功能）
   */
  const unlikeComment = async (commentId: string): Promise<{ success: boolean; message: string }> => {
    // 这个功能在当前API规范中没有定义，作为扩展功能预留
    const validCommentId = validateUserId(commentId);
    
    // 暂时返回成功响应，实际需要后端支持
    return {
      success: true,
      message: '取消点赞成功'
    };
  };

  /**
   * 获取评论的回复列表
   */
  const getCommentReplies = async (
    commentId: string,
    cursor?: string,
    limit: number = 10
  ): Promise<PaginatedResponse<CommentResponse>> => {
    // 通过获取评论列表并过滤parent_comment_id来实现
    // 实际应该有专门的API端点
    const validCommentId = validateUserId(commentId);
    const validLimit = validatePaginationParams(cursor, limit).limit;

    // 暂时使用通用评论列表API，实际需要专门的回复API
    const request: GetCommentsRequest = {
      parent_comment_id: validCommentId,
      cursor,
      limit: validLimit
    };

    const response = await api.post<ApiResponse<PaginatedResponse<CommentResponse>>>(
      '/api/v1/public/comments/list',
      request
    );

    return response.data!;
  };

  /**
   * 格式化评论时间显示
   */
  const formatCommentTime = (createdAt: string): string => {
    const now = new Date();
    const commentTime = new Date(createdAt);
    const diffMs = now.getTime() - commentTime.getTime();
    const diffMinutes = Math.floor(diffMs / (1000 * 60));
    const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

    if (diffMinutes < 1) {
      return '刚刚';
    } else if (diffMinutes < 60) {
      return `${diffMinutes}分钟前`;
    } else if (diffHours < 24) {
      return `${diffHours}小时前`;
    } else if (diffDays < 7) {
      return `${diffDays}天前`;
    } else {
      return commentTime.toLocaleDateString('zh-CN');
    }
  };

  /**
   * 验证评论内容
   */
  const validateCommentContent = (content: string): { isValid: boolean; error?: string } => {
    if (!content || content.trim().length === 0) {
      return { isValid: false, error: '评论内容不能为空' };
    }
    
    if (content.length > 500) {
      return { isValid: false, error: '评论内容不能超过500字符' };
    }
    
    // 检查是否包含敏感词（简单示例）
    const sensitiveWords = ['垃圾', '广告', '色情'];
    const hasSensitiveWord = sensitiveWords.some(word => content.includes(word));
    if (hasSensitiveWord) {
      return { isValid: false, error: '评论内容包含敏感词汇' };
    }
    
    return { isValid: true };
  };

  return {
    // 核心API方法
    addComment,
    replyComment,
    deleteComment,
    getComments,
    getUserComments,
    getCommentReplies,
    
    // 扩展功能
    likeComment,
    unlikeComment,
    
    // 工具方法
    formatCommentTime,
    validateCommentContent
  };
};

/**
 * 评论相关的常量定义
 */
export const COMMENT_CONSTANTS = {
  // 评论内容限制
  MAX_CONTENT_LENGTH: 500,
  MIN_CONTENT_LENGTH: 1,
  
  // 默认分页大小
  DEFAULT_PAGE_SIZE: 20,
  DEFAULT_REPLY_PAGE_SIZE: 10,
  
  // 评论状态
  STATUSES: {
    ACTIVE: 'active',
    DELETED: 'deleted',
    HIDDEN: 'hidden'
  },
  
  // 评论类型
  TYPES: {
    COMMENT: 'comment',
    REPLY: 'reply'
  }
} as const;
