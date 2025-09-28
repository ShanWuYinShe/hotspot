/**
 * 认证相关工具函数
 * 替代过度设计的 useAuthGuard composable
 */

// 简单的认证检查错误
export const createAuthRequiredError = (options?: {
  accounts?: any[];
  message?: string;
}) => {
  const error = new Error(
    options?.message || "需要登录后才能进行该操作",
  ) as any;
  error.status = 401;
  error.code = "AUTH_REQUIRED";
  error.details = options?.accounts;
  return error;
};

// 检查是否已认证，未认证则抛出错误
export function ensureAuthenticatedOrThrow(): void {
  const auth = useAuthStore();
  if (auth.isAuthenticated) return;

  const list = auth.accountsList || [];
  throw createAuthRequiredError({
    accounts: (list as any[]).map((a: any) => ({
      user_id: a.user_id,
      nickname: a.nickname,
      avatar_url: a.avatar_url,
    })),
  });
}

// 检查是否已认证（返回布尔值）
export function isAuthenticated(): boolean {
  const auth = useAuthStore();
  return auth.isAuthenticated;
}

// 获取当前用户
export function getCurrentUser() {
  const auth = useAuthStore();
  return auth.user;
}
