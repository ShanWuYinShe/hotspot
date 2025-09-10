/**
 * 全局错误处理插件
 * 设置全局错误处理器
 */


export default defineNuxtPlugin(async () => {
  // 获取错误处理器
  const { handleError: handleGlobalError } = useErrorHandler();

  // 处理未捕获的Promise拒绝
  window.addEventListener("unhandledrejection", async (event) => {
    console.warn("[ErrorHandler] Unhandled promise rejection:", event.reason);

    try {
      await handleGlobalError(event.reason, 'GlobalErrorHandler');
      event.preventDefault(); // 阻止默认的错误处理
    } catch (handlerError) {
      console.error("[ErrorHandler] Error in error handler:", handlerError);
    }
  });

  // 处理JavaScript运行时错误
  window.addEventListener("error", async (event) => {
    console.warn(
      "[ErrorHandler] JavaScript error:",
      event.error || event.message,
    );

    try {
      const error = event.error || new Error(event.message);
      if (error) {
        error.context = {
          filename: event.filename,
          lineno: event.lineno,
          colno: event.colno,
        };
        await handleGlobalError(error, 'GlobalErrorHandler');
      }
    } catch (handlerError) {
      console.error("[ErrorHandler] Error in error handler:", handlerError);
    }
  });

  // 处理资源加载错误
  window.addEventListener(
    "error",
    (event) => {
      const target = event.target as HTMLElement;
      if (target && target !== (window as any)) {
        const error = new Error(
          `Failed to load resource: ${(target as any).src || (target as any).href || "unknown"}`,
        );
        error.name = "ResourceLoadError";
        (error as any).context = {
          tagName: target.tagName,
          src: (target as any).src,
          href: (target as any).href,
        };

        handleGlobalError(error, 'GlobalErrorHandler');
      }
    },
    true,
  ); // 使用捕获阶段

  // 网络状态监听
  window.addEventListener("offline", () => {
    const error = new Error("网络连接已断开");
    error.name = "NetworkOfflineError";
    (error as any).status = 0;
    handleGlobalError(error, 'GlobalErrorHandler');
  });

  if (import.meta.dev) {
    console.debug("✅ 全局错误处理插件初始化完成");
  }
});
