/**
 * 简化的HTTP工具类
 * 使用HarmonyOS原生http模块，去除冗余封装
 * 增强错误处理和日志记录
 */

import http from '@ohos.net.http';
import request from '@ohos.request';
import common from '@ohos.app.ability.common';
import { API_BASE_URL, StorageKeys } from '../../constants/AppConstants';
import { ApiEndpoints } from '../config/ApiEndpoints';
import { ApiResponse, AuthData, HealthCheckData, UploadCategory, UploadMediaData } from '../types/api';
import { PreferencesUtil } from '../storage/PreferencesUtil';
import { hilog } from '@kit.PerformanceAnalysisKit';
import {
  extractApiErrorSnapshot,
  guessUserMessageFromSnapshot,
  shouldForceLogoutOnAuthError,
  isAuthErrorCode
} from './ApiErrorUtil';

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
  return typeof record.code === 'number' && typeof record.request_id === 'string';
}


// Removed UploadResult interface as it is replaced by UploadMediaData from types/api.ets


export class HttpUtil {
  /**
   * 全局刷新锁：避免并发 refresh_token 触发“轮换冲突”
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
  }

  /**
   * 刷新 token（App 场景：refresh_token 放在 body）
   */
  private static async ensureRefreshToken(): Promise<boolean> {
    if (HttpUtil.refreshPromise) return HttpUtil.refreshPromise;

    HttpUtil.refreshPromise = (async () => {
      try {
        const authData = await HttpUtil.getAuthData();
        const refreshToken = authData?.refresh_token;
        if (!refreshToken) return false;

        const baseUrl = await HttpUtil.getBaseUrl();
        const fullUrl = `${baseUrl}${ApiEndpoints.AUTH.REFRESH_TOKEN}`;
        const httpRequest = http.createHttp();

        const headers: Record<string, string> = { 'Content-Type': 'application/json' };
        const requestOptions: http.HttpRequestOptions = {
          method: http.RequestMethod.POST,
          header: headers,
          extraData: JSON.stringify({ refresh_token: refreshToken, client_type: 'app' }),
          connectTimeout: 30000,
          readTimeout: 30000
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
          return true;
        }

        if (shouldForceLogoutOnAuthError(payloadRecord.code, payloadRecord.message)) {
          await HttpUtil.clearAuthData();
        }

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

  private static async buildHeaders(): Promise<Record<string, string>> {
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

    return headers;
  }

  /**
   * 通用请求方法 - 支持强类型请求体
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

    // 记录请求开始
    hilog.info(DOMAIN_NUMBER, TAG, `🚀 HTTP请求开始: ${method} ${fullUrl}`);
    if (data) {
      hilog.debug(DOMAIN_NUMBER, TAG, `📤 请求数据: ${JSON.stringify(data)}`);
    }

    try {
      const tokenForRetryCheck = await HttpUtil.getToken();
      const hasAuthHeader = !!tokenForRetryCheck;

      const headers = await HttpUtil.buildHeaders();
      const requestOptions: http.HttpRequestOptions = {
        method,
        header: headers,
        extraData: data ? JSON.stringify(data) : undefined,
        connectTimeout: 30000, // 30秒连接超时
        readTimeout: 30000     // 30秒读取超时
      };

      hilog.debug(DOMAIN_NUMBER, TAG, `📋 请求头: ${JSON.stringify(requestOptions.header)}`);

      const response = await httpRequest.request(fullUrl, requestOptions);
      const responseTime = Date.now() - startTime;

      // 记录响应状态
      hilog.info(DOMAIN_NUMBER, TAG, `📡 HTTP响应: ${response.responseCode} (${responseTime}ms)`);

      // 解析响应数据
      let result: Object;
      try {
        result = JSON.parse(response.result as string);
        hilog.debug(DOMAIN_NUMBER, TAG, `📥 响应数据: ${JSON.stringify(result)}`);
      } catch (parseError) {
        const errorMsg = `JSON解析失败: ${parseError}`;
        hilog.error(DOMAIN_NUMBER, TAG, `❌ ${errorMsg}`);
        hilog.error(DOMAIN_NUMBER, TAG, `原始响应: ${response.result}`);
        throw new Error(errorMsg);
      }

      // 注意：后端使用真实 HTTP 状态码表达“错误大类”，客户端不应仅因非 200 就失败。
      // 业务成功与否仍以 ApiResponse.code 判定。

      if (!isApiResponse(result)) {
        const errorMsg = '响应格式不符合 ApiResponse 约定';
        hilog.error(DOMAIN_NUMBER, TAG, `❌ ${errorMsg}`);
        throw new Error(errorMsg);
      }

      // 仅当“原请求带 Authorization”且第一次尝试失败时，才尝试 refresh
      const shouldTryRefresh =
        !retried &&
        hasAuthHeader &&
        (response.responseCode === 401 || (() => {
          const resultRecord = result as Record<string, Object>;
          const resultCode = resultRecord.code;
          return resultCode !== undefined && isAuthErrorCode(resultCode);
        })());
      if (shouldTryRefresh) {
        const refreshed = await HttpUtil.ensureRefreshToken();
        if (refreshed) {
          return await HttpUtil.request<TResponse, TRequest>(url, method, data, true);
        }
      }

      // 检查业务状态码（业务成功与否以 code 判定）
      const resultRecord = result as Record<string, Object>;
      if (resultRecord.code !== 0) {
        const snapshot = extractApiErrorSnapshot(result, response.responseCode);
        const userMessage = guessUserMessageFromSnapshot(snapshot, '操作失败');
        hilog.error(DOMAIN_NUMBER, TAG, `❌ 业务错误 [${snapshot.code}]: ${userMessage}`);

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

      hilog.info(DOMAIN_NUMBER, TAG, `✅ 请求成功: ${method} ${url} (${responseTime}ms)`);
      return result as TResponse;

    } catch (error) {
      const responseTime = Date.now() - startTime;
      const errorMsg = error instanceof Error ? error.message : '未知错误';

      hilog.error(DOMAIN_NUMBER, TAG, `❌ 请求失败: ${method} ${url} (${responseTime}ms)`);
      hilog.error(DOMAIN_NUMBER, TAG, `错误详情: ${errorMsg}`);

      // 网络连接错误的特殊处理
      if (errorMsg.includes('Network') || errorMsg.includes('timeout') || errorMsg.includes('connect')) {
        const baseUrl = await HttpUtil.getBaseUrl();
        hilog.error(DOMAIN_NUMBER, TAG, `🌐 网络连接问题，请检查服务器地址: ${baseUrl}`);
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
      hilog.info(DOMAIN_NUMBER, TAG, `🚀 开始上传文件到: ${url}`);
      const response = await httpRequest.request(url, {
        method: http.RequestMethod.PUT,
        header: headers,
        extraData: data,
        connectTimeout: 60000, // 60秒连接超时
        readTimeout: 60000     // 60秒读取超时
      });

      hilog.info(DOMAIN_NUMBER, TAG, `📡 上传响应: ${response.responseCode}`);

      if (response.responseCode < 200 || response.responseCode >= 300) {
        throw new Error(`Upload failed with status ${response.responseCode}`);
      }
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : '未知错误';
      hilog.error(DOMAIN_NUMBER, TAG, `❌ 上传失败: ${errorMsg}`);
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
  static async testConnection(): Promise<ApiResponse<HealthCheckData>> {
    const emptyRequest: Record<string, Object> = {};
    return HttpUtil.request<ApiResponse<HealthCheckData>, Record<string, Object>>(
      ApiEndpoints.SYSTEM.HEALTH,
      http.RequestMethod.POST,
      emptyRequest
    );
  }
}
