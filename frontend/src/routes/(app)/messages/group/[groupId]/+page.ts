import { requireNumericId } from '@/lib/route-guards';

// 群聊窗口守卫：/messages/group/:groupId 仅接受纯数字 ID。

export const load = ({ params }: { params: { groupId: string } }) => {
	requireNumericId(params.groupId);
	return {};
};
