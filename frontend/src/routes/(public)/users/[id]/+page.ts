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
    const response = await usersService.getProfile({ user_id: userId }, { fetch });

    if (isSuccess(response) && response.data) {
      return {
        user: response.data.user,
        recentMoments: response.data.recent_moments,
        userId
      };
    }

    return {
      user: null,
      recentMoments: null,
      userId,
      error: response.message || '用户不存在'
    };
  } catch (error) {
    return {
      user: null,
      recentMoments: null,
      userId,
      error: '加载失败'
    };
  }
};
