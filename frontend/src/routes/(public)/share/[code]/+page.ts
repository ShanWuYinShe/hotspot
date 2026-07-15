import { shareService } from '$lib/api/services';
import { isSuccess } from '$lib/api/client';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ params, fetch }) => {
  const shareCode = params.code;

  try {
    const response = await shareService.get({ share_code: shareCode }, { fetch });

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
