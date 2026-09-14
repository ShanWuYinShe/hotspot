import { redirect } from '@sveltejs/kit';

// 附近一级 Tab 落点：原聚合门户页（搜索框 + 四入口卡片 + 预览横滑）已收敛为
// 三段独立内容页（附近的人/群组/动态，各为独立路由、不再共用段位导航；
// 地图/搜索收敛为各页 header 工具入口）。/nearby 语义由「附近的人」承接。
export function load() {
	redirect(307, '/nearby/people');
}
