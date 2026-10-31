<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '@/lib/utils';

	// Textarea —— 多行文本原语（输入族唯一多行实现；2026-10-29 一致性收口新增）。
	// 与 forms/Input 共用 .input-surface 表面与 label/error 契约；所有多行输入
	// （简介/公告/备注/群简介/瞬间正文）统一用这一个壳与字号，不再各页手写 padding。
	// 高度由 rows / minHeightClass 决定，壳内边距固定 px-3.5 py-3（与单行 39/43 档区分）。
	// footer snippet 渲染在壳内底部（字符计数等），保证计数与输入框同一边界。

	let {
		value = '',
		onChange,
		placeholder = '',
		rows = 3,
		maxlength,
		label,
		labelSnippet,
		id,
		ariaLabel,
		hasError = false,
		errorText,
		disabled = false,
		resize = 'none',
		/** textarea 额外 class（min-h 等布局几何在这里传） */
		inputClass = '',
		/** 外层容器额外 class */
		class: className = '',
		/** 壳内底部插槽（字符计数等） */
		footer,
		onEnter
	}: {
		value?: string;
		onChange?: (value: string) => void;
		placeholder?: string;
		rows?: number;
		maxlength?: number;
		label?: string;
		labelSnippet?: Snippet;
		id?: string;
		ariaLabel?: string;
		hasError?: boolean;
		errorText?: string;
		disabled?: boolean;
		resize?: 'none' | 'vertical';
		inputClass?: string;
		class?: string;
		footer?: Snippet;
		onEnter?: () => void;
	} = $props();

	const uid = $props.id();
	const autoId = 'textarea-' + uid;
	const inputId = $derived(id ?? autoId);
	const errorId = $derived(inputId + '-error');
	const errId = $derived(hasError && errorText ? errorId : undefined);
</script>

<div class={cn('flex flex-col gap-1.5', className)}>
	{#if label}
		<label for={inputId} class="text-xs font-medium tracking-field text-text-secondary">{label}</label>
	{:else if labelSnippet}
		{@render labelSnippet()}
	{/if}
	<div class="rounded-md input-surface px-3.5 py-3">
		<!-- 聚焦态由外层 input-surface:focus-within 整框承担（见 utilities.css） -->
		<textarea
			id={inputId}
			{rows}
			{placeholder}
			{maxlength}
			{disabled}
			aria-label={ariaLabel}
			aria-invalid={hasError ? 'true' : undefined}
			aria-describedby={errId}
			bind:value
			oninput={(e) => onChange?.(e.currentTarget.value)}
			onkeydown={(e) => {
				if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) onEnter?.();
			}}
			class={cn(
				'w-full resize-none bg-transparent text-md leading-copy text-text-primary',
				resize === 'vertical' ? 'resize-y' : 'resize-none',
				inputClass
			)}
		></textarea>
		{#if footer}
			<div class="mt-2 flex items-center justify-end text-xs text-text-secondary">
				{@render footer()}
			</div>
		{/if}
	</div>
	{#if hasError && errorText}
		<p id={errorId} class="text-xs text-status-danger" role="alert">{errorText}</p>
	{/if}
</div>
