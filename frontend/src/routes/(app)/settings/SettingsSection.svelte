<script lang="ts">
	import type { Snippet } from 'svelte';

	// 设置页区块骨架：左标题/描述栏（平板 200px / 桌面 220px）+ 右内容区；
	// 平板（≥768px）也启用两栏——单栏时行内 label 与右侧控件间距可达 400px。
	// 区块间分隔线由父级 divide-y 承担，首区块的顶部间距经 class 透传（first:pt-4）。

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

<div class="grid grid-cols-1 gap-4 py-6 min-[768px]:grid-cols-[200px_minmax(0,1fr)] min-[768px]:gap-8 lg:grid-cols-[220px_minmax(0,1fr)] lg:gap-10 {className}">
	<div>
		<h2 class="text-sm font-semibold text-text-primary">{title}</h2>
		<p class="text-xs text-text-secondary mt-1 leading-relaxed">
			{description}
		</p>
	</div>
	<div>
		{@render children()}
	</div>
</div>
