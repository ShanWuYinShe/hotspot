<script lang="ts">
	import { MapPin, X, ChevronRight, Clock, Check } from '@lucide/svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import SectionHead from '@/components/home/SectionHead.svelte';
	import Button from '@/components/ui/button.svelte';
	import { type PinType, type Pin, type LocationResult } from '../shared/location-data';
	import { PIN_COLORS, PIN_LABELS } from '../shared/location-shared';
	import PinMedia from '../shared/PinMedia.svelte';
	import ConfirmDialog from '@/components/overlays/ConfirmDialog.svelte';

	// 移动端地图「底部 Sheet 列表内容」：搜索建议 / 选中图钉 mini-card / 最近使用列表。
	// 页面私有组件（colocate），由 map/LocationMobile.svelte 渲染（仅 snap !== 'min' 时显示）。
	// 选中态经 $bindable 双向绑定；跨状态联动（选地点→移动地图中心）走 onSelectLoc 回调。
	// 原「热门地点」Tab 为静态演示数据脚手架，已按产品决定移除。
	//
	// R13：分区标题收口到 SectionHead（rail 变体），最近记录空态升级为带图标的
	// EmptyState（iconTone=accent），选中图钉卡与列表行的图标/按钮尺寸对齐。

	let {
		searchQuery = $bindable(''),
		searchResults = $bindable<LocationResult[]>([]),
		searching = false,
		selectedPin = $bindable<Pin | null>(null),
		selectedLoc = $bindable<LocationResult | null>(null),
		recentItems = $bindable<LocationResult[]>([]),
		onSelectLoc,
		onOpenPinDetail,
		onClearPin
	}: {
		searchQuery?: string;
		searchResults?: LocationResult[];
		/** 地理编码请求进行中（OSM 代理偶发数秒慢，需「搜索中」指示）。 */
		searching?: boolean;
		selectedPin?: Pin | null;
		selectedLoc?: LocationResult | null;
		recentItems?: LocationResult[];
		onSelectLoc: (loc: LocationResult) => void;
		onOpenPinDetail: (pin: Pin) => void;
		onClearPin: () => void;
	} = $props();

	// 清空最近记录不可恢复：走全站统一的 ui/dialog 危险操作确认
	// （原先用 window.confirm——原生弹窗无法随主题/明暗变化，与全站危险操作
	//   确认范式（settings 退出登录、me/edit 注销、groups 退群）割裂）。
	let confirmClearOpen = $state(false);
</script>

