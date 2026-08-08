<script lang="ts">
	import { Search, X } from '@lucide/svelte';
	import type { Snippet } from 'svelte';
	import { cn } from '@/lib/utils';

	// 搜索输入框（对齐 Figma Make SearchField）：
	// rounded-md + input-surface，无内联 padding（由调用方通过 class 提供）。
	// value 非空时显示清空按钮，否则渲染 trailing。

	let {
		value,
		onChange,
		placeholder = '搜索...',
		class: className = '',
		/** 进入页面后自动聚焦（搜索页样板行为）。 */
		autofocus = false,
		/** 回车回调（搜索页把当前词记入最近搜索）。 */
		onEnter,
		/** 输入框字号（text-* 刻度类，默认 text-sm=12px）。禁止内联 font-size。 */
		textClass = 'text-sm',
		trailing
	}: {
		value: string;
		onChange: (value: string) => void;
		placeholder?: string;
		class?: string;
		autofocus?: boolean;
		onEnter?: (value: string) => void;
		textClass?: string;
		trailing?: Snippet;
	} = $props();
</script>

<div class={cn('flex items-center gap-2.5 rounded-md input-surface', className)}>
	<Search size={14} color="var(--cst-text-secondary)" />
	<!-- svelte-ignore a11y_autofocus：搜索页进入自动聚焦是样板行为 -->
	<input
		{autofocus}
		bind:value
		oninput={(e) => onChange(e.currentTarget.value)}
		onkeydown={(e) => {
			if (e.key === 'Enter') onEnter?.(value);
		}}
		{placeholder}
		class="min-w-0 flex-1 bg-transparent outline-none text-text-primary {textClass}"
	/>
	{#if value}
		<button
			type="button"
			class="shrink-0 outline-none"
			aria-label="清空搜索"
			onclick={() => onChange('')}
		>
			<X size={13} color="var(--cst-text-secondary)" />
		</button>
	{:else if trailing}
		{@render trailing()}
	{/if}
</div>
