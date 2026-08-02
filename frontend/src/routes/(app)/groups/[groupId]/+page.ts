import { requireNumericId } from '@/lib/route-guards';

// 群组详情守卫：/groups/:groupId 仅接受纯数字 ID，非法 ID 直接 404。

export const load = ({ params }: { params: { groupId: string } }) => {
	requireNumericId(params.groupId);
	return {};
};
