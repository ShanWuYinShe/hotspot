/**
 * 简化的HTTP工具类
 * 使用HarmonyOS原生http模块，去除冗余封装
 * 增强错误处理和日志记录
 */

import http from '@ohos.net.http';
import request from '@ohos.request';
import common from '@ohos.app.ability.common';
import { API_BASE_URL, StorageKeys } from '../../constants/AppConstants';
import { ApiResponse, AppAuthData, HealthCheckData, UploadCategory, UploadMediaData } from '../types/api';
import { PreferencesUtil } from '../storage/PreferencesUtil';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF08;
const TAG: string = 'HttpUtil';

interface DebugInfo {
  baseUrl: string;
  hasToken: boolean;
  tokenPreview?: string;
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

  private static async getAuthData(): Promise<AppAuthData | null> {
    try {
      return await PreferencesUtil.getObject<AppAuthData>(StorageKeys.USER_INFO);
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
        const fullUrl = `${baseUrl}/api/v1/public/auth/refresh_token`;
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
        let payload: any;
        try {
          payload = JSON.parse(response.result as string);
        } catch {
          return false;
        } finally {
          httpRequest.destroy();
        }

        if (!payload || typeof payload.code !== 'number') return false;

        if (payload.code === 0 && payload.data) {
          const newAuth = payload.data as AppAuthData;
          await PreferencesUtil.set(StorageKeys.ACCESS_TOKEN, newAuth.access_token);
          await PreferencesUtil.setObject(StorageKeys.USER_INFO, newAuth);
          return true;
        }

        if (payload.code === 1101 || payload.code === 1106) {
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
      const isRecord = (v: unknown): v is Record<string, unknown> => typeof v === 'object' && v !== null;
      const isApiResponse = (v: unknown): v is ApiResponse<unknown> => {
        if (!isRecord(v)) return false;
        return typeof v.code === 'number' && typeof v.request_id === 'string';
      };

      let result: unknown;
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
        (response.responseCode === 401 || result.code === 1101);
      if (shouldTryRefresh) {
        const refreshed = await HttpUtil.ensureRefreshToken();
        if (refreshed) {
          return await HttpUtil.request<TResponse, TRequest>(url, method, data, true);
        }
      }

      // 检查业务状态码（业务成功与否以 code 判定）
      if (result.code !== 0) {
        const errorMsg = `业务错误 [${result.code}]: ${result.message || '未知错误'}`;
        hilog.error(DOMAIN_NUMBER, TAG, `❌ ${errorMsg}`);
        throw new Error(errorMsg);
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
   * 上传文件
   * 使用 @ohos.request 模块
   * 注意：fileUri 必须是 request 模块支持的 URI (如 internal://cache/...)
   * 调用者需负责确保文件已准备好并位于可访问位置
   */
  static async uploadFile(
    context: common.UIAbilityContext,
    fileUri: string,
    category: UploadCategory
  ): Promise<UploadMediaData> {
    const baseUrl = await HttpUtil.getBaseUrl();
    const fullUrl = `${baseUrl}/api/v1/protected/media/upload`;
    const token = await HttpUtil.getToken();

    interface UploadTaskState {
      // 不同系统版本字段名可能不同，保持可选 + 最小依赖
      responseCode?: number;
      message?: string;
      result?: string;
      data?: string;
      // eslint-disable-next-line @typescript-eslint/ban-types
      [key: string]: unknown;
    }

    const isRecord = (v: unknown): v is Record<string, unknown> =>
      typeof v === 'object' && v !== null;

    const extractText = (state: UploadTaskState): string | null => {
      const candidates = [state.result, state.message, state.data];
      for (const c of candidates) {
        if (typeof c === 'string' && c.trim().length > 0) return c;
      }
      // 兼容某些实现把响应放在 responseBody / response / body
      for (const key of ['responseBody', 'response', 'body']) {
        const v = state[key];
        if (typeof v === 'string' && v.trim().length > 0) return v;
      }
      return null;
    };

    const parseApiResponse = (text: string): UploadMediaData => {
      const parsed = JSON.parse(text) as ApiResponse<UploadMediaData>;
      if (!parsed || typeof parsed !== 'object') {
        throw new Error('上传响应解析失败');
      }
      if (parsed.code !== 0 || !parsed.data) {
        throw new Error(parsed.message || '上传失败');
      }
      return parsed.data;
    };

    return new Promise((resolve, reject) => {
      // 从 fileUri 中提取文件名
      let fileName = 'file';
      try {
          const parts = fileUri.split('/');
          if (parts.length > 0) {
              fileName = parts[parts.length - 1];
          }
      } catch (e) {}

      const uploadConfig: request.UploadConfig = {
        url: fullUrl,
        header: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'multipart/form-data'
        },
        method: 'POST',
        files: [
          { filename: fileName, name: 'file', uri: fileUri, type: 'application/octet-stream' }
        ],
        data: [
          { name: 'category', value: category }
        ]
      };

      try {
        request.uploadFile(context, uploadConfig).then((uploadTask) => {
          uploadTask.on('complete', (taskStates) => {
             // taskStates 的结构在不同版本上可能不同：兼容 array/object
             const state: UploadTaskState | null =
               Array.isArray(taskStates) ? (taskStates[0] as UploadTaskState) : (isRecord(taskStates) ? (taskStates as UploadTaskState) : null);

             if (!state) {
               reject(new Error('上传完成但未收到状态信息'));
               return;
             }

             if (typeof state.responseCode === 'number' && state.responseCode >= 400) {
               reject(new Error(`上传失败(${state.responseCode})`));
               return;
             }

             const text = extractText(state);
             if (!text) {
               reject(new Error('上传完成但未收到服务端响应'));
               return;
             }

             try {
               resolve(parseApiResponse(text));
             } catch (e) {
               reject(e instanceof Error ? e : new Error('上传响应解析失败'));
             }
          });
          
          uploadTask.on('fail', (taskStates) => {
             hilog.error(DOMAIN_NUMBER, TAG, `上传任务失败`);
             reject(new Error('上传失败'));
          });
        }).catch((err: Error) => {
          hilog.error(DOMAIN_NUMBER, TAG, `创建上传任务失败: ${err}`);
          reject(err);
        });
      } catch (err) {
        hilog.error(DOMAIN_NUMBER, TAG, `上传异常: ${err}`);
        reject(err);
      }
    });
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
    return HttpUtil.request<ApiResponse<HealthCheckData>, object>(
      '/api/v1/public/system/health', 
      http.RequestMethod.POST, 
      {}
    );
  }
}
