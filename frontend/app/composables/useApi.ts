/**
 * API 客户端 Composable
 * 封装统一的 HTTP 请求逻辑
 */


export const useApi = () => {
  // 预加载依赖，避免动态导入的性能开销
  const { handleError: handleAppError } = useErrorHandler();
  const config = useRuntimeConfig();

  // 🎯 最优雅的响应处理：统一处理后端的 { code, msg/message, data } 格式
  const processEnvelope = (resp: unknown, meta: { url: string; method: string }) => {
    // 检查是否为标准API响应格式
    if (resp && typeof resp === "object" && "code" in resp) {
      const envelope = resp as any;
      
      // ✅ 成功：code === 0，返回数据
      if (envelope.code === 0) {
        return envelope.data;
      }

      // ❌ 失败：code !== 0，创建统一的业务错误
      const message = envelope.msg || envelope.message || "操作失败";
      const error: any = new Error(message);
      
      // 标记为业务错误，便于错误处理器识别
      error.isBusinessError = true;
      error.businessCode = envelope.code;
      error.status = 200; // HTTP成功，业务失败
      error.data = envelope;
      error.endpoint = meta.url;
      error.method = meta.method;
      
      throw error;
    }

    // 非标准格式，直接返回
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

  // 统一错误处理
  const handleError = async (
    error: any,
    endpoint?: string,
    method?: string,
  ) => {
    // 使用统一的错误处理器获取准确的错误信息
    const errorInfo = handleApiError(error, endpoint);
    
    // 增强错误信息
    const enhancedError = {
      ...error,
      endpoint,
      method,
      title: errorInfo.title,
      message: errorInfo.message,
      type: errorInfo.type,
      canRetry: errorInfo.canRetry,
      actionText: errorInfo.actionText
    };

    // 更新系统状态
    if (errorInfo.type === 'network' || errorInfo.type === 'server') {
      // 系统状态更新已移除
    } else if (errorInfo.type === 'client' || errorInfo.type === 'validation') {
      // 系统状态更新已移除
    } else if (errorInfo.type === 'auth') {
      // 系统状态更新已移除
    }

    // 只使用一套错误处理系统，避免重复处理
    if (import.meta.dev) {
      console.error(`[API Error - ${endpoint}]:`, enhancedError);
    }

    // 处理认证错误（不在这里清除token，交由中间件处理）
    if (error?.status === 401 || errorInfo.type === 'auth') {
      // 可以在这里发送事件通知认证状态变化
      // 但不直接操作认证store，保持职责分离
    }

    throw enhancedError;
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
      // 系统状态更新已移除

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
      return undefined;
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
        // 系统状态更新已移除

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
