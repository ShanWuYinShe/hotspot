/**
 * API 客户端 Composable
 * 封装统一的 HTTP 请求逻辑
 */

import { useSystemStatus } from "./useSystemStatus";
import { processApiResponse, processApiRequest } from "~/utils/bigint-json";

export const useApi = () => {
  // 预加载依赖，避免动态导入的性能开销
  const systemStatus = useSystemStatus();
  const { handleError: handleAppError } = useErrorHandler();
  const config = useRuntimeConfig();

  // 统一处理服务端包裹响应：当 code!=0 时抛出标准错误对象
  const processEnvelope = (resp: unknown, meta: { url: string; method: string }) => {
    // 仅当响应形如 { code, msg, data } 时启用包裹处理
    if (resp && typeof resp === "object" && "code" in resp) {
      const code = (resp as any).code;
      if (code === 0 && "data" in resp) {
        return (resp as any).data;
      }

      // code 非 0：按统一规范抛错（status:200，data: 原始包裹）
      const message = (resp as any).msg || (resp as any).message || "请求失败";
      const error: any = new Error(message);
      error.status = 200;
      error.code = code;
      error.data = resp;
      error.endpoint = meta.url;
      error.method = meta.method;
      throw error;
    }

    // 非包裹响应（尽量保持兼容）
    return resp;
  };

  // 获取认证 token
  const getAuthHeaders = () => {
    const authStore = useAuthStore();
    const t = (authStore as any)?.token;
    const token = typeof t === "string" ? t : t?.value;
    return token
      ? {
          Authorization: `Bearer ${token}`,
        }
      : {};
  };

  // 错误处理
  const handleError = async (
    error: any,
    endpoint?: string,
    method?: string,
  ) => {

    // 增强错误信息
    if (error?.status) {
      error.endpoint = endpoint;
      error.method = method;
    }

    // 统一为错误补充 message（优先从后端 data.msg / data.message 读取）
    if (!error?.message) {
      const msg = error?.data?.msg || error?.data?.message;
      if (msg) error.message = msg;
    }

    // 更新系统状态
    if (
      error?.status >= 500 ||
      error?.status === 0 ||
      error?.name === "TypeError"
    ) {
      systemStatus.updateApiStatus("error");
    } else if (error?.status >= 400) {
      systemStatus.updateApiStatus("warning");
    } else if (error?.status === 200 && error?.data && error?.data?.code !== 0) {
      // 业务层错误但 HTTP 200：标记为 warning
      systemStatus.updateApiStatus("warning");
    }

    // 使用统一错误处理器
    await handleAppError(error, 'useApi', 'handleError');

    // 仅在明确未授权时清除认证，避免误登出（交由调用层或全局中间件处理跳转）
    try {
      const authStore = useAuthStore();
      if (error?.status === 401) {
        // 不在这里清除，以便调用层可进行提示或补救；保持最小侵入
        // 可在中间件/页面层按需处理
      }
    } catch (_) {}

    throw error;
  };

  // 预校验是否需要登录
  const precheckAuth = async (authRequired?: boolean) => {
    if (!authRequired) return;
    const authStore = useAuthStore();
    if (!(authStore.token || authStore.refreshToken)) {
      const err: any = new Error("需要登录后才能进行该操作");
      err.status = 401;
      err.code = "AUTH_REQUIRED";
      throw err;
    }
  };

  // 尝试刷新一次
  const tryRefreshOnce = async (): Promise<boolean> => {
    const authStore = useAuthStore();
    if (authStore.refreshToken) {
      try {
        await authStore.refreshAuth();
        return true;
      } catch {
        // ignore
      }
    }
    return false;
  };

  // 统一执行器：带 auth 预校验与 401 一次性自动刷新重试
  const run = async <T>(
    fn: () => Promise<T>,
    opts: { url: string; method: string; auth?: boolean },
  ): Promise<T | undefined> => {
    try {
      await precheckAuth(opts.auth);
      const result = await fn();

      // 成功请求，更新API状态
      systemStatus.updateApiStatus("good");

      return result;
    } catch (error: any) {
      if (opts.auth && error?.status === 401) {
        const refreshed = await tryRefreshOnce();
        if (refreshed) {
          try {
            return await fn();
          } catch (err) {
            await handleError(err, opts.url, opts.method);
            return;
          }
        }
      }
      await handleError(error, opts.url, opts.method);
    }
  };

  return {
    // removed per backend constraints (GET not supported)

    // POST 请求
    async post(url: string, body?: any, options: any = {}) {
      const { auth, headers, ...rest } = options || {};
      return await run(
        () =>
          $fetch(url, {
            baseURL: config.public.apiBaseUrl,
            method: "POST",
            body,
            headers: {
              "Content-Type": "application/json",
              ...getAuthHeaders(),
              ...headers,
            },
            ...rest,
          }).then((resp: any) => processEnvelope(resp, { url, method: "POST" })),
        { url, method: "POST", auth },
      );
    },

    // removed per backend constraints (PUT not supported)

    // removed per backend constraints (DELETE not supported)

    // POST 请求（公开，不需要认证）
    async postPublic(url: string, body?: any, options: any = {}) {
      try {
        const result = await $fetch(url, {
          baseURL: config.public.apiBaseUrl,
          method: "POST",
          body,
          headers: {
            "Content-Type": "application/json",
            ...options.headers,
          },
          ...options,
        }).then((resp: any) => processEnvelope(resp, { url, method: "POST" }));

        // 成功请求，更新API状态
        systemStatus.updateApiStatus("good");

        return result;
      } catch (error) {
        await handleError(error, url, "POST");
      }
    },

    // POST 表单请求（multipart/form-data）
    async postForm(url: string, formData: FormData, options: any = {}) {
      const { auth, headers, ...rest } = options || {};
      return await run(
        () =>
          $fetch(url, {
            baseURL: config.public.apiBaseUrl,
            method: "POST",
            body: formData,
            headers: {
              ...getAuthHeaders(),
              ...headers,
              // 不设置 Content-Type，让浏览器自动设置 multipart/form-data 边界
            },
            ...rest,
          }).then((resp: any) => processEnvelope(resp, { url, method: "POST" })),
        { url, method: "POST", auth },
      );
    },
  };
};
