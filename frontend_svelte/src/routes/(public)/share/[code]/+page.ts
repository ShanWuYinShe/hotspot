import { shareService } from '$lib/api/services';
import { isSuccess } from '$lib/api/client';
import type { PageLoad } from './$types';

/**
 * 禁用 SSR：API client 依赖 HttpOnly Cookie 鉴权，
 * SSR 阶段的 globalThis.fetch 无法解析相对 URL 且不携带 Cookie。
 */
export const ssr = false;

export const load: PageLoad = async ({ params }) => {
  const shareCode = params.code;

  try {
    const response = await shareService.get({ share_code: shareCode });

    if (isSuccess(response) && response.data) {
      return {
        share: response.data,
        shareCode
      };
    }

    return {
      share: null,
      shareCode,
      error: response.message || '分享不存在或已过期'
    };
  } catch (error) {
    return {
      share: null,
      shareCode,
      error: '加载失败'
    };
  }
};
