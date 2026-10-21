<script lang="ts">
	import type { Snippet } from 'svelte';

	// 设置页区块骨架：左标题/描述栏（平板 240px / 桌面 22vw 自适应，clamp 240–300px）+ 右内容区；
	// 平板（≥768px）也启用两栏——单栏时行内 label 与右侧控件间距可达 400px。
	// 区块间分隔线由父级 divide-y 承担；页顶/页底留白由页面 contentClassName 统一承担，
	// 首区块由页面传 first:pt-0、末区块传 last:pb-0，避免与其叠成双倍留白。

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

<div class="grid grid-cols-1 gap-3 py-6 min-[768px]:grid-cols-[240px_minmax(0,1fr)] min-[768px]:gap-8 lg:grid-cols-[clamp(240px,22vw,300px)_minmax(0,1fr)] lg:gap-10 {className}">
	<div>
		<h2 class="text-md font-semibold text-text-primary">{title}</h2>
		<p class="text-xs text-text-secondary mt-1 leading-relaxed">
			{description}
		</p>
	</div>
	<div>
		{@render children()}
	</div>
</div>
