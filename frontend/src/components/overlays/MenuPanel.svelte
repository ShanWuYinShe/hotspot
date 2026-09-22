<script lang="ts">
	import { Check } from '@lucide/svelte';
	import type { Snippet } from 'svelte';

	// MenuPanel — 弹出菜单面板的唯一实现（标题 + 选项行 + 可选底部动作）。
	// 此前地图源菜单/筛选菜单各写一套行 markup，行间距与选中态各自为政；
	// 新菜单只传数据（options/footer），不复制行结构。浮层容器仍由调用方的
	// ui/popover（或 dropdown-menu）Content 承载，本组件只负责面板内容。
	// 选中指示统一为实心圆勾（此前地图源用纯文本 ✓，与筛选菜单不统一）。
	export interface MenuPanelOption {
		id: string;
		title: string;
		/** 副标题（选项说明） */
		desc?: string;
		/** 行尾元信息（如筛选计数 "12" / "—"）；不传不占位 */
		badge?: string;
		selected: boolean;
		/** 未选中时整行降淡（如筛选菜单的已隐藏项） */
		dimmed?: boolean;
	}

	export interface MenuPanelFooter {
		actionLabel: string;
		actionDisabled?: boolean;
		onAction: () => void;
	}

	let {
		title,
		/** 标题右侧元信息（如 "2/3 已选"）；不传时标题退为小字 caption 形态 */
		meta,
		options,
		onSelect,
		/** 单选（radio）/ 多选（checkbox）语义，仅决定行的 role */
		multi = false,
		/** 行首图标（筛选菜单的图钉徽章等）；按选项渲染 */
		leading,
		/** 底部左侧点缀（如图例）；不传留空 */
		footerLeading,
		footer,
		widthClass = 'w-64',
		ariaLabel
	}: {
		title: string;
		meta?: string;
		options: MenuPanelOption[];
		onSelect: (id: string) => void;
		multi?: boolean;
		leading?: Snippet<[MenuPanelOption]>;
		footerLeading?: Snippet;
		footer?: MenuPanelFooter;
		widthClass?: string;
		ariaLabel?: string;
	} = $props();
</script>

<div class="{widthClass} overflow-hidden" role="menu" aria-label={ariaLabel ?? title}>
	{#if meta !== undefined}
		<div class="flex items-center justify-between border-b border-border-default px-3.5 py-2.5">
			<span class="text-sm font-medium text-text-primary">{title}</span>
			<span class="text-xxs text-text-secondary">{meta}</span>
		</div>
	{:else}
		<p class="px-2.5 pt-1 pb-1.5 text-xxs" style="color:var(--cst-text-secondary);">{title}</p>
	{/if}
	<div class="flex flex-col gap-1 p-1.5">
		{#each options as opt (opt.id)}
			<button
				type="button"
				role={multi ? 'menuitemcheckbox' : 'menuitemradio'}
				aria-checked={opt.selected}
				class="flex w-full items-center gap-3 rounded-lg px-2.5 py-2 text-left focus-ring transition-colors {opt.selected ? 'bg-accent-soft hover:bg-interactive-active' : 'hover:bg-interactive-hover'} {opt.dimmed ? 'opacity-soft' : ''}"
				onclick={() => onSelect(opt.id)}
			>
				{#if leading}
					{@render leading(opt)}
				{/if}
				<span class="min-w-0 flex-1">
					<span class="block text-sm text-text-primary">{opt.title}</span>
					{#if opt.desc}
						<span class="mt-0.5 block text-xxs text-text-secondary">{opt.desc}</span>
					{/if}
				</span>
				{#if opt.badge !== undefined}
					<span class="shrink-0 text-xs text-text-secondary">{opt.badge}</span>
				{/if}
				<span
					class="flex size-5 shrink-0 items-center justify-center rounded-full transition-colors {opt.selected ? 'bg-accent-default text-text-on-accent' : 'bg-border-default text-transparent'}"
					aria-hidden="true"
				>
					<Check class="size-3" strokeWidth={3} />
				</span>
			</button>
		{/each}
	</div>
	{#if footer}
		<div class="flex items-center justify-between border-t border-border-default px-3.5 py-2">
			<span class="flex items-center gap-2 text-xs text-text-secondary">{@render footerLeading?.()}</span>
			<button
				type="button"
				class="link-btn focus-ring rounded-sm px-1 text-xs text-accent-default {footer.actionDisabled ? 'pointer-events-none opacity-soft' : ''}"
				disabled={footer.actionDisabled}
				onclick={footer.onAction}
			>
				{footer.actionLabel}
			</button>
		</div>
	{/if}
</div>
