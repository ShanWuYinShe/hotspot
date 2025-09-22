/**
 * 安全举报服务 - Composition API 风格
 * 提供内容安全和举报相关的所有 API 调用功能
 * 
 * 基于 backend_doc.md 5.2.7 内容安全API
 */

import { 
  REPORT_ENDPOINTS, 
  USER_ENDPOINTS,
  buildReportUrl,
  buildUserUrl 
} from "~/config/api";
import { 
  validateUserId,
  validateContent,
  validatePaginationParams 
} from "~/utils/api-validation";
import type { 
  ApiResponse,
  PaginatedResponse,
  CreateReportRequest,
  CreateReportResponse,
  GetMyReportsRequest,
  GetMyReportsResponse,
  GetReportsRequest,
  GetReportsResponse,
  HandleReportRequest,
  HandleReportResponse,
  BlockUserRequest,
  BlockUserResponse,
  ReportResponse
} from "~/types/api";

export const useSecurityService = () => {
  const api = useApi();

  /**
   * 创建举报
   */
  const createReport = async (
    targetType: string,
    targetId: string,
    reason: string,
    description?: string
  ): Promise<CreateReportResponse> => {
    // 发送前验证参数
    const validTargetId = validateUserId(targetId); // 通用ID验证
    const validReason = validateContent(reason, 1, 50);
    const validDescription = description ? validateContent(description, 1, 500) : undefined;

    const request: CreateReportRequest = {
      target_type: targetType,
      target_id: validTargetId,
      reason: validReason,
      description: validDescription
    };

    const response = await api.post<ApiResponse<CreateReportResponse>>(
      buildReportUrl(REPORT_ENDPOINTS.PROTECTED.CREATE, false),
      request,
      { auth: true }
    );

    return response.data!;
  };

  /**
   * 举报用户
   */
  const reportUser = async (
    userId: string,
    reason: string,
    description?: string
  ): Promise<CreateReportResponse> => {
    return await createReport('user', userId, reason, description);
  };

  /**
   * 举报群组
   */
  const reportGroup = async (
    groupId: string,
    reason: string,
    description?: string
  ): Promise<CreateReportResponse> => {
    return await createReport('group', groupId, reason, description);
  };

  /**
   * 举报消息
   */
  const reportMessage = async (
    messageId: string,
    reason: string,
    description?: string
  ): Promise<CreateReportResponse> => {
    return await createReport('message', messageId, reason, description);
  };

  /**
   * 举报生活瞬间
   */
  const reportMoment = async (
    momentId: string,
    reason: string,
    description?: string
  ): Promise<CreateReportResponse> => {
    return await createReport('moment', momentId, reason, description);
  };

  /**
   * 举报评论
   */
  const reportComment = async (
    commentId: string,
    reason: string,
    description?: string
  ): Promise<CreateReportResponse> => {
    return await createReport('comment', commentId, reason, description);
  };

  /**
   * 获取我的举报记录
   */
  const getMyReports = async (
    cursor?: string,
    limit: number = 20
  ): Promise<PaginatedResponse<ReportResponse>> => {
    // 发送前验证参数
    const validLimit = validatePaginationParams(cursor, limit).limit;

    const request: GetMyReportsRequest = {
      cursor,
      limit: validLimit
    };

    const response = await api.post<ApiResponse<PaginatedResponse<ReportResponse>>>(
      buildReportUrl(REPORT_ENDPOINTS.PROTECTED.MY_REPORTS, false),
      request,
      { auth: true }
    );

    return response.data!;
  };

  /**
   * 获取举报列表（管理员功能）
   */
  const getReports = async (
    status?: string,
    targetType?: string,
    cursor?: string,
    limit: number = 20
  ): Promise<PaginatedResponse<ReportResponse>> => {
    // 发送前验证参数
    const validLimit = validatePaginationParams(cursor, limit).limit;

    const request: GetReportsRequest = {
      status,
      target_type: targetType,
      cursor,
      limit: validLimit
    };

    const response = await api.post<ApiResponse<PaginatedResponse<ReportResponse>>>(
      buildReportUrl(REPORT_ENDPOINTS.PROTECTED.LIST, false),
      request,
      { auth: true }
    );

    return response.data!;
  };

  /**
   * 处理举报（管理员功能）
   */
  const handleReport = async (
    reportId: string,
    action: string,
    note?: string
  ): Promise<HandleReportResponse> => {
    const validReportId = validateUserId(reportId); // 通用ID验证
    const validAction = validateContent(action, 1, 50);
    const validNote = note ? validateContent(note, 1, 500) : undefined;

    const request: HandleReportRequest = {
      report_id: validReportId,
      action: validAction,
      note: validNote
    };

    const response = await api.post<ApiResponse<HandleReportResponse>>(
      buildReportUrl(REPORT_ENDPOINTS.PROTECTED.HANDLE, false),
      request,
      { auth: true }
    );

    return response.data!;
  };

  /**
   * 拉黑用户
   */
  const blockUser = async (
    userId: string,
    reason?: string
  ): Promise<BlockUserResponse> => {
    const validUserId = validateUserId(userId);
    const validReason = reason ? validateContent(reason, 1, 200) : undefined;

    const request: BlockUserRequest = {
      user_id: validUserId,
      reason: validReason
    };

    const response = await api.post<ApiResponse<BlockUserResponse>>(
      buildUserUrl(USER_ENDPOINTS.PROTECTED.BLOCK_USER, false),
      request,
      { auth: true }
    );

    return response.data!;
  };

  /**
   * 获取举报原因的显示名称
   */
  const getReportReasonDisplayName = (reason: string): string => {
    const reasonMap: Record<string, string> = {
      'spam': '垃圾信息',
      'harassment': '骚扰行为',
      'hate_speech': '仇恨言论',
      'violence': '暴力内容',
      'nudity': '色情内容',
      'fake_info': '虚假信息',
      'copyright': '版权侵犯',
      'privacy': '隐私侵犯',
      'other': '其他原因'
    };
    return reasonMap[reason] || reason;
  };

  /**
   * 获取举报状态的显示名称
   */
  const getReportStatusDisplayName = (status: string): string => {
    const statusMap: Record<string, string> = {
      'pending': '待处理',
      'reviewing': '审核中',
      'resolved': '已解决',
      'rejected': '已拒绝',
      'closed': '已关闭'
    };
    return statusMap[status] || status;
  };

  /**
   * 获取举报状态的显示样式
   */
  const getReportStatusStyle = (status: string) => {
    const statusStyles: Record<string, { color: string; bgColor: string }> = {
      'pending': { color: '#ff9500', bgColor: '#fff5e6' },
      'reviewing': { color: '#007aff', bgColor: '#e6f3ff' },
      'resolved': { color: '#34c759', bgColor: '#e6f7ea' },
      'rejected': { color: '#ff3b30', bgColor: '#ffe6e6' },
      'closed': { color: '#8e8e93', bgColor: '#f2f2f7' }
    };
    return statusStyles[status] || statusStyles.pending;
  };

  /**
   * 获取举报目标类型的显示名称
   */
  const getReportTargetTypeDisplayName = (targetType: string): string => {
    const typeMap: Record<string, string> = {
      'user': '用户',
      'group': '群组',
      'message': '消息',
      'moment': '生活瞬间',
      'comment': '评论'
    };
    return typeMap[targetType] || targetType;
  };

  /**
   * 验证举报原因是否有效
   */
  const isValidReportReason = (reason: string): boolean => {
    const validReasons = [
      'spam', 'harassment', 'hate_speech', 'violence', 
      'nudity', 'fake_info', 'copyright', 'privacy', 'other'
    ];
    return validReasons.includes(reason);
  };

  /**
   * 验证举报目标类型是否有效
   */
  const isValidReportTargetType = (targetType: string): boolean => {
    const validTypes = ['user', 'group', 'message', 'moment', 'comment'];
    return validTypes.includes(targetType);
  };

  return {
    // 核心API方法
    createReport,
    getMyReports,
    getReports,
    handleReport,
    blockUser,
    
    // 便捷举报方法
    reportUser,
    reportGroup,
    reportMessage,
    reportMoment,
    reportComment,
    
    // 工具方法
    getReportReasonDisplayName,
    getReportStatusDisplayName,
    getReportStatusStyle,
    getReportTargetTypeDisplayName,
    isValidReportReason,
    isValidReportTargetType
  };
};

