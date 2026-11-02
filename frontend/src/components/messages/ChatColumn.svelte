<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '@/lib/utils';

	// ChatColumn —— 聊天页内容轴（页头 / 消息列 / 输入栏三处共用的唯一列宽来源）。
	//
	// 聊天页是独立画布页面（豁免 StandardPageLayout），但三块内容必须同轴：
	// 内容轴随视口逐档放宽：base 768 / lg 1024 / 2xl 1152 / ≥2200 1280；
	// 页头、消息列、输入栏三处都消费本组件，改档位只改这里，页面/组件内不得再写 max-w-*。
	let {
		children,
		class: className = ''
	}: {
		children: Snippet;
		class?: string;
	} = $props();
</script>

<div data-chat-column class={cn('mx-auto w-full max-w-3xl lg:max-w-5xl 2xl:max-w-6xl min-[2200px]:max-w-7xl', className)}>
	{@render children()}
</div>
