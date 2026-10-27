<script lang="ts">
	import type { Snippet } from 'svelte';

	// 设置页区块骨架（R13 改单列）：组标题/描述在上、内容卡在下。
	// 原两栏（左 240–300 描述列 + 右内容列）在 1440 宽下会留出 130+px 空档，且右列被压窄
	// 导致分段控件（主题/明暗/地图源）文字截断成首字；单列后每行控件获得整行宽度。
	// 区块间距由本组件的 pb-6 单侧提供；页顶/页底留白仍由页面 contentClassName 统一承担。

	let {
		title,
		description,
		class: className = '',
		children
	}: {
		title: string;
		description: string;
		class?: string;
		children: Snippet;
	} = $props();
</script>

<section class="pb-6 last:pb-0 {className}">
	<div class="mb-3">
		<h2 class="text-md font-semibold text-text-primary">{title}</h2>
		<p class="mt-1 text-xs leading-relaxed text-text-secondary">{description}</p>
	</div>
	{@render children()}
</section>
