// 活动详情路由参数预加载（对齐其他详情页 +page.ts 模式）。
export const load = ({ params }: { params: { eventId: string } }) => {
	return {
		eventId: params.eventId
	};
};
