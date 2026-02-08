import { usersService } from '$lib/api/services';
import { isSuccess } from '$lib/api/client';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ params }) => {
	const userId = parseInt(params.id);

	try {
		const response = await usersService.getUser({ user_id: userId });
		
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
