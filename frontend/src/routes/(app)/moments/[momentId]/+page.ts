import { requireNumericId } from '@/lib/route-guards';

// 动态详情守卫：/moments/:postId 仅接受纯数字 ID，非法 ID 直接 404。

export const load = ({ params }: { params: { postId: string } }) => {
	requireNumericId(params.postId);
	return {};
};
