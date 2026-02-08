import { groupsService } from '$lib/api/services';
import { isSuccess } from '$lib/api/client';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ params }) => {
	const groupId = params.id;

	try {
		const response = await groupsService.getGroup({ group_id: groupId });
		
		if (isSuccess(response) && response.data) {
			return {
				group: response.data,
				groupId
			};
		}
		
		return {
			group: null,
			groupId,
			error: response.message || '群组不存在'
		};
	} catch (error) {
		return {
			group: null,
			groupId,
			error: '加载失败'
		};
	}
};
