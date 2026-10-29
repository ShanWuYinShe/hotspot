<script lang="ts">
	import { Search, X } from '@lucide/svelte';
	import type { Snippet } from 'svelte';
	import { cn } from '@/lib/utils';

	// 搜索输入框原语（2026-10-29 重构）。
	// - 胶囊形（rounded-full）+ 39px 高：py-2(8) + text-md 行高 21 + hairline 2。
	//   与表单输入（forms/Input，rounded-md 16）形成「搜索 = 胶囊 / 表单 = 圆角矩形」
	//   两派语言：搜索更轻、更可点，表单更稳。
	// - 表面/边框/焦点环/错误态全部由 input-surface 承担（见 utilities.css），
	//   本组件只负责结构与行为；调用方不再传 padding/圆角（历史上同一原语被传成
	//   px-3/py-2.5/rounded-md/rounded-full 四种形态）。
	// - 清空按钮 32px 视觉（hit-slop → 44 命中），hover 填充；清空后焦点回到输入框。
	// - 默认字号 text-md(14)：修复「默认 text-sm(12) 与各页传参并存」的字号漂移。
	let {
		value,
		onChange,
		placeholder = '搜索…',
		class: className = '',
		/** 进入页面后自动聚焦（搜索页样板行为）。 */
		autofocus = false,
		/** 回车回调（搜索页把当前词记入最近搜索）。 */
		onEnter,
		/** 聚焦回调（地图移动端搜索框聚焦时展开面板）。 */
		onFocus,
		/** 读屏名称：按页面语义传入（如「搜索会话」），不要一律用「搜索」。 */
		ariaLabel = '搜索',
		/** 输入框字号（text-* 刻度类，默认 text-md=14px）。禁止内联 font-size。 */
		textClass = 'text-md',
		trailing
	}: {
		value: string;
		onChange: (value: string) => void;
		placeholder?: string;
		class?: string;
		autofocus?: boolean;
		onEnter?: (value: string) => void;
		onFocus?: () => void;
		ariaLabel?: string;
		textClass?: string;
		trailing?: Snippet;
	} = $props();

	let inputEl = $state<HTMLInputElement | null>(null);

	function clear() {
		onChange('');
		inputEl?.focus();
	}
</script>

<div class={cn('group flex items-center gap-2.5 rounded-full input-surface px-3.5 py-2', className)}>
	<Search
		size={16}
		strokeWidth={1.75}
		class="shrink-0 text-text-secondary transition-colors duration-rapid group-focus-within:text-accent-default"
		aria-hidden="true"
	/>
	<!-- svelte-ignore a11y_autofocus：搜索页进入自动聚焦是样板行为 -->
	<!-- 聚焦态由外层 input-surface:focus-within 整框承担（见 utilities.css），内层不画自身焦点框 -->
	<input
		bind:this={inputEl}
		{autofocus}
		aria-label={ariaLabel}
		type="text"
		inputmode="search"
		enterkeyhint="search"
		autocomplete="off"
		autocapitalize="off"
		spellcheck={false}
		data-search-input
		bind:value
		oninput={(e) => onChange(e.currentTarget.value)}
		onkeydown={(e) => {
			// IME 组合中的 Enter/Escape 是选词/取消候选，不能当提交/清空（中文输入法致命）
			if (e.isComposing) return;
			if (e.key === 'Enter') onEnter?.(value);
			if (e.key === 'Escape' && value) {
				e.preventDefault();
				// 阻止冒泡：bits-ui Dialog 在 document 上监听 Esc 且不检查 defaultPrevented，
				// 不拦截会「清空输入 + 同时关闭弹层」；有内容时第一次 Esc 只清空，
				// 空输入时第二次 Esc 才交给弹层关闭。
				e.stopPropagation();
				onChange('');
			}
		}}
		onfocus={() => onFocus?.()}
		{placeholder}
		class="min-w-0 flex-1 bg-transparent text-text-primary {textClass}"
	/>
	{#if value}
		<!-- 32px 视觉 + hit-slop → 44px 命中；-my-2 抵消纵向占位，输入框高度不变 -->
		<button
			type="button"
			class="hit-slop hover-btn -my-2 -mr-1 flex size-8 shrink-0 items-center justify-center rounded-full text-text-secondary focus-ring hover:bg-interactive-hover hover:text-text-primary"
			aria-label="清空搜索"
			onclick={clear}
		>
			<X size={14} strokeWidth={2} aria-hidden="true" />
		</button>
	{:else if trailing}
		{@render trailing()}
	{/if}
</div>
