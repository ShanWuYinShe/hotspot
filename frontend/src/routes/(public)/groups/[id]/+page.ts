import { groupsService } from '$lib/api/services';
import { isSuccess } from '$lib/api/client';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ params, fetch }) => {
  const groupId = params.id;

  try {
    const response = await groupsService.getGroup({ group_id: groupId }, { fetch });

    if (isSuccess(response) && response.data) {
      return {
        groupData: response.data,
        groupId
      };
    }

    return {
      groupData: null,
      groupId,
      error: response.message || '群组不存在'
    };
  } catch (error) {
    return {
      groupData: null,
      groupId,
      error: '加载失败'
    };
  }
};
