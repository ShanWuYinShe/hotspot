import { usersService } from '$lib/api/services';
import { isSuccess } from '$lib/api/client';
import type { PageLoad } from './$types';

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
