import { requireProfileSection } from '@/lib/route-guards';

// 个人主页分区守卫：/me/:section 仅接受 moments / groups / relationships，
// 未知分区（如 /me/whatever）直接 404，避免静默回退到总览页。

export const load = ({ params }: { params: { section: string } }) => {
	requireProfileSection(params.section);
	return {};
};
