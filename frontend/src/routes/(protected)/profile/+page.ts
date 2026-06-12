/**
 * 个人资料页数据加载
 *
 * 使用 +page.ts 而不是 +page.server.ts 的原因：
 * - 后端 API 需要 access_token（存储在客户端）
 * - 服务器端只能访问 refresh_token（HttpOnly cookie）
 * - 因此数据加载必须在客户端进行
 *
 * 优势：
 * - 利用 SvelteKit 的预加载机制（hover 时预加载）
 * - 数据在导航时加载，而不是组件挂载后
 * - 更好的用户体验
 */

import { usersService } from '$lib/api/services';
import { isSuccess } from '$lib/api/client';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch }) => {
  try {
    const response = await usersService.getMyProfile({ fetch });

    if (isSuccess(response) && response.data) {
      return {
        profile: response.data
      };
    }

    return {
      profile: null,
      error: response.message || '加载失败'
    };
  } catch (error) {
    return {
      profile: null,
      error: '加载失败'
    };
  }
};
