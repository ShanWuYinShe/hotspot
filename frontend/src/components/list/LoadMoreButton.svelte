<script lang="ts">
	// LoadMoreButton — 手动「加载更多」按钮的唯一实现（components/list）。
	// 内容型列表默认走无限滚动哨兵；需要显式分页控制成本的场景（管理后台
	// 六模块）用它，不再各页手写 pill 按钮。label 由调用方组装
	//（「加载更多」或「加载更多 (N)」）。
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
	<button
		type="button"
		class="hover-btn rounded-full border border-border-default bg-surface-primary px-5 py-2 text-sm text-text-secondary focus-ring transition-colors hover:text-text-primary disabled:opacity-faint"
		disabled={loading}
		onclick={onLoad}
	>
		{loading ? '加载中…' : label}
	</button>
</div>
