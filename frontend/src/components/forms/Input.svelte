<script lang="ts">
	import { X } from '@lucide/svelte';
	import type { Snippet } from 'svelte';
	import type { HTMLInputAttributes } from 'svelte/elements';
	import { cn } from '@/lib/utils';

	// Input —— 表单单行输入原语（ui-components.md backlog #2 落地）。
	// 样式走 input-surface（聚焦/错误态由 :focus-within 与 :has(input[aria-invalid])
	// 在 utilities.css 统一承担，本组件不写内联风格）。
	// 行为契约对齐鸿蒙 SystemInput：前缀图标（prefix snippet）/ 清除按钮（clearable，
	// value 非空显示）/ 错误态三件套（hasError + errorText → aria-invalid +
	// aria-describedby + role="alert" 就近错误行）。
	// 受控模式：value 进、onChange 出（同 SearchInput）。label 两种形态：
	// label（标准 tracking-field 档位行）或 labelSnippet（自定义整行，含图标等，
	// 调用方自行用 id 关联 for=）；无可见 label 时传 ariaLabel。

	let {
		value = '',
		onChange,
		placeholder = '',
		type = 'text',
		/** 标准字段名（渲染 tracking-field 档位 label，与 input 以 id 关联） */
		label,
		/** 自定义 label 整行（含图标等），调用方自行写 <label for={id}> */
		labelSnippet,
		/** 显式指定 input id（未传自动生成）；有 label/labelSnippet 时必传以关联 */
		id,
		/** 前缀图标区（调用方渲染 lucide 图标，size-4 shrink-0） */
		prefix,
		/** value 非空时显示清空按钮 */
		clearable = false,
		hasError = false,
		/** 错误文案（与 hasError 同传，渲染 role="alert" 就近错误行） */
		errorText,
		disabled = false,
		autocomplete,
		inputmode,
		min,
		max,
		maxlength,
		/** 无可见 label 时的读屏名称 */
		ariaLabel,
		/** 输入字号（text-* 刻度类）。禁止内联 font-size。 */
		textClass = 'text-sm',
		/** 输入框内额外 class（如 w-full） */
		inputClass = '',
		/** 外层容器额外 class */
		class: className = '',
		onEnter
	}: {
		value?: string;
		onChange?: (value: string) => void;
		placeholder?: string;
		type?: 'text' | 'email' | 'password' | 'number';
		label?: string;
		labelSnippet?: Snippet;
		id?: string;
		prefix?: Snippet;
		clearable?: boolean;
		hasError?: boolean;
		errorText?: string;
		disabled?: boolean;
		autocomplete?: HTMLInputAttributes['autocomplete'];
		inputmode?: 'text' | 'numeric' | 'decimal' | 'email';
		min?: number;
		max?: number;
		maxlength?: number;
		ariaLabel?: string;
		textClass?: string;
		inputClass?: string;
		class?: string;
		onEnter?: (value: string) => void;
	} = $props();

	const uid = $props.id();
	const autoId = `input-${uid}`;
	const inputId = $derived(id ?? autoId);
	const errorId = $derived(`${inputId}-error`);
</script>

<div class="flex flex-col gap-1.5">
	{#if label}
		<label for={inputId} class="text-xs font-medium tracking-field text-text-secondary">{label}</label>
	{:else if labelSnippet}
		{@render labelSnippet()}
	{/if}
	<span class={cn('flex items-center gap-2 input-surface rounded-md', className)}>
		{@render prefix?.()}
		<!-- 聚焦态由外层 input-surface:focus-within 整框承担（见 utilities.css），内层不画自身焦点框 -->
		<input
			id={inputId}
			{type}
			{placeholder}
			{disabled}
			{autocomplete}
			{inputmode}
			{min}
			{max}
			{maxlength}
			aria-label={ariaLabel}
			aria-invalid={hasError ? 'true' : undefined}
			aria-describedby={hasError && errorText ? errorId : undefined}
			bind:value
			oninput={(e) => onChange?.(e.currentTarget.value)}
			onkeydown={(e) => {
				if (e.key === 'Enter') onEnter?.(value);
			}}
			class={cn('min-w-0 flex-1 bg-transparent text-text-primary', textClass, inputClass)}
		/>
		{#if clearable && value && !disabled}
			<button
				type="button"
				aria-label="清空"
				class="shrink-0 rounded-full p-0.5 text-text-secondary hover:text-text-primary focus-ring"
				onclick={() => {
					value = '';
					onChange?.('');
				}}
			>
				<X size={14} />
			</button>
		{/if}
	</span>
	{#if hasError && errorText}
		<p id={errorId} class="text-xs text-status-danger" role="alert">{errorText}</p>
	{/if}
</div>