{#if searchQuery && searchResults.length > 0}
	<ListContainer class="animate-fade-in">
		{#each searchResults as r}
			{@const isSelected = selectedLoc?.displayName === r.displayName}
			<ListItem
				pressed={isSelected}
				onClick={() => onSelectLoc(r)}
				ariaLabel={`选择地点 ${r.displayName}`}
				class="flex list-row-standard items-center gap-3 px-4 py-2.5"
			>
				<span class="flex size-9 shrink-0 items-center justify-center rounded-full {isSelected ? 'bg-surface-primary' : 'bg-accent-soft'}">
					<MapPin class="size-3.5 text-accent-default" strokeWidth={1.75} />
				</span>
				<span class="{isSelected ? 'text-text-on-accent' : 'text-text-primary'} min-w-0 flex-1 truncate text-sm">{r.displayName}</span>
				{#if isSelected}
					<Check class="size-3.5 shrink-0 text-text-on-accent" strokeWidth={2.5} />
				{/if}
			</ListItem>
		{/each}
	</ListContainer>
{:else if searchQuery && searching}
	<!-- 搜索中指示（OSM 代理偶发数秒级慢响应，零反馈会被当成搜索失效） -->
	<EmptyState bare title="搜索中…" class="py-8">
		{#snippet icon()}
			<span class="spinner size-6" role="status" aria-label="搜索中"></span>
		{/snippet}
	</EmptyState>
{:else if searchQuery && searchResults.length === 0}
	<!-- 空态统一 EmptyState（bare + 紧凑 py-8：地图 sheet 不是整页，不套 py-20） -->
	<EmptyState bare title="未找到匹配地点" description="换个关键词或拖动地图再试试" class="py-8">
		{#snippet icon()}
			<MapPin size={24} class="text-text-secondary opacity-faint" strokeWidth={1.5} />
		{/snippet}
	</EmptyState>
{:else}
	<div class="animate-fade-in">
		<!-- Selected pin mini-card -->
		{#if selectedPin}
			<div class="mx-4 mb-3 flex items-center gap-3 rounded-md bg-surface-island p-2.5 shadow-xs">
				<PinMedia pin={selectedPin} size={42} />
				<div class="min-w-0 flex-1">
					<p class="truncate text-md font-medium text-text-primary">{selectedPin.name || selectedPin.title}</p>
					<p class="mt-0.5 truncate text-xs">
						<span class="font-medium" style="color:{PIN_COLORS[selectedPin.type]};">{PIN_LABELS[selectedPin.type]}</span>
						<span class="text-text-secondary">
							{selectedPin.distance ? ` · ${selectedPin.distance}` : ''}{selectedPin.tag ? ` · #${selectedPin.tag}` : ''}
						</span>
					</p>
				</div>
				<Button
					variant="surface-accent"
					size="sm"
					class="hit-slop shrink-0"
					onclick={() => onOpenPinDetail(selectedPin)}
				>
					查看
				</Button>
				<button
					type="button"
					class="hover-action -m-1 flex shrink-0 items-center justify-center rounded-full p-1 focus-ring"
					aria-label="清除选中地点"
					title="清除"
					onclick={onClearPin}
				>
					<X class="size-4 text-text-secondary" strokeWidth={1.75} />
				</button>
			</div>
		{/if}

		<!-- 最近使用（搜索选点记录） -->
		<div class="px-4">
			<SectionHead
				variant="rail"
				title="最近使用"
				count={recentItems.length > 0 ? recentItems.length : undefined}
				action={recentItems.length > 0 ? '清空' : undefined}
				onAction={() => (confirmClearOpen = true)}
			/>
		</div>

		<!-- List items -->
		{#if recentItems.length > 0}
			<ListContainer>
				{#each recentItems as loc}
					{@const isSelected = selectedLoc?.displayName === loc.displayName}
					<ListItem
						pressed={isSelected}
						onClick={() => onSelectLoc(loc)}
						ariaLabel={`选择最近地点 ${loc.displayName}`}
						class="flex list-row-standard items-center gap-3 px-4 py-2.5"
					>
						<div
							class="flex size-9 shrink-0 items-center justify-center rounded-full"
							style="background:{isSelected ? 'var(--cst-surface-primary)' : 'var(--cst-accent-soft)'};"
						>
							{#if loc.emoji}
								<span class="text-lg" aria-hidden="true">{loc.emoji}</span>
							{:else}
								<MapPin class="size-3.5 text-accent-default" strokeWidth={1.75} />
							{/if}
						</div>
						<div class="min-w-0 flex-1">
							<p class="truncate {isSelected ? 'text-text-on-accent' : 'text-text-primary'} text-sm">{loc.displayName}</p>
							{#if loc.desc}
								<p class="{isSelected ? 'text-text-on-accent opacity-emphasis' : 'text-text-secondary'} mt-0.5 truncate text-xs">{loc.desc}</p>
							{/if}
						</div>
						{#if isSelected}
							<Check class="size-3.5 shrink-0 text-text-on-accent" strokeWidth={2.5} />
						{:else}
							<ChevronRight class="size-3.5 shrink-0 text-text-secondary" strokeWidth={1.5} />
						{/if}
					</ListItem>
				{/each}
			</ListContainer>
		{:else}
			<EmptyState
				iconTone="accent"
				title="暂无最近记录"
				description="搜索并选择地点后，会记录在这里"
				iconSize={38}
				class="py-2"
			>
				{#snippet icon()}
					<Clock class="size-5" strokeWidth={1.5} />
				{/snippet}
			</EmptyState>
		{/if}
	</div>
{/if}

{#if confirmClearOpen}
	<ConfirmDialog
		open={confirmClearOpen}
		tone="danger"
		icon={Clock}
		title="清空最近记录？"
		description="此操作不可恢复，清空后需要重新搜索地点才会再次记录。"
		confirmLabel="清空"
		layout="stacked"
		onCancel={() => (confirmClearOpen = false)}
		onConfirm={() => {
			recentItems = [];
			confirmClearOpen = false;
		}}
	/>
{/if}
