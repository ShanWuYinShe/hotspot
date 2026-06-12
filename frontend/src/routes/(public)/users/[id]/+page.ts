import { usersService } from '$lib/api/services';
import { isSuccess } from '$lib/api/client';
import type { PageLoad } from './$types';

/**
 * 禁用 SSR：API client 依赖 HttpOnly Cookie 鉴权，
 * SSR 阶段的 globalThis.fetch 无法解析相对 URL 且不携带 Cookie。
 */
export const ssr = false;

export const load: PageLoad = async ({ params, fetch }) => {
  const userId = params.id;

  try {
    const response = await usersService.getUser({ user_id: userId }, { fetch });

    if (isSuccess(response) && response.data) {
      return {
        user: response.data,
        userId
      };
    }

    return {
      user: null,
      userId,
      error: response.message || '用户不存在'
    };
  } catch (error) {
    return {
      user: null,
      userId,
      error: '加载失败'
    };
  }
};
