<script lang="ts">
	import { Check, Palette } from '@lucide/svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import { theme, setColorScheme, setTheme, THEME_OPTIONS, COLOR_SCHEME_OPTIONS, type ColorSchemeMode } from '@/stores/theme.svelte';
	import * as Popover from '@/components/ui/popover';

	// 唯一的外观/模式控制。选择面板保持打开，仅显式完成、外部点击或 Escape 关闭。
	// 弹层行为现由 bits-ui Popover 提供。

	let { class: className = '' }: { class?: string } = $props();

	// 主题选项来自 theme store 的 THEME_OPTIONS（与设置页共用，避免两份配置漂移）

	let open = $state(false);
	let rootEl = $state<HTMLElement | null>(null);

	const currentTheme = $derived(THEME_OPTIONS.find((item) => item.id === theme.theme)?.label ?? '星际');
	const modeLabel = $derived(
		theme.colorScheme === 'auto' ? '跟随系统' : theme.colorScheme === 'dark' ? '深色' : '浅色'
	);

</script>

<Popover.Root bind:open>
	<div bind:this={rootEl} class="relative {className}">
		<ActionIconButton
			icon={Palette}
			label="外观设置"
			onClick={() => (open = !open)}
			variant="ghost"
			size={36}
			class="header-action"
		/>
	</div>

	<Popover.Content customAnchor={rootEl} class="w-76 rounded-xl p-3 shadow-raised-lg border border-border-default bg-surface-primary" role="dialog" aria-label="外观设置">
			<div class="px-1 pb-2.5">
				<p class="eyebrow text-text-secondary">外观</p>
				<p class="mt-0.5 text-xs text-text-secondary">
					当前：<span class="font-medium text-text-primary">{currentTheme}</span> · <span class="font-medium text-text-primary">{modeLabel}</span>
				</p>
			</div>

			<section
				aria-labelledby="theme-choice-label"
				class="rounded-lg ring-1 ring-border-default surface-tinted p-2.5"
			>
				<div class="flex items-center justify-between pb-2 pt-0.5 px-0.5">
					<p id="theme-choice-label" class="eyebrow">主题</p>
					<span class="text-xxs text-text-secondary">选择一种基调</span>
				</div>
				<div class="grid grid-cols-3 gap-2" role="group" aria-label="选择主题">
					{#each THEME_OPTIONS as item}
						{@const selected = theme.theme === item.id}
						<button
							type="button"
							onclick={() => setTheme(item.id)}
							aria-pressed={selected}
							class="rounded-md border p-2.5 text-left outline-none transition-all duration-standard focus-visible:ring-2 focus-visible:ring-accent-default {selected
								? 'border-accent-border bg-accent-soft text-text-accent-on-soft shadow-raised-sm'
								: 'border-border-default bg-surface-primary text-text-secondary hover:bg-interactive-hover hover:text-text-primary'}"
						>
							<div class="flex items-center justify-between mb-2">
								<span
									class="block h-5 w-5 rounded-full ring-2 ring-offset-1 ring-offset-surface-primary"
									style="background:{item.swatch};--tw-ring-color:{selected ? 'var(--cst-accent-default)' : 'var(--cst-border-default)'};"
								></span>
								{#if selected}
									<Check size={13} class="text-accent-default" />
								{/if}
							</div>
							<span class="block text-sm font-medium text-text-primary">{item.label}</span>
							<span class="mt-0.5 block text-xxs text-text-secondary">{item.note}</span>
						</button>
					{/each}
				</div>
			</section>

			<!-- 明暗三档（跟随系统/浅色/深色）：接入 TabBar segmented 形态，等宽底轨无粘连 -->
			<section
				aria-labelledby="color-scheme-label"
				class="rounded-lg ring-1 ring-border-default surface-tinted p-2.5 mt-2.5"
			>
				<div class="flex items-center justify-between pb-2 pt-0.5 px-0.5">
					<p id="color-scheme-label" class="eyebrow">明暗</p>
					<span class="text-xxs text-text-secondary">当前：{modeLabel}</span>
				</div>
				<TabBar
					variant="segmented"
					items={COLOR_SCHEME_OPTIONS.map((o) => ({ id: o.id, label: o.label }))}
					value={theme.colorScheme}
					onChange={(id) => setColorScheme(id as ColorSchemeMode)}
				/>
			</section>

	</Popover.Content>
</Popover.Root>
