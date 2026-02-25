/**
 * HTTP 工具类
 *
 * 功能：
 * - 统一的请求/响应处理（ApiResponse 信封解析）
 * - 自动 Token 刷新（主动+被动两层机制）
 * - 幂等键自动注入
 * - 认证错误（10100）自动刷新并重试
 * - 账号封禁（10201）自动登出
 */

import http from '@ohos.net.http';
import request from '@ohos.request';
import common from '@ohos.app.ability.common';
import util from '@ohos.util';
import { API_BASE_URL, StorageKeys } from '../../constants/AppConstants';
import { ApiEndpoints } from '../config/ApiEndpoints';
import { ApiResponse, AuthData, HealthStatus, MediaCategory, UploadedMedia } from '../types/api';
import { PreferencesUtil } from '../storage/PreferencesUtil';
import { hilog } from '@kit.PerformanceAnalysisKit';
import {
  extractApiErrorSnapshot,
  guessUserMessageFromSnapshot,
  isAuthErrorCode,
  isAccountBannedCode
} from './ApiErrorUtil';

/**
 * Spec MUST 幂等端点列表
 * 这些端点的每次请求必须携带 Idempotency-Key，缺少则会被后端拒绝
 */
const MUST_IDEMPOTENT_ENDPOINTS: string[] = [
  // 认证
  ApiEndpoints.AUTH.REGISTER,
  ApiEndpoints.AUTH.GUEST,
  ApiEndpoints.AUTH.REQUEST_PASSWORD_RESET,
  ApiEndpoints.AUTH.RESET_PASSWORD,
  ApiEndpoints.AUTH.ME.CHANGE_PASSWORD,
  ApiEndpoints.USER.ME.EMAIL_REQUEST_CHANGE,
  ApiEndpoints.USER.ME.EMAIL_CONFIRM_CHANGE,
  ApiEndpoints.USER.ME.PHONE_REQUEST_CHANGE,
  ApiEndpoints.USER.ME.PHONE_CONFIRM_CHANGE,
  // 消息
  ApiEndpoints.CHAT.PRIVATE_MESSAGES.SEND,
  ApiEndpoints.CHAT.GROUP_MESSAGES.SEND,
  // 创建类
  ApiEndpoints.MOMENT.CREATE,
  ApiEndpoints.COMMENT.CREATE,
  ApiEndpoints.REPORT.CREATE,
  ApiEndpoints.SHARE.CREATE,
  // 群组
  ApiEndpoints.GROUP.CREATE,
  ApiEndpoints.GROUP.ME.JOIN,
  ApiEndpoints.GROUP.MEMBERS.INVITE,
  ApiEndpoints.GROUP.MEMBERS.BAN,
  ApiEndpoints.GROUP.ANNOUNCEMENTS.CREATE,
  ApiEndpoints.GROUP.APPLICATIONS.APPROVE,
  ApiEndpoints.GROUP.APPLICATIONS.REJECT,
  ApiEndpoints.GROUP.ME.INVITES.ACCEPT,
  ApiEndpoints.GROUP.ME.INVITES.REJECT,
  // 上传
  ApiEndpoints.UPLOAD.COMPLETE_UPLOAD,
];

const DOMAIN_NUMBER: number = 0xFF08;
const TAG: string = 'HttpUtil';

interface DebugInfo {
  baseUrl: string;
  hasToken: boolean;
  tokenPreview?: string;
}

function isRecord(v: Object): boolean {
  return v !== null && typeof v === 'object';
}

function isApiResponse(v: Object): boolean {
  if (!isRecord(v)) return false;
  const record = v as Record<string, Object>;
  return typeof record.code === 'number';
}


export class HttpUtil {
  /**
   * 全局刷新锁：避免并发 refresh_token 触发"轮换冲突"
   * Spec 建议：同一时刻只允许一个刷新请求在飞
   */
  private static refreshPromise: Promise<boolean> | null = null;

  private static async getBaseUrl(): Promise<string> {
    try {
      const url = await PreferencesUtil.get<string>(StorageKeys.SERVER_BASE_URL, API_BASE_URL);
      return typeof url === 'string' && url.trim().length > 0 ? url.trim() : API_BASE_URL;
    } catch {
      return API_BASE_URL;
    }
  }

