<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '@/lib/utils';

	// Checkbox —— 复选框原语（输入族唯一 checkbox 实现，2026-10-29 一致性收口新增）。
	// 原生 input[type=checkbox] + accent 主题色（accent-accent-default 走语义 token，
	// 三主题/明暗自动生效），视觉 16px；可见文案通过 children 传入，
	// 无可见文案时传 ariaLabel。多选列表/权限勾选/危险确认统一用它，不再各页写裸 checkbox。

	let {
		checked = false,
		onChange,
		disabled = false,
		ariaLabel,
		id,
		class: className = '',
		inputClass = '',
		children
	}: {
		checked?: boolean;
		onChange?: (checked: boolean, event: Event) => void;
		disabled?: boolean;
		ariaLabel?: string;
		id?: string;
		class?: string;
		inputClass?: string;
		children?: Snippet;
	} = $props();

	const uid = $props.id();
	const autoId = 'checkbox-' + uid;
	const inputId = $derived(id ?? autoId);
</script>

<label
	for={inputId}
	class={cn(
		'inline-flex cursor-pointer select-none items-start gap-2 text-sm text-text-primary',
		disabled ? 'cursor-not-allowed opacity-muted' : '',
		className
	)}
>
	<input
		id={inputId}
		type="checkbox"
		{checked}
		{disabled}
		aria-label={ariaLabel}
		onchange={(e) => onChange?.((e.currentTarget as HTMLInputElement).checked, e)}
		class={cn(
			'mt-0.5 size-4 shrink-0 cursor-pointer rounded-sm border-border-default accent-accent-default focus-ring disabled:cursor-not-allowed',
			inputClass
		)}
	/>
	{#if children}
		<span class="min-w-0">{@render children()}</span>
	{/if}
</label>
