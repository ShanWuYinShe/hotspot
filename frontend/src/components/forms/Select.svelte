<script lang="ts">
	import { ChevronDown } from '@lucide/svelte';
	import type { Snippet } from 'svelte';
	import { cn } from '@/lib/utils';

	// Select —— 下拉选择原语（输入族唯一 select 实现，2026-10-29 一致性收口新增）。
	// variant='field'：表单/筛选字段（.input-surface + 16 圆角 + text-md + 右侧 chevron，
	//   与 forms/Input 同视觉；density='default' 43px / 'compact' 39px）。
	// variant='ghost'：行内排序/轻量切换（透明底、accent 文字、无边框），
	//   与筛选胶囊/行内动作同一语言；不要用 field 外观做行内排序。
	// 保留原生 <select> 的可访问性；appearance-none 后由组件绘制 chevron。

	let {
		value = '',
		onChange,
		label,
		labelSnippet,
		id,
		ariaLabel,
		disabled = false,
		variant = 'field',
		density = 'default',
		class: className = '',
		children
	}: {
		value?: string;
		onChange?: (value: string) => void;
		label?: string;
		labelSnippet?: Snippet;
		id?: string;
		ariaLabel?: string;
		disabled?: boolean;
		variant?: 'field' | 'ghost';
		density?: 'default' | 'compact';
		class?: string;
		children?: Snippet;
	} = $props();

	const uid = $props.id();
	const autoId = 'select-' + uid;
	const selectId = $derived(id ?? autoId);
	const shellPadding = $derived(density === 'compact' ? 'px-3.5 py-2' : 'px-4 py-2.5');
</script>

{#if variant === 'ghost'}
	<span class={cn('relative inline-flex items-center', className)}>
		<select
			id={selectId}
			{value}
			{disabled}
			aria-label={ariaLabel}
			onchange={(e) => onChange?.((e.currentTarget as HTMLSelectElement).value)}
			class="cursor-pointer appearance-none rounded-sm bg-transparent py-0.5 pl-0.5 pr-4 text-xs font-medium text-accent-default focus-ring"
		>
			{@render children?.()}
		</select>
		<ChevronDown class="pointer-events-none absolute right-0 size-3 text-accent-default" strokeWidth={1.75} aria-hidden="true" />
	</span>
{:else}
	<div class={cn('flex flex-col gap-1.5', className)}>
		{#if label}
			<label for={selectId} class="text-xs font-medium tracking-field text-text-secondary">{label}</label>
		{:else if labelSnippet}
			{@render labelSnippet()}
		{/if}
		<span class="relative flex items-center">
			<select
				id={selectId}
				{value}
				{disabled}
				aria-label={ariaLabel}
				onchange={(e) => onChange?.((e.currentTarget as HTMLSelectElement).value)}
				class={cn('input-surface w-full cursor-pointer appearance-none rounded-md pr-9 text-md text-text-primary', shellPadding)}
			>
				{@render children?.()}
			</select>
			<ChevronDown class="pointer-events-none absolute right-3 size-3.5 text-text-secondary" strokeWidth={1.75} aria-hidden="true" />
		</span>
	</div>
{/if}
