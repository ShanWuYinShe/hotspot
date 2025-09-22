/**
 * 通知服务 - Composition API 风格
 * 提供通知推送相关的所有 API 调用功能
 * 
 * 基于 backend_doc.md 5.2.8 通知推送API
 */

import { 
  NOTIFICATION_ENDPOINTS, 
  buildNotificationUrl 
} from "~/config/api";
import { 
  validateUserId,
  validateContent,
  validatePaginationParams 
} from "~/utils/api-validation";
import type { 
  ApiResponse,
  PaginatedResponse,
  SendNotificationRequest,
  SendNotificationResponse,
  GetNotificationsRequest,
  GetNotificationsResponse,
  MarkNotificationReadRequest,
  MarkNotificationReadResponse,
  GetUnreadNotificationCountRequest,
  GetUnreadNotificationCountResponse,
  GetNotificationSettingsRequest,
  GetNotificationSettingsResponse,
  UpdateNotificationSettingsRequest,
  UpdateNotificationSettingsResponse,
  NotificationResponse
} from "~/types/api";

export const useNotificationService = () => {
  const api = useApi();

  /**
   * 发送通知（管理员功能）
   */
  const sendNotification = async (
    userId: string,
    title: string,
    content: string,
    type: string = 'system',
    priority: string = 'normal',
    data?: Record<string, any>
  ): Promise<SendNotificationResponse> => {
    // 发送前验证参数
    const validUserId = validateUserId(userId);
    const validTitle = validateContent(title, 1, 100);
    const validContent = validateContent(content, 1, 500);

    const request: SendNotificationRequest = {
      user_id: validUserId,
      title: validTitle,
      content: validContent,
      notification_type: type,
      priority,
      data: data ? JSON.stringify(data) : undefined
    };

    const response = await api.post<ApiResponse<SendNotificationResponse>>(
      buildNotificationUrl(NOTIFICATION_ENDPOINTS.PROTECTED.SEND, false),
      request,
      { auth: true }
    );

    return response.data!;
  };

  /**
   * 获取通知列表
   */
  const getNotifications = async (
    status?: string,
    cursor?: string,
    limit: number = 20
  ): Promise<PaginatedResponse<NotificationResponse>> => {
    // 发送前验证参数
    const validLimit = validatePaginationParams(cursor, limit).limit;

    const request: GetNotificationsRequest = {
      status,
      cursor,
      limit: validLimit
    };

    const response = await api.post<ApiResponse<PaginatedResponse<NotificationResponse>>>(
      buildNotificationUrl(NOTIFICATION_ENDPOINTS.PROTECTED.LIST, false),
      request,
      { auth: true }
    );

    return response.data!;
  };

  /**
   * 标记通知为已读
   */
  const markNotificationRead = async (
    notificationIds: string[]
  ): Promise<MarkNotificationReadResponse> => {
    if (!notificationIds.length) {
      throw new Error('通知ID列表不能为空');
    }

    const request: MarkNotificationReadRequest = {
      notification_ids: notificationIds
    };

    const response = await api.post<ApiResponse<MarkNotificationReadResponse>>(
      buildNotificationUrl(NOTIFICATION_ENDPOINTS.PROTECTED.MARK_READ, false),
      request,
      { auth: true }
    );

    return response.data!;
  };

  /**
   * 标记单个通知为已读
   */
  const markSingleNotificationRead = async (
    notificationId: string
  ): Promise<MarkNotificationReadResponse> => {
    return await markNotificationRead([notificationId]);
  };

  /**
   * 获取未读通知数量
   */
  const getUnreadNotificationCount = async (): Promise<GetUnreadNotificationCountResponse> => {
    const request: GetUnreadNotificationCountRequest = {};

    const response = await api.post<ApiResponse<GetUnreadNotificationCountResponse>>(
      buildNotificationUrl(NOTIFICATION_ENDPOINTS.PROTECTED.UNREAD_COUNT, false),
      request,
      { auth: true }
    );

    return response.data!;
  };

  /**
   * 获取通知设置
   */
  const getNotificationSettings = async (): Promise<GetNotificationSettingsResponse> => {
    const request: GetNotificationSettingsRequest = {};

    const response = await api.post<ApiResponse<GetNotificationSettingsResponse>>(
      buildNotificationUrl(NOTIFICATION_ENDPOINTS.PROTECTED.SETTINGS, false),
      request,
      { auth: true }
    );

    return response.data!;
  };

  /**
   * 更新通知设置
   */
  const updateNotificationSettings = async (
    settings: {
      friend_request?: boolean;
      new_message?: boolean;
      group_invite?: boolean;
      moment_like?: boolean;
      moment_comment?: boolean;
      system_notification?: boolean;
      push_enabled?: boolean;
      email_enabled?: boolean;
      quiet_hours_start?: string;
      quiet_hours_end?: string;
    }
  ): Promise<UpdateNotificationSettingsResponse> => {
    const request: UpdateNotificationSettingsRequest = settings;

    const response = await api.post<ApiResponse<UpdateNotificationSettingsResponse>>(
      buildNotificationUrl(NOTIFICATION_ENDPOINTS.PROTECTED.UPDATE_SETTINGS, false),
      request,
      { auth: true }
    );

    return response.data!;
  };

  /**
   * 批量标记所有通知为已读
   */
  const markAllNotificationsRead = async (): Promise<MarkNotificationReadResponse> => {
    // 先获取所有未读通知
    const notifications = await getNotifications('unread');
    
    if (notifications.items.length === 0) {
      return {
        success: true,
        message: '没有未读通知'
      };
    }

    const notificationIds = notifications.items.map(n => n.notification_id);
    return await markNotificationRead(notificationIds);
  };

  /**
   * 获取通知类型的显示名称
   */
  const getNotificationTypeDisplayName = (type: string): string => {
    const typeMap: Record<string, string> = {
      'friend_request': '好友请求',
      'friend_accepted': '好友通过',
      'new_message': '新消息',
      'group_invite': '群组邀请',
      'moment_like': '瞬间点赞',
      'moment_comment': '瞬间评论',
      'system': '系统通知',
      'security': '安全通知'
    };
    return typeMap[type] || '未知通知';
  };

  /**
   * 获取通知优先级的显示样式
   */
  const getNotificationPriorityStyle = (priority: string) => {
    const priorityStyles: Record<string, { color: string; bgColor: string }> = {
      'low': { color: '#666666', bgColor: '#f5f5f5' },
      'normal': { color: '#333333', bgColor: '#ffffff' },
      'high': { color: '#ff6b35', bgColor: '#fff5f0' },
      'urgent': { color: '#ff0000', bgColor: '#ffe6e6' }
    };
    return priorityStyles[priority] || priorityStyles.normal;
  };

  return {
    // 核心API方法
    sendNotification,
    getNotifications,
    markNotificationRead,
    markSingleNotificationRead,
    getUnreadNotificationCount,
    getNotificationSettings,
    updateNotificationSettings,
    
    // 便捷方法
    markAllNotificationsRead,
    
    // 工具方法
    getNotificationTypeDisplayName,
    getNotificationPriorityStyle
  };
};

/**
 * 通知相关的常量定义
 */
export const NOTIFICATION_CONSTANTS = {
  // 通知类型
  TYPES: {
    FRIEND_REQUEST: 'friend_request',
    FRIEND_ACCEPTED: 'friend_accepted',
    NEW_MESSAGE: 'new_message',
    GROUP_INVITE: 'group_invite',
    MOMENT_LIKE: 'moment_like',
    MOMENT_COMMENT: 'moment_comment',
    SYSTEM: 'system',
    SECURITY: 'security'
  },
  
  // 通知优先级
  PRIORITIES: {
    LOW: 'low',
    NORMAL: 'normal',
    HIGH: 'high',
    URGENT: 'urgent'
  },
  
  // 通知状态
  STATUSES: {
    UNREAD: 'unread',
    READ: 'read'
  },
  
  // 默认分页大小
  DEFAULT_PAGE_SIZE: 20,
  
  // 最大批量操作数量
  MAX_BATCH_SIZE: 100
} as const;