/**
 * 安全举报相关的常量定义
 */
export const SECURITY_CONSTANTS = {
  // 举报原因
  REPORT_REASONS: {
    SPAM: 'spam',
    HARASSMENT: 'harassment',
    HATE_SPEECH: 'hate_speech',
    VIOLENCE: 'violence',
    NUDITY: 'nudity',
    FAKE_INFO: 'fake_info',
    COPYRIGHT: 'copyright',
    PRIVACY: 'privacy',
    OTHER: 'other'
  },
  
  // 举报状态
  REPORT_STATUSES: {
    PENDING: 'pending',
    REVIEWING: 'reviewing',
    RESOLVED: 'resolved',
    REJECTED: 'rejected',
    CLOSED: 'closed'
  },
  
  // 举报目标类型
  REPORT_TARGET_TYPES: {
    USER: 'user',
    GROUP: 'group',
    MESSAGE: 'message',
    MOMENT: 'moment',
    COMMENT: 'comment'
  },
  
  // 处理动作
  HANDLE_ACTIONS: {
    APPROVE: 'approve',
    REJECT: 'reject',
    BLOCK_USER: 'block_user',
    DELETE_CONTENT: 'delete_content',
    WARNING: 'warning'
  },
  
  // 默认分页大小
  DEFAULT_PAGE_SIZE: 20,
  
  // 最大描述长度
  MAX_DESCRIPTION_LENGTH: 500
} as const;
