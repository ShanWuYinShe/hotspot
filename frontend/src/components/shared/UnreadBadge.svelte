<script lang="ts">
	// UnreadBadge — 全局未读数气泡（导航图标/侧栏项/底部 Tab 消费）。
	// 契约：紧凑型徽标（AGENTS.md「徽标与标签」）inline-flex items-center
	// justify-center leading-none + h-5 px-2 + text-xxs；计数 99+ 截断，
	// tabular-nums 防数字跳动，max 99 后不再增长宽度。
	// 颜色走 accent 实底（对齐 ChatListItem 未读徽章既有视觉：bg-accent-default
	// + text-text-on-accent）。

	let { count, max = 99 }: { count: number; max?: number } = $props();

	const display = $derived(count > max ? `${max}+` : String(count));
</script>

{#if count > 0}
	<span
		class="inline-flex h-5 min-w-5 items-center justify-center rounded-full px-2 text-xxs font-semibold leading-none tabular-nums bg-accent-default text-text-on-accent"
	>{display}</span
	>
{/if}
