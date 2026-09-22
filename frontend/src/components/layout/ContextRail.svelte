<script lang="ts">
	import type { Snippet } from 'svelte';

	// 所有右侧辅助内容的统一对齐原语。
	// 宽度/分隔线/内缩/顶边来自共享 context-rail CSS token。
	// 用具名 section（region 地标）而非 aside/complementary：rail 渲染在页面
	// main 内部，complementary 地标要求顶层（axe landmark-complementary-is-top-level），
	// region 嵌套合法；调用方可传 ariaLabel 覆盖默认名。

	let {
		children,
		attached = false,
		class: className = '',
		ariaLabel = '相关信息'
	}: {
		children: Snippet;
		/** Attached rails fill the available page height and own their scroll region. */
		attached?: boolean;
		class?: string;
		ariaLabel?: string;
	} = $props();
</script>

<section class="context-rail {attached ? 'context-rail--attached' : ''} {className}" aria-label={ariaLabel}>
	<div class={attached ? 'context-rail__content context-rail__content--scroll' : 'context-rail__content'}>
		{@render children()}
	</div>
</section>
