import { requireNumericId } from '@/lib/route-guards';

// 私聊窗口守卫：chatId 实为对端 user_id（spec: target_user_id），仅接受纯数字 ID。

export const load = ({ params }: { params: { chatId: string } }) => {
	requireNumericId(params.chatId);
	return {};
};
