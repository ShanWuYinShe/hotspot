<script lang="ts">
	import { Check, MapPin, Monitor, Moon, Palette, Sun } from '@lucide/svelte';
	import { cn } from '@/lib/utils';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import SettingsSection from './SettingsSection.svelte';
	import SettingsControlRow from './SettingsControlRow.svelte';
	import {
		theme,
		setTheme,
		setColorScheme,
		THEME_OPTIONS,
		COLOR_SCHEME_OPTIONS,
		type ColorSchemeMode,
		type AppTheme
	} from '@/stores/theme.svelte';
	import {
		mapSourceStore,
		setMapSource,
		MAP_SOURCE_OPTIONS,
		initMapSource,
		type MapSource
	} from '@/stores/mapSource.svelte';

	$effect(() => initMapSource());

	const themeNote = $derived(THEME_OPTIONS.find((t) => t.id === theme.theme)?.note ?? '');
	const colorSchemeDesc = $derived(
		theme.colorScheme === 'auto'
			? '跟随系统外观'
			: theme.colorScheme === 'dark'
				? '当前使用深色外观'
				: '当前使用浅色外观'
	);
	const mapSourceDesc = $derived(
		mapSourceStore.source === 'amap' ? '高德地图 · 国内地点更贴合' : 'OpenStreetMap · 全球开源数据'
	);
</script>

