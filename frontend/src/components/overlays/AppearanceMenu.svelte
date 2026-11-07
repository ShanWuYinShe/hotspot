<script lang="ts">
	import { Palette } from '@lucide/svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import { theme, setColorScheme, setTheme, THEME_OPTIONS, COLOR_SCHEME_OPTIONS, type ColorSchemeMode, type AppTheme } from '@/stores/theme.svelte';
	import * as Popover from '@/components/ui/popover';

	// 唯一的外观/模式控制。选择面板保持打开，仅显式完成、外部点击或 Escape 关闭。
	// 弹层行为现由 bits-ui Popover 提供。

	let { class: className = '', size = 36 }: { class?: string; size?: 32 | 34 | 36 | 42 } = $props();

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
			size={size}
			ariaExpanded={open}
			ariaHasPopup="dialog"
			class="header-action"
		/>
	</div>

	<Popover.Content customAnchor={rootEl} class="w-76 p-3" role="dialog" aria-label="外观设置">
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
				<!-- 主题选择与 /settings「应用主题」同一语言：TabBar segmented + swatch 色点
				     （选中 = accent 淡底 + semibold + ✓），不再自绘三卡片，避免两处表现漂移。 -->
				<TabBar
					variant="segmented"
					size="md"
					items={THEME_OPTIONS.map((t) => ({ id: t.id, label: t.label, swatch: t.swatch }))}
					value={theme.theme}
					onChange={(id) => setTheme(id as AppTheme)}
				/>
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