  /**
   * 设置访问令牌并持久化存储
   */
  static async setToken(token: string): Promise<void> {
    try {
      await PreferencesUtil.set(StorageKeys.ACCESS_TOKEN, token);
      hilog.info(DOMAIN_NUMBER, TAG, '访问令牌已保存');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `保存访问令牌失败: ${error}`);
    }
  }

  /**
   * 清除访问令牌
   */
  static async clearToken(): Promise<void> {
    try {
      await PreferencesUtil.remove(StorageKeys.ACCESS_TOKEN);
      hilog.info(DOMAIN_NUMBER, TAG, '访问令牌已清除');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `清除访问令牌失败: ${error}`);
    }
  }

  /**
   * 获取访问令牌
   */
  static async getToken(): Promise<string> {
    try {
      return await PreferencesUtil.get<string>(StorageKeys.ACCESS_TOKEN, '');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `获取访问令牌失败: ${error}`);
      return '';
    }
  }

  private static async getAuthData(): Promise<AuthData | null> {
    try {
      return await PreferencesUtil.getObject<AuthData>(StorageKeys.USER_INFO);
    } catch {
      return null;
    }
  }

  private static async clearAuthData(): Promise<void> {
    try {
      await PreferencesUtil.remove(StorageKeys.ACCESS_TOKEN);
      await PreferencesUtil.remove(StorageKeys.USER_INFO);
    } catch {
      // ignore
    }
    // 同步重置全局 UI 状态，让 @StorageLink 绑定的组件立即切换到未登录视图
    try {
      AppStorage.setOrCreate('currentUser', null);
      AppStorage.setOrCreate('isLoggedIn', false);
      AppStorage.setOrCreate('userType', '');
    } catch {
      // ignore
    }
    hilog.info(DOMAIN_NUMBER, TAG, '认证数据已清除，UI 状态已重置为未登录');
  }

  /**
   * 刷新 token（App 场景：refresh_token 放在 body）
   *
   * Spec 规则：
   * - refresh_token 端点是 token 可选的，即使 access_token 已过期也可调用
   * - App 端必须在 body 中传 refresh_token，不传 access_token
   * - 并发刷新通过 refreshPromise 锁避免竞态
   */
  private static async ensureRefreshToken(): Promise<boolean> {
    if (HttpUtil.refreshPromise) return HttpUtil.refreshPromise;

    HttpUtil.refreshPromise = (async () => {
      try {
        const authData = await HttpUtil.getAuthData();
        const refreshToken = authData?.refresh_token;
        if (!refreshToken) {
          hilog.warn(DOMAIN_NUMBER, TAG, '无 refresh_token，无法刷新');
          return false;
        }

        const baseUrl = await HttpUtil.getBaseUrl();
        const fullUrl = `${baseUrl}${ApiEndpoints.AUTH.REFRESH_TOKEN}`;
        const httpRequest = http.createHttp();

        // 刷新请求不携带 Authorization header（access_token 可能已过期）
        const headers: Record<string, string> = { 'Content-Type': 'application/json' };
        const requestOptions: http.HttpRequestOptions = {
          method: http.RequestMethod.POST,
          header: headers,
          extraData: JSON.stringify({ refresh_token: refreshToken, client_type: 'app' }),
          connectTimeout: 15000,
          readTimeout: 15000
        };

        const response = await httpRequest.request(fullUrl, requestOptions);
        let payload: Object;
        try {
          payload = JSON.parse(response.result as string) as Object;
        } catch {
          return false;
        } finally {
          httpRequest.destroy();
        }

        const payloadRecord = payload as Record<string, Object>;
        if (!payloadRecord || typeof payloadRecord.code !== 'number') return false;

        if (payloadRecord.code === 0 && payloadRecord.data) {
          const newAuth = payloadRecord.data as AuthData;
          await PreferencesUtil.set(StorageKeys.ACCESS_TOKEN, newAuth.access_token);
          await PreferencesUtil.setObject(StorageKeys.USER_INFO, newAuth);
          hilog.info(DOMAIN_NUMBER, TAG, 'Token 刷新成功');
          return true;
        }

        // 刷新失败（10100=refresh_token 无效、10201=账号被封禁等），清除认证状态
        hilog.warn(DOMAIN_NUMBER, TAG, `Token 刷新失败，code=${payloadRecord.code}`);
        await HttpUtil.clearAuthData();
        return false;
      } catch (error) {
        hilog.warn(DOMAIN_NUMBER, TAG, `refresh_token failed: ${JSON.stringify(error)}`);
        return false;
      } finally {
        HttpUtil.refreshPromise = null;
      }
    })();

    return HttpUtil.refreshPromise;
  }

  /**
   * 主动刷新：检查 access_token 是否即将过期，提前刷新
   * 在发起请求前调用，避免请求失败后再刷新导致额外延迟
   */
  private static async proactiveRefreshIfNeeded(): Promise<void> {
    try {
      const authData = await HttpUtil.getAuthData();
      if (!authData?.access_token_expires_at || !authData?.refresh_token) return;

      const expiresAt = new Date(authData.access_token_expires_at).getTime();
      const now = Date.now();
      const thresholdMs = 3 * 60 * 1000; // 提前 3 分钟刷新

      if (expiresAt - now < thresholdMs && expiresAt - now > 0) {
        hilog.info(DOMAIN_NUMBER, TAG, `Access token 将在 ${Math.round((expiresAt - now) / 1000)}s 后过期，主动刷新`);
        await HttpUtil.ensureRefreshToken();
      }
    } catch {
      // 主动刷新失败不影响正常请求，被动刷新会兜底
    }
  }

  private static async buildHeaders(url?: string): Promise<Record<string, string>> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json'
    };

    try {
      const token = await HttpUtil.getToken();
      if (token) {
        headers['Authorization'] = `Bearer ${token}`;
      }
    } catch (error) {
      hilog.warn(DOMAIN_NUMBER, TAG, `获取令牌失败，使用无认证请求: ${error}`);
    }

    // 自动为 MUST 幂等端点注入 Idempotency-Key
    if (url && MUST_IDEMPOTENT_ENDPOINTS.includes(url)) {
      headers['Idempotency-Key'] = util.generateRandomUUID(false);
    }

    return headers;
  }

  /**
   * 通用请求方法 - 支持强类型请求体
   *
   * Token 刷新流程（双层机制）：
   * 1. 主动刷新：请求发出前检查 access_token_expires_at，即将过期则提前刷新
   * 2. 被动刷新：收到 10100 时自动刷新 token 并重试一次
   */
  static async request<TResponse, TRequest>(
    url: string,
    method: http.RequestMethod = http.RequestMethod.POST,
    data?: TRequest,
    retried: boolean = false
  ): Promise<TResponse> {
    const baseUrl = await HttpUtil.getBaseUrl();
    const fullUrl = `${baseUrl}${url}`;
    const httpRequest = http.createHttp();
    const startTime = Date.now();

    hilog.info(DOMAIN_NUMBER, TAG, `HTTP请求: ${method} ${fullUrl}`);

    try {
      const tokenForRetryCheck = await HttpUtil.getToken();
      const hasAuthHeader = !!tokenForRetryCheck;

      // 主动刷新：如果 token 即将过期，先刷新再发请求
      if (hasAuthHeader && !retried) {
        await HttpUtil.proactiveRefreshIfNeeded();
      }

      const headers = await HttpUtil.buildHeaders(url);
      const requestOptions: http.HttpRequestOptions = {
        method,
        header: headers,
        extraData: data ? JSON.stringify(data) : undefined,
        connectTimeout: 30000,
        readTimeout: 30000
      };

      const response = await httpRequest.request(fullUrl, requestOptions);
      const responseTime = Date.now() - startTime;
      hilog.info(DOMAIN_NUMBER, TAG, `HTTP响应: ${response.responseCode} (${responseTime}ms)`);

      let result: Object;
      try {
        result = JSON.parse(response.result as string);
        hilog.debug(DOMAIN_NUMBER, TAG, `响应数据: ${JSON.stringify(result)}`);
      } catch (parseError) {
        throw new Error(`JSON解析失败: ${parseError}`);
      }

      // 业务成功与否仅以 ApiResponse.code 判定（后端统一返回 HTTP 200）
      if (!isApiResponse(result)) {
        throw new Error('响应格式不符合 ApiResponse 约定');
      }

      const resultRecord = result as Record<string, Object>;
      const resultCode = resultRecord.code as number;

      // 10201：账号被封禁 -> 强制登出，不尝试刷新
      if (isAccountBannedCode(resultCode)) {
        await HttpUtil.clearAuthData();
        const err = new Error('账号已被封禁');
        (err as Object as Record<string, Object>).code = resultCode;
        throw err;
      }

      // 10100：认证失败 -> 首次尝试时自动刷新 token 并重试
      if (!retried && hasAuthHeader && isAuthErrorCode(resultCode)) {
        hilog.info(DOMAIN_NUMBER, TAG, '收到 10100 认证失败，尝试刷新 token');
        const refreshed = await HttpUtil.ensureRefreshToken();
        if (refreshed) {
          return await HttpUtil.request<TResponse, TRequest>(url, method, data, true);
        }
        // 刷新失败：ensureRefreshToken 内部已清除认证状态
        hilog.warn(DOMAIN_NUMBER, TAG, '令牌刷新失败，已强制登出');
      }

      // 其他业务错误
      if (resultCode !== 0) {
        const snapshot = extractApiErrorSnapshot(result, response.responseCode);
        const userMessage = guessUserMessageFromSnapshot(snapshot, '操作失败');
        hilog.error(DOMAIN_NUMBER, TAG, `业务错误 [${snapshot.code}]: ${userMessage}`);

        const err = new Error(userMessage);
        const errRecord = err as Object as Record<string, Object>;
        if (snapshot.code !== undefined) {
          errRecord.code = snapshot.code;
        }
        if (snapshot.status !== undefined) {
          errRecord.status = snapshot.status;
        }
        throw err;
      }

      hilog.info(DOMAIN_NUMBER, TAG, `请求成功: ${method} ${url} (${responseTime}ms)`);
      return result as TResponse;

    } catch (error) {
      const responseTime = Date.now() - startTime;
      const errorMsg = error instanceof Error ? error.message : '未知错误';
      hilog.error(DOMAIN_NUMBER, TAG, `请求失败: ${method} ${url} (${responseTime}ms) ${errorMsg}`);

      if (errorMsg.includes('Network') || errorMsg.includes('timeout') || errorMsg.includes('connect')) {
        const baseUrl2 = await HttpUtil.getBaseUrl();
        hilog.error(DOMAIN_NUMBER, TAG, `网络连接问题，请检查服务器地址: ${baseUrl2}`);
      }

      throw new Error(errorMsg);
    } finally {
      httpRequest.destroy();
    }
  }

  // POST 方法 - 支持强类型请求体
  static post<TResponse, TRequest>(url: string, data: TRequest): Promise<TResponse> {
    return HttpUtil.request<TResponse, TRequest>(url, http.RequestMethod.POST, data);
  }

  /**
   * 上传二进制数据到指定 URL (PUT)
   * 用于 S3 Presigned URL 上传
   */
  static async uploadToUrl(url: string, data: ArrayBuffer, headers: Record<string, string>): Promise<void> {
    const httpRequest = http.createHttp();
    try {
      hilog.info(DOMAIN_NUMBER, TAG, `开始上传文件到: ${url}`);
      const response = await httpRequest.request(url, {
        method: http.RequestMethod.PUT,
        header: headers,
        extraData: data,
        connectTimeout: 60000,
        readTimeout: 60000
      });

      hilog.info(DOMAIN_NUMBER, TAG, `上传响应: ${response.responseCode}`);

      if (response.responseCode < 200 || response.responseCode >= 300) {
        throw new Error(`Upload failed with status ${response.responseCode}`);
      }
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : '未知错误';
      hilog.error(DOMAIN_NUMBER, TAG, `上传失败: ${errorMsg}`);
      throw new Error(errorMsg);
    } finally {
      httpRequest.destroy();
    }
  }

  /**
   * 获取当前配置信息（用于调试）
   */
  static async getDebugInfo(): Promise<DebugInfo> {
    const token = await HttpUtil.getToken();
    const baseUrl = await HttpUtil.getBaseUrl();
    return {
      baseUrl,
      hasToken: !!token,
      tokenPreview: token ? `${token.substring(0, 10)}...` : undefined
    };
  }

  /**
   * 测试服务器连接 - 使用系统健康检查API
   */
  static async testConnection(): Promise<ApiResponse<HealthStatus>> {
    const emptyRequest: Record<string, Object> = {};
    return HttpUtil.request<ApiResponse<HealthStatus>, Record<string, Object>>(
      ApiEndpoints.SYSTEM.HEALTH,
      http.RequestMethod.POST,
      emptyRequest
    );
  }
}