<!-- 外观与显示 -->
<SettingsSection title="外观与显示" description="自定义界面的视觉色彩与深浅模式，全局即时生效">
	<ListContainer bordered inset class="overflow-hidden">
		<SettingsControlRow
			title="应用主题"
			description={themeNote ? `选择全局视觉色彩风格 · ${themeNote}` : '选择全局视觉色彩风格'}
		>
			{#snippet icon()}
				<Palette class="size-4 text-accent-default" strokeWidth={1.5} />
			{/snippet}
			{#snippet control()}
				<div class="grid grid-cols-3 gap-2 sm:gap-3" role="radiogroup" aria-label="选择全局视觉色彩风格">
					{#each THEME_OPTIONS as opt}
						{@const isSelected = theme.theme === opt.id}
						{@const details = opt.id === 'vibrant'
							? { desc: '现代深空 · 冷冽通透', preview: ['#0B0E14', '#131924', '#4C82F7'] }
							: opt.id === 'forest'
								? { desc: '自然纸面 · 平实温润', preview: ['#F6F7F3', '#EAECE4', '#39714F'] }
								: { desc: '宣纸朱砂 · 东方意境', preview: ['#F5F2EA', '#EAE5D8', '#A63A2E'] }}
						<button
							type="button"
							role="radio"
							aria-checked={isSelected}
							onclick={() => setTheme(opt.id as AppTheme)}
							class={cn(
								'group relative flex flex-col justify-between rounded-xl p-2.5 sm:p-3 text-left transition-all duration-rapid focus-ring hit-slop-44',
								isSelected
									? 'bg-surface-island ring-2 ring-accent-default shadow-raised-sm'
									: 'bg-surface-subtle/70 border border-border-default/60 hover:bg-surface-island hover:border-accent-border/60 hover:shadow-xs'
							)}
						>
							<div class="flex items-center justify-between gap-1.5 sm:gap-2">
								<div class="flex items-center gap-1 rounded-full bg-surface-primary/90 px-1.5 py-0.5 sm:px-2 sm:py-1 ring-1 ring-border-default/40 transition-transform duration-standard group-hover:scale-105">
									<span class="size-2 sm:size-2.5 rounded-full shrink-0 shadow-2xs" style="background-color: {details.preview[2]};" aria-hidden="true"></span>
									<span class="size-1.5 sm:size-2 rounded-full shrink-0 opacity-80" style="background-color: {details.preview[1]};" aria-hidden="true"></span>
									<span class="size-1.5 sm:size-2 rounded-full shrink-0 opacity-50" style="background-color: {details.preview[0]};" aria-hidden="true"></span>
								</div>
								{#if isSelected}
									<span class="flex size-4 sm:size-4.5 items-center justify-center rounded-full bg-accent-default text-content-inverse shadow-2xs shrink-0" aria-hidden="true">
										<Check class="size-2.5 sm:size-3" strokeWidth={2.5} />
									</span>
								{:else}
									<span class="size-4 sm:size-4.5 rounded-full border border-border-default/60 group-hover:border-accent-default/60 group-hover:bg-accent-soft/30 transition-colors shrink-0" aria-hidden="true"></span>
								{/if}
							</div>
							<div class="mt-2 min-w-0">
								<p class="text-xs sm:text-sm font-semibold truncate transition-colors {isSelected ? 'text-text-primary' : 'text-text-secondary group-hover:text-text-primary'}">
									{opt.label}
								</p>
								<p class="mt-0.5 hidden sm:block text-xxs leading-snug text-text-secondary">
									{details.desc}
								</p>
							</div>
						</button>
					{/each}
				</div>
			{/snippet}
		</SettingsControlRow>

		<SettingsControlRow title="外观模式" description={colorSchemeDesc}>
			{#snippet icon()}
				{#if theme.isDark}
					<Moon class="size-4 text-accent-default" strokeWidth={1.5} />
				{:else}
					<Sun class="size-4 text-accent-default" strokeWidth={1.5} />
				{/if}
			{/snippet}
			{#snippet control()}
				<div class="grid grid-cols-3 gap-2 sm:gap-3" role="radiogroup" aria-label="外观模式">
					{#each COLOR_SCHEME_OPTIONS as opt}
						{@const isSelected = theme.colorScheme === opt.id}
						{@const Icon = opt.id === 'dark' ? Moon : opt.id === 'light' ? Sun : Monitor}
						{@const note = opt.id === 'dark' ? '纯正深邃极夜' : opt.id === 'light' ? '清亮温润如昼' : '随系统自动切换'}
						<button
							type="button"
							role="radio"
							aria-checked={isSelected}
							onclick={() => setColorScheme(opt.id as ColorSchemeMode)}
							class={cn(
								'group relative flex flex-col justify-between rounded-xl p-2.5 sm:p-3 text-left transition-all duration-rapid focus-ring hit-slop-44',
								isSelected
									? 'bg-surface-island ring-2 ring-accent-default shadow-raised-sm'
									: 'bg-surface-subtle/70 border border-border-default/60 hover:bg-surface-island hover:border-accent-border/60 hover:shadow-xs'
							)}
						>
							<div class="flex items-center justify-between gap-1.5 sm:gap-2">
								<div class="flex size-6 sm:size-7 items-center justify-center rounded-lg bg-surface-primary/90 ring-1 ring-border-default/40 transition-colors duration-rapid {isSelected ? 'text-accent-default bg-accent-soft/30' : 'text-text-secondary group-hover:text-accent-default'} shrink-0">
									<Icon class="size-3.5 sm:size-4" strokeWidth={1.75} />
								</div>
								{#if isSelected}
									<span class="flex size-4 sm:size-4.5 items-center justify-center rounded-full bg-accent-default text-content-inverse shadow-2xs shrink-0" aria-hidden="true">
										<Check class="size-2.5 sm:size-3" strokeWidth={2.5} />
									</span>
								{:else}
									<span class="size-4 sm:size-4.5 rounded-full border border-border-default/60 group-hover:border-accent-default/60 group-hover:bg-accent-soft/30 transition-colors shrink-0" aria-hidden="true"></span>
								{/if}
							</div>
							<div class="mt-2 min-w-0">
								<p class="text-xs sm:text-sm font-semibold truncate transition-colors {isSelected ? 'text-text-primary' : 'text-text-secondary group-hover:text-text-primary'}">
									{opt.label}
								</p>
								<p class="mt-0.5 hidden sm:block text-xxs leading-snug text-text-secondary">
									{note}
								</p>
							</div>
						</button>
					{/each}
				</div>
			{/snippet}
		</SettingsControlRow>
	</ListContainer>
</SettingsSection>

<!-- 地图与位置 -->
<SettingsSection
	title="地图与位置"
	description="选择地图页使用的底图，两种底图都能看到附近的人与瞬间，差别在地点名称与街道的贴合程度。"
>
	<ListContainer bordered inset class="overflow-hidden">
		<SettingsControlRow title="地图源" description={mapSourceDesc}>
			{#snippet icon()}
				<MapPin class="size-4 text-accent-default" strokeWidth={1.5} />
			{/snippet}
			{#snippet control()}
				<div class="w-full sm:max-w-xs">
					<TabBar
						variant="segmented"
						size="md"
						items={MAP_SOURCE_OPTIONS.map((o) => ({ id: o.id, label: o.id === 'amap' ? '高德地图' : 'OpenStreetMap' }))}
						value={mapSourceStore.source}
						onChange={(id) => setMapSource(id as MapSource)}
					/>
				</div>
			{/snippet}
		</SettingsControlRow>
	</ListContainer>
</SettingsSection>
