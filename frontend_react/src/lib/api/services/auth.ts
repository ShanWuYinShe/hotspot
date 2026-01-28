/**
 * 认证服务
 * 处理登录、注册、登出等认证相关操作
 * 
 * 基于 @api_spec_auth.mdc 规范
 */

import { apiRequest, apiRequestPublic, clearTokens } from '@/lib/api/client';
import { ENDPOINTS } from '@/lib/api/endpoints';
import type { ApiResponse, AuthData, MyCard, LoginRequest, RegisterRequest, ClientType } from '@/types/api';
import { useAuthStore, setAccessToken, setRefreshToken, getRefreshToken } from '@/stores/auth';

// Web 端固定 client_type
const CLIENT_TYPE: ClientType = 'web';

/**
 * 登录
 * @api POST /api/v1/auth/login
 */
export async function login(loginId: string, password: string): Promise<ApiResponse<AuthData>> {
  const response = await apiRequestPublic<AuthData>(ENDPOINTS.AUTH.LOGIN, {
    login_id: loginId,
    password,
    client_type: CLIENT_TYPE,
  } as LoginRequest);

  if (response.code === 0 && response.data) {
    useAuthStore.getState().setAuthData(response.data);
    // 获取用户详情
    await fetchCurrentUser();
  }

  return response;
}

/**
 * 注册
 * @api POST /api/v1/auth/register
 */
export async function register(
  loginId: string,
  password: string,
  nickname: string
): Promise<ApiResponse<AuthData>> {
  const response = await apiRequestPublic<AuthData>(ENDPOINTS.AUTH.REGISTER, {
    login_id: loginId,
    password,
    nickname,
    client_type: CLIENT_TYPE,
  } as RegisterRequest);

  if (response.code === 0 && response.data) {
    useAuthStore.getState().setAuthData(response.data);
    // 获取用户详情
    await fetchCurrentUser();
  }

  return response;
}

/**
 * 访客注册（临时用户）
 * 访客是一种临时用户，拥有登录 token，但只能通过此接口获取
 * @api POST /api/v1/auth/guest
 */
export async function registerGuest(nickname: string): Promise<ApiResponse<AuthData>> {
  const response = await apiRequestPublic<AuthData>(ENDPOINTS.AUTH.GUEST, {
    nickname,
    client_type: CLIENT_TYPE,
  });

  if (response.code === 0 && response.data) {
    useAuthStore.getState().setAuthData(response.data);
    // 获取用户详情
    await fetchCurrentUser();
  }

  return response;
}

/**
 * 登出
 * @api POST /api/v1/auth/logout
 */
export async function logout(): Promise<void> {
  try {
    await apiRequest(ENDPOINTS.AUTH.LOGOUT, {});
  } catch {
    // 忽略登出 API 错误
  } finally {
    clearTokens();
    useAuthStore.getState().logout();
  }
}

/**
 * 刷新 Token
 * Web 端：refresh_token 通过 HttpOnly Cookie 传输，请求体不携带
 * @api POST /api/v1/auth/refresh_token
 */
export async function refreshToken(): Promise<ApiResponse<AuthData>> {
  // Web 端：不需要在请求体中携带 refresh_token（通过 Cookie）
  // 但我们使用 localStorage，所以需要携带
  const currentRefreshToken = getRefreshToken();
  
  const response = await apiRequestPublic<AuthData>(ENDPOINTS.AUTH.REFRESH_TOKEN, {
    client_type: CLIENT_TYPE,
    refresh_token: currentRefreshToken || undefined,
  });

  if (response.code === 0 && response.data) {
    setAccessToken(response.data.access_token);
    if (response.data.refresh_token) {
      setRefreshToken(response.data.refresh_token);
    }
    // 更新过期时间
    useAuthStore.getState().setAuthData(response.data);
  }

  return response;
}

/**
 * 检查认证状态
 * Token 可选；未登录时返回 is_valid: false
 * @api POST /api/v1/auth/check_status
 */
export async function checkAuthStatus(): Promise<ApiResponse<{ is_valid: boolean; user_id?: number; nickname?: string; expires_at?: string }>> {
  return apiRequestPublic<{ is_valid: boolean; user_id?: number; nickname?: string; expires_at?: string }>(
    ENDPOINTS.AUTH.CHECK_STATUS,
    {}
  );
}

/**
 * 获取当前用户信息
 * @api POST /api/v1/user/me/profile
 */
export async function fetchCurrentUser(): Promise<ApiResponse<MyCard>> {
  const response = await apiRequest<MyCard>(ENDPOINTS.USER.ME.PROFILE, {});

  if (response.code === 0 && response.data) {
    useAuthStore.getState().setUser(response.data);
  }

  return response;
}

/**
 * 修改密码
 * @api POST /api/v1/auth/me/change_password
 */
export async function changePassword(
  oldPassword: string,
  newPassword: string
): Promise<ApiResponse<void>> {
  return apiRequest<void>(ENDPOINTS.AUTH.CHANGE_PASSWORD, {
    old_password: oldPassword,
    new_password: newPassword,
  });
}

/**
 * 初始化认证状态
 * 在应用启动时调用
 */
export async function initAuth(): Promise<void> {
  const store = useAuthStore.getState();

  try {
    if (store.checkTokenValid()) {
      await fetchCurrentUser();
    } else {
      store.logout();
    }
  } catch {
    store.logout();
  }
}
