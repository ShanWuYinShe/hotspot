import { requireNumericId } from '@/lib/route-guards';

// 用户详情守卫：/users/:userId 仅接受纯数字 ID，非法 ID 直接 404。

export const load = ({ params }: { params: { userId: string } }) => {
	requireNumericId(params.userId);
	return {};
};
