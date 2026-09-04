<script lang="ts">
	import { Search, X } from '@lucide/svelte';
	import type { Snippet } from 'svelte';
	import { cn } from '@/lib/utils';

	// 搜索输入框：rounded-md + input-surface，无内联 padding（由调用方通过 class 提供）。
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
	<!-- 聚焦态由外层 input-surface:focus-within 整框承担（见 utilities.css），内层不画自身焦点框 -->
	<input
		{autofocus}
		aria-label="搜索"
		bind:value
		oninput={(e) => onChange(e.currentTarget.value)}
		onkeydown={(e) => {
			if (e.key === 'Enter') onEnter?.(value);
			if (e.key === 'Escape' && value) onChange('');
		}}
		{placeholder}
		class="min-w-0 flex-1 bg-transparent text-text-primary {textClass}"
	/>
	{#if value}
		<button
			type="button"
			class="hover-action -m-1.5 flex shrink-0 items-center justify-center rounded-full p-1.5 focus-ring hover:text-text-primary"
			aria-label="清空搜索"
			onclick={() => onChange('')}
		>
			<X size={13} color="var(--cst-text-secondary)" />
		</button>
	{:else if trailing}
		{@render trailing()}
	{/if}
</div>
