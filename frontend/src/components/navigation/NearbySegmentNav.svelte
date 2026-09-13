<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import TabBar from './TabBar.svelte';
	import { ROUTES } from '@/lib/routes';

	// NearbySegmentNav — 附近板块统一段位导航（附近的人/群组/动态）。
	// 三个分区子页共享，置于 PageHeader below 随 header 吸顶；段间为真实路由
	// 导航（非单页状态切换），保留各子页深链与 moments 的 ?tab= 内容源参数。

	const SEGMENTS = [
		{ id: 'people', label: '附近的人', path: ROUTES.nearbyPeople },
		{ id: 'groups', label: '附近群组', path: ROUTES.nearbyGroups },
		{ id: 'moments', label: '附近动态', path: ROUTES.nearbyMoments }
	];

	function activeFromPath(pathname: string): string {
		if (pathname === ROUTES.nearbyGroups) return 'groups';
		if (pathname === ROUTES.nearbyMoments) return 'moments';
		return 'people';
	}

	const active = $derived(activeFromPath(page.url.pathname));
</script>

<TabBar
	items={SEGMENTS}
	value={active}
	onChange={(id) => {
		const target = SEGMENTS.find((s) => s.id === id);
		if (target) void goto(target.path);
	}}
	variant="segmented"
/>
