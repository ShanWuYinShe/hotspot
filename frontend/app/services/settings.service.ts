/**
 * 设置服务 - Composition API 风格
 * 提供用户设置相关的所有 API 调用功能
 */

import { USER_ENDPOINTS, buildUserUrl } from "~/config/api";
import type { UserSettings } from "~/types/api";


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

// UserSettings 类型已从 types/index.ts 导入

export const useSettingsService = () => {
  const api = useApi();

  // 获取用户设置
  const getUserSettings = async (): Promise<UserSettings> => {
    // 直接使用useApi的统一错误处理，不再自己包装错误
    const response = await api.post(
      buildUserUrl(USER_ENDPOINTS.PROTECTED.GET_SETTINGS),
      {},
      { auth: true },
    );
    return response;
  };

  // 更新用户设置（统一接口）
  const updateUserSettings = async (
    settings: Partial<UserSettings>,
  ): Promise<void> => {
    // 直接使用useApi的统一错误处理，不再自己包装错误
    await api.post(buildUserUrl(USER_ENDPOINTS.PROTECTED.UPDATE_SETTINGS), settings, {
      auth: true,
    });
  };

  return {
    getUserSettings,
    updateUserSettings,
  };
};
