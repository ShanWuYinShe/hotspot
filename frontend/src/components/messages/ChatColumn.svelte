<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '@/lib/utils';

	// ChatColumn —— 聊天页内容轴（页头 / 消息列 / 输入栏三处共用的唯一列宽来源）。
	//
	// 聊天页是独立画布页面（豁免 StandardPageLayout），但三块内容必须同轴：
	// 共享「max-w-3xl；≥lg 放宽到 max-w-5xl（1024）」这一档，面板窄于该档时自然 w-full。
	// R16 整改前：消息列用 max-w-3xl lg:max-w-5xl、输入栏只有 max-w-3xl、群聊头部又缺
	// lg:max-w-5xl —— 1440 下输入栏比消息列左右各窄 26px，1920 下头部/输入栏比消息列
	// 每侧窄 128px（「头部飘着、消息挤在一侧」的直接原因）。
	// 现在三处都消费本组件，改档位只改这里；页面/组件内不得再各自写字面 max-w-*。
	let {
		children,
		class: className = ''
	}: {
		children: Snippet;
		class?: string;
	} = $props();
</script>

<div data-chat-column class={cn('mx-auto w-full max-w-3xl lg:max-w-5xl', className)}>
	{@render children()}
</div>
