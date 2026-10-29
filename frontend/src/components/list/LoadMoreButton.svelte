<script lang="ts">
	import Button from '@/components/ui/button.svelte';

	// LoadMoreButton — 手动「加载更多」按钮的唯一实现（components/list）。
	// 内容型列表默认走无限滚动哨兵；需要显式分页控制成本的场景（管理后台
	// 六模块）用它，不再各页手写 pill 按钮。label 由调用方组装
	//（「加载更多」或「加载更多 (N)」）。
	// 形态走 ui/button 的 secondary + flat（无变体投影）+ shape="pill"（圆角由原语形状档表达，
	// 调用方不再传 rounded-*）+ className 只留布局/字色/字形：
	// h-auto（覆盖尺寸档高度）、px-5 py-2、text-sm、次级文字色（hover 转正文色）——与原手写样式逐类等价。
	// hit-slop：视觉高度 36px 低于 44px 触控下限，用伪元素向四周扩 6px（不参与布局流、
	// 不改变视觉）。2026-10-29 真实 seed 数据（首页出现「加载更多」）触控门禁实测 eff=35 暴露。
	let {
		label = '加载更多',
		loading = false,
		onLoad
	}: {
		label?: string;
		loading?: boolean;
		onLoad: () => void;
	} = $props();
</script>

<div class="flex justify-center">
	<Button
		variant="secondary"
		flat
		shape="pill"
		disabled={loading}
		onclick={onLoad}
		class="hit-slop h-auto px-5 py-2 text-sm text-text-secondary hover:text-text-primary"
	>
		{loading ? '加载中…' : label}
	</Button>
</div>
