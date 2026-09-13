import { redirect } from '@sveltejs/kit';

// 附近一级 Tab 落点：原聚合门户页（搜索框 + 四入口卡片 + 预览横滑）已收敛为
// 三段内容页（附近的人/群组/动态，段位由 NearbySegmentNav 承担、地图/搜索
// 收敛为 header 工具入口）。/nearby 语义由默认段位「附近的人」承接。
export function load() {
	redirect(307, '/nearby/people');
}
