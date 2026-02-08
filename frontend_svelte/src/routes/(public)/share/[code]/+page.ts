import { getShare } from '$lib/api/services/share';
import { isSuccess } from '$lib/api/client';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ params }) => {
	const shareCode = params.code;

	try {
		const response = await getShare({ share_code: shareCode });
		
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
