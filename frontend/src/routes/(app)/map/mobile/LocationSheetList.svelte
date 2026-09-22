<script lang="ts">
	import { MapPin, X, ChevronRight, Clock, Check } from '@lucide/svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import { type PinType, type Pin, type LocationResult } from '../shared/location-data';
	import { PIN_COLORS, PIN_LABELS, PIN_TONES } from '../shared/location-shared';
import PinMedia from '../shared/PinMedia.svelte';
import * as Dialog from '@/components/ui/dialog';
	import Button from '@/components/ui/button.svelte';

	// 移动端地图「底部 Sheet 列表内容」：搜索建议 / 选中图钉 mini-card / 最近使用列表。
	// 页面私有组件（colocate），由 map/LocationMobile.svelte 渲染（仅 snap !== 'min' 时显示）。
	// 选中态经 $bindable 双向绑定；跨状态联动（选地点→移动地图中心）走 onSelectLoc 回调。
	// 原「热门地点」Tab 为静态演示数据脚手架，已按产品决定移除。

	let {
		searchQuery = $bindable(''),
		searchResults = $bindable<LocationResult[]>([]),
		selectedPin = $bindable<Pin | null>(null),
		selectedLoc = $bindable<LocationResult | null>(null),
		recentItems = $bindable<LocationResult[]>([]),
		onSelectLoc,
		onOpenPinDetail,
		onClearPin
	}: {
		searchQuery?: string;
		searchResults?: LocationResult[];
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
				class="flex list-row-standard items-center gap-3 px-4 py-3"
			>
				<div class="flex items-center justify-center rounded-full shrink-0 size-8.5 {isSelected ? 'bg-surface-primary' : 'bg-accent-soft'}">
					<MapPin class="size-3.5 text-accent-default"   strokeWidth={1.75} />
				</div>
				<span class="{isSelected ? 'text-text-on-accent' : 'text-text-primary'} text-sm"  style="flex:1">{r.displayName}</span>
				{#if isSelected}
					<Check class="size-3.5 shrink-0 text-text-on-accent"   strokeWidth={2.5} />
				{/if}
			</ListItem>
		{/each}
	</ListContainer>
{:else if searchQuery && searchResults.length === 0}
	<!-- 空态统一 EmptyState（bare + 紧凑 py-8：地图 sheet 不是整页，不套 py-20） -->
	<EmptyState bare title="未找到匹配地点" class="py-8">
		{#snippet icon()}
			<MapPin size={24} class="opacity-faint text-text-secondary"   strokeWidth={1.5} />
		{/snippet}
	</EmptyState>
{:else}
	<div class="animate-fade-in">
		<!-- Selected pin mini-card -->
		{#if selectedPin}
			<div class="flex items-center gap-3 mx-4 mb-2 p-3 rounded-md" style="background:{PIN_TONES[selectedPin.type].bg};border:var(--cst-border-width-hairline) solid {PIN_TONES[selectedPin.type].border};">
				<PinMedia pin={selectedPin} size={42} />
				<div class="flex-1 min-w-0">
					<p class="text-text-primary text-sm">{selectedPin.name || selectedPin.title}</p>
					<p class="text-text-secondary text-xs mt-px" >
						{PIN_LABELS[selectedPin.type]}
						{selectedPin.distance ? ` · ${selectedPin.distance}` : ''}
						{selectedPin.tag ? ` · #${selectedPin.tag}` : ''}
					</p>
				</div>
				<button type="button" class="hover-btn scale-btn px-2.5 py-1.5 rounded-full focus-ring shrink-0 text-sm" style="background:{PIN_COLORS[selectedPin.type]};color:var(--cst-content-on-marker);" onclick={() => onOpenPinDetail(selectedPin)}>
					查看
				</button>
				<button type="button" class="hover-btn hit-slop focus-ring shrink-0 ml-0.5" aria-label="清除选中地点" onclick={onClearPin}>
					<X class="size-3.5 text-text-secondary"   strokeWidth={2} />
				</button>
			</div>
		{/if}

		<!-- 最近使用（搜索选点记录） -->
		<div class="flex items-center justify-between px-4 pb-2">
			<span class="text-xs text-text-secondary">最近使用</span>
			{#if recentItems.length > 0}
				<button
					type="button"
					class="link-btn -my-1 px-2 py-1 focus-ring text-xs text-text-secondary hover:text-accent-default"
					onclick={() => (confirmClearOpen = true)}
				>
					清空
				</button>
			{/if}
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
						class="flex list-row-standard items-center gap-3 px-4 py-3"
					>
						<div class="flex items-center justify-center rounded-full shrink-0 bg-surface-primary size-9 border border-border-default shadow-raised-sm"  >
							{#if loc.emoji}
								<span class="text-lg" aria-hidden="true">{loc.emoji}</span>
							{:else}
								<MapPin class="size-3.5 text-accent-default" strokeWidth={1.75} />
							{/if}
						</div>
						<div class="flex-1 min-w-0">
							<p class="truncate {isSelected ? 'text-text-on-accent' : 'text-text-primary'} text-sm">{loc.displayName}</p>
							{#if loc.desc}
								<p class="{isSelected ? 'text-text-on-accent opacity-emphasis' : 'text-text-secondary'} text-xs mt-px" >{loc.desc}</p>
							{/if}
						</div>
						{#if isSelected}
							<Check class="size-3.5 shrink-0 text-text-on-accent"   strokeWidth={2.5} />
						{:else}
							<ChevronRight class="size-3.5 shrink-0 text-border-default" strokeWidth={1.5} />
						{/if}
					</ListItem>
				{/each}
			</ListContainer>
		{:else}
			<EmptyState bare title="暂无最近记录" class="py-8">
				{#snippet icon()}
					<Clock size={24} class="opacity-faint text-text-secondary"   strokeWidth={1.5} />
				{/snippet}
			</EmptyState>
		{/if}
	</div>
{/if}

{#if confirmClearOpen}
	<Dialog.Root bind:open={confirmClearOpen}>
		<Dialog.Content class="max-w-90 p-0">
			<div class="px-5 pt-6 pb-4">
				<div class="mb-3.5 flex size-10 items-center justify-center rounded-full bg-danger-bg text-status-danger ring-1 ring-status-danger-border">
					<Clock class="size-5" />
				</div>
				<Dialog.Title class="text-lg font-semibold tracking-title text-text-primary">清空最近记录？</Dialog.Title>
				<Dialog.Description class="mt-1 text-sm text-text-secondary leading-relaxed">此操作不可恢复，清空后需要重新搜索地点才会再次记录。</Dialog.Description>
			</div>
			<div class="flex flex-col gap-2 px-5 pb-5">
				<Button
					type="button"
					variant="danger"
					size="lg"
					class="w-full"
					onclick={() => {
						recentItems = [];
						confirmClearOpen = false;
					}}
				>清空</Button>
				<Button type="button" variant="secondary" size="lg" class="w-full" onclick={() => (confirmClearOpen = false)}>取消</Button>
			</div>
		</Dialog.Content>
	</Dialog.Root>
{/if}
