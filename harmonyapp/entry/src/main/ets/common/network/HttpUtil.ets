/**
 * 简化的HTTP工具类
 * 使用HarmonyOS原生http模块，去除冗余封装
 * 增强错误处理和日志记录
 */

import http from '@ohos.net.http';
import { BusinessCode, API_BASE_URL } from '../../constants/AppConstants';
import { ApiResponse, HealthCheckRequest, HealthCheckResponse } from '../types/api';
import { PreferencesUtil } from '../storage/PreferencesUtil';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF08;
const TAG: string = 'HttpUtil';

interface DebugInfo {
  baseUrl: string;
  hasToken: boolean;
  tokenPreview?: string;
}


export class HttpUtil {
  private static baseUrl = API_BASE_URL;

  /**
   * 设置访问令牌并持久化存储
   */
  static async setToken(token: string): Promise<void> {
    try {
      await PreferencesUtil.set('access_token', token);
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
      await PreferencesUtil.remove('access_token');
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
      return await PreferencesUtil.get<string>('access_token', '');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `获取访问令牌失败: ${error}`);
      return '';
    }
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
    data?: TRequest
  ): Promise<TResponse> {
    const fullUrl = `${HttpUtil.baseUrl}${url}`;
    const httpRequest = http.createHttp();
    const startTime = Date.now();

    // 记录请求开始
    hilog.info(DOMAIN_NUMBER, TAG, `🚀 HTTP请求开始: ${method} ${fullUrl}`);
    if (data) {
      hilog.debug(DOMAIN_NUMBER, TAG, `📤 请求数据: ${JSON.stringify(data)}`);
    }

    try {
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

      // 检查HTTP状态码
      if (response.responseCode !== 200) {
        const errorMsg = `HTTP错误: ${response.responseCode} - ${response.result}`;
        hilog.error(DOMAIN_NUMBER, TAG, `❌ ${errorMsg}`);
        throw new Error(errorMsg);
      }

      // 解析响应数据
      let result: ApiResponse<TResponse>;
      try {
        result = JSON.parse(response.result as string);
        hilog.debug(DOMAIN_NUMBER, TAG, `📥 响应数据: ${JSON.stringify(result)}`);
      } catch (parseError) {
        const errorMsg = `JSON解析失败: ${parseError}`;
        hilog.error(DOMAIN_NUMBER, TAG, `❌ ${errorMsg}`);
        hilog.error(DOMAIN_NUMBER, TAG, `原始响应: ${response.result}`);
        throw new Error(errorMsg);
      }

      // 检查业务状态码
      if (result.code !== BusinessCode.SUCCESS) {
        const errorMsg = `业务错误 [${result.code}]: ${result.msg}`;
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
        hilog.error(DOMAIN_NUMBER, TAG, `🌐 网络连接问题，请检查服务器地址: ${HttpUtil.baseUrl}`);
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
   * 获取当前配置信息（用于调试）
   */
  static async getDebugInfo(): Promise<DebugInfo> {
    const token = await HttpUtil.getToken();
    return {
      baseUrl: HttpUtil.baseUrl,
      hasToken: !!token,
      tokenPreview: token ? `${token.substring(0, 10)}...` : undefined
    };
  }

  /**
   * 测试服务器连接 - 直接使用健康检查API
   */
  static async testConnection(): Promise<ApiResponse<HealthCheckResponse>> {
    const request: HealthCheckRequest = {};
    return HttpUtil.request<ApiResponse<HealthCheckResponse>, HealthCheckRequest>('/health', http.RequestMethod.GET, request);
  }
}