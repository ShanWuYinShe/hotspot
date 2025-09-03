/**
 * 用户设置相关 API 服务
 * 简化版本，只保留核心功能
 */

import { USER_ENDPOINTS, buildUserUrl } from "~/config/api";


// 临时错误创建函数
const createApiError = (params: {
  status?: number;
  code?: string;
  message: string;
  details?: any;
}) => {
  const error = new Error(params.message) as any;
  error.status = params.status || 400;
  error.code = params.code || "API_ERROR";
  error.details = params.details;
  return error;
};

export interface UserSettings {
  notifications: {
    push_enabled: boolean;
    email_enabled: boolean;
    sms_enabled: boolean;
    group_notifications: boolean;
    message_notifications: boolean;
    friend_requests: boolean;
    // 新增细粒度消息提示设置
    direct_message_toast: boolean; // 私聊消息Toast提示
    group_message_toast: boolean; // 群组消息Toast提示
    system_notification_toast: boolean; // 系统通知Toast提示
    error_notifications: boolean; // 错误提示开关
    success_notifications: boolean; // 成功提示开关
  };
  privacy: {
    profile_visibility: "public" | "friends" | "private";
    location_sharing: boolean;
    online_status: boolean;
    show_in_search: boolean;
  };
  preferences: {
    language: string;
    theme: "light" | "dark" | "auto";
    distance_unit: "km" | "miles";
    time_format: "12h" | "24h";
  };
  account: {
    nickname: string;
    bio?: string;
    email?: string;
    phone?: string;
    avatar_url?: string;
  };
}

export const useSettingsService = () => {
  const api = useApi();

  // 获取用户设置
  const getUserSettings = async (): Promise<UserSettings> => {
    try {
      const response = await api.post(
        buildUserUrl(USER_ENDPOINTS.GET_SETTINGS),
        {},
        { auth: true },
      );
      return response;
    } catch (e: any) {
      throw createApiError({
        status: e?.status || 400,
        code: e?.code || "SETTINGS_GET_FAILED",
        message: e?.message || "获取设置失败",
        details: e?.details,
      });
    }
  };

  // 更新用户设置（统一接口）
  const updateUserSettings = async (
    settings: Partial<UserSettings>,
  ): Promise<void> => {
    try {
      await api.post(buildUserUrl(USER_ENDPOINTS.UPDATE_SETTINGS), settings, {
        auth: true,
      });
    } catch (e: any) {
      throw createApiError({
        status: e?.status || 400,
        code: e?.code || "SETTINGS_UPDATE_FAILED",
        message: e?.message || "更新设置失败",
        details: e?.details,
      });
    }
  };

  return {
    getUserSettings,
    updateUserSettings,
  };
};
