<script lang="ts">
	import type { Snippet } from 'svelte';

	// 所有右侧辅助内容的统一对齐原语。
	// 宽度/分隔线/内缩/顶边来自共享 context-rail CSS token。

	let {
		children,
		attached = false,
		className = ''
	}: {
		children: Snippet;
		/** Attached rails fill the available page height and own their scroll region. */
		attached?: boolean;
		className?: string;
	} = $props();
</script>

<aside class="context-rail {attached ? 'context-rail--attached' : ''} {className}">
	<div class={attached ? 'context-rail__content context-rail__content--scroll' : 'context-rail__content'}>
		{@render children()}
	</div>
</aside>
