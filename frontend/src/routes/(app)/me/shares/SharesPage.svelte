<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { platform } from '@/stores/platform.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import Badge from '@/components/data-display/Badge.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import InfiniteSentinel from '@/components/list/InfiniteSentinel.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import NeedsAuthNotice from '@/components/auth/NeedsAuthNotice.svelte';
	import ConfirmDialog from '@/components/overlays/ConfirmDialog.svelte';
	import SearchInput from '@/components/forms/SearchInput.svelte';
	import Button from '@/components/ui/button.svelte';
	import { fetchCreatedShares, revokeShare } from '@/api/client';
	import type { ShareListItem, ShareType } from '@/api/types';
	import { copyText } from '@/lib/clipboard';
	import { formatMomentTime } from '@/lib/utils';
	import { shareAvailability, shareAvailabilityLabel, shareLink, filterShares, SHARE_TYPE_LABELS } from '@/lib/shares';
	import { showToast } from '@/stores/toast.svelte';
	import { smartBack } from '@/lib/smart-back';
	import { ROUTES } from '@/lib/routes';
	import { Link2, Share2, Trash2 } from '@lucide/svelte';

	// 我的分享（/me/shares）：消费 /api/v1/share/fetch_created（spec api_spec_share.md），
	// 补齐 Web 端「分享码创建后不可管理」的缺口（鸿蒙/Android 已有对应页面）。
	// 口径：is_revoked=true = 当前不可用（已撤销或已过期，服务端同口径且 preview=null）。

	type SharesTab = 'all' | ShareType;

	const TABS: { id: SharesTab; label: string }[] = [
		{ id: 'all', label: '全部' },
		{ id: 'moment', label: '瞬间' },
		{ id: 'user', label: '用户' },
		{ id: 'group', label: '群组' },
		{ id: 'location', label: '位置' }
	];

	function isTab(value: string | null): value is SharesTab {
		return value === 'all' || value === 'moment' || value === 'user' || value === 'group' || value === 'location';
	}

	const activeTab = $derived<SharesTab>(
		isTab(page.url.searchParams.get('tab')) ? (page.url.searchParams.get('tab') as SharesTab) : 'all'
	);
	const typeParam = $derived(activeTab === 'all' ? null : (activeTab as ShareType));

	// W19：本地搜索（契约无 keyword，见 lib/shares.filterShares）；词写入 ?q= 便于刷新/返回保持
	let q = $state(page.url.searchParams.get('q') ?? '');
	let items = $state<ShareListItem[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let needsAuth = $state(false);
	let nextCursor = $state<string | null>(null);
	let loadingMore = $state(false);
	let reloadKey = $state(0);

	const filteredItems = $derived(filterShares(items, q));

	function onSearch(value: string): void {
		q = value;
		if (typeof window === 'undefined') return;
		const url = new URL(window.location.href);
		if (value.trim()) url.searchParams.set('q', value);
		else url.searchParams.delete('q');
		window.history.replaceState({}, '', url);
	}

	let revokeTarget = $state<ShareListItem | null>(null);
	let revokeLoading = $state(false);

	const device = $derived(platform.platform);

	function switchTab(tab: SharesTab) {
		if (tab === activeTab) return;
		void goto(tab === 'all' ? ROUTES.meShares : `${ROUTES.meShares}?tab=${tab}`);
	}

	// 请求序号：切 tab（或快速重复触发拉取）时丢弃过期响应，避免旧类型数据覆盖新 tab、
	// 旧请求的 finally 提前结束 loading（W12）。
	let loadSeq = 0;

	async function loadPage(reset: boolean) {
		const seq = ++loadSeq;
		if (reset) {
			loading = true;
			loadError = null;
			needsAuth = false;
		} else {
			loadingMore = true;
		}
		try {
			const res = await fetchCreatedShares({
				share_type: typeParam,
				pagination: { limit: 20, cursor: reset ? undefined : (nextCursor ?? undefined) }
			});
			if (seq !== loadSeq) return;
			if (res.code === 10100) {
				needsAuth = true;
				items = [];
				nextCursor = null;
				return;
			}
			if (res.code !== 0 || !res.data) {
				loadError = /token|令牌/i.test(res.message ?? '') ? '登录状态已过期，请重新登录' : (res.message ?? '加载失败，请稍后重试');
				if (reset) items = [];
				return;
			}
			items = reset ? res.data.items : [...items, ...res.data.items];
			nextCursor = res.data.next_cursor;
		} catch {
			if (seq !== loadSeq) return;
			loadError = '加载失败，请稍后重试';
			if (reset) items = [];
		} finally {
			if (seq === loadSeq) {
				loading = false;
				loadingMore = false;
			}
		}
	}

	$effect(() => {
		void reloadKey;
		void activeTab;
		void loadPage(true);
	});

	async function copy(share: ShareListItem) {
		const ok = await copyText(shareLink(share.share_code, location.origin));
		showToast(ok ? '分享链接已复制' : '复制失败，请手动复制', ok ? 'success' : 'danger');
	}

	async function confirmRevoke() {
		const target = revokeTarget;
		if (!target) return;
		revokeLoading = true;
		try {
			const res = await revokeShare({ share_id: target.share_id, share_code: null });
			if (res.code === 0) {
				items = items.map((i) => (i.share_id === target.share_id ? { ...i, is_revoked: true, preview: null } : i));
				showToast('分享已撤销', 'success');
				revokeTarget = null;
			} else if (res.code === 10100) {
				needsAuth = true;
				revokeTarget = null;
			} else {
				showToast(res.message ?? '撤销失败，请稍后重试', 'danger');
			}
		} catch {
			showToast('撤销失败，请稍后重试', 'danger');
		} finally {
			revokeLoading = false;
		}
	}
</script>

<svelte:head>
	<title>我的分享 · HotSpot</title>
</svelte:head>

<StandardPageLayout contentMaxWidth="var(--cst-content-breath)" contentClassName="pt-4 pb-6">
	{#snippet header()}
		<PageHeader
			variant="integrated"
			title="我的分享"
			islandMeta={TABS.find((t) => t.id === activeTab)?.label ?? '分享管理'}
			maxWidth="var(--cst-content-breath)"
			titleSize={device === 'desktop' ? 'xl' : '2xl'}
			onBack={() => smartBack(ROUTES.settings)}
		>
			{#snippet islandIcon()}
				<Share2 class="size-3.5 text-accent-default" strokeWidth={1.75} />
			{/snippet}
			{#snippet below()}
				<TabBar items={TABS} value={activeTab} onChange={(id) => switchTab(id as SharesTab)} />
			{/snippet}
		</PageHeader>
	{/snippet}

	{#if !needsAuth && !loading && (items.length > 0 || q)}
		<div class="pb-3">
			<SearchInput
				value={q}
				onChange={onSearch}
				placeholder="搜索分享标题、类型或分享码"
				ariaLabel="搜索我的分享"
			/>
		</div>
	{/if}

	{#if needsAuth}
		<NeedsAuthNotice spotlight title="登录后查看" description="登录后即可管理你创建的分享链接。" />
	{:else if loading}
		<InlineLoading label="加载分享记录…" />
	{:else if loadError}
		<LoadError message={loadError} onRetry={() => (reloadKey += 1)} />
	{:else if items.length === 0}
		<EmptyState
			pageCentered
			spotlight
			iconTone="accent"
			title={activeTab === 'all' ? '还没有分享记录' : '该类型下没有分享'}
			description={activeTab === 'all'
				? '在瞬间、群组或个人主页点击「分享」生成的链接会在这里集中管理。'
				: '换个类型看看，或回到全部。'}
		>
			{#snippet icon()}
				<Share2 class="size-7" strokeWidth={1.5} />
			{/snippet}
		</EmptyState>
	{:else if filteredItems.length === 0}
		<EmptyState
			pageCentered
			spotlight
			iconTone="accent"
			title="没有匹配的分享"
			description="只搜索已加载的分享记录，换个关键词试试。"
		>
			{#snippet icon()}
				<Share2 class="size-7" strokeWidth={1.5} />
			{/snippet}
		</EmptyState>
	{:else}
		<ListContainer bordered class="overflow-hidden" ariaLabel="分享记录列表">
			{#each filteredItems as item (item.share_id)}
				{@const availability = shareAvailability(item)}
				<ListItem
					class="px-3 py-3"
					ariaLabel={`${SHARE_TYPE_LABELS[item.share_type]}分享：${item.preview?.title ?? '内容已不可访问'}，${shareAvailabilityLabel(item)}`}
				>
					<div class="flex min-w-0 flex-1 items-start gap-3 py-1">
						<div class="flex size-12 shrink-0 items-center justify-center overflow-hidden rounded-md bg-surface-subtle">
							{#if item.preview?.cover_url}
								<img src={item.preview.cover_url} alt="" class="size-full object-cover" loading="lazy" />
							{:else}
								<Share2 class="size-5 text-text-secondary" strokeWidth={1.5} />
							{/if}
						</div>
						<div class="flex min-w-0 flex-1 flex-col gap-1">
							<div class="flex items-center gap-2">
								<span class="min-w-0 flex-1 truncate text-sm font-medium text-text-primary">
									{item.preview?.title ?? '内容已不可访问'}
								</span>
								<Badge size="status" class="ml-auto shrink-0" tone={availability === 'active' ? 'online' : 'neutral'}>
									{shareAvailabilityLabel(item)}
								</Badge>
							</div>
							<p class="text-xs text-text-secondary">
								{SHARE_TYPE_LABELS[item.share_type]}
								{#if item.preview?.subtitle} · {item.preview.subtitle}{/if}
							</p>
							<p class="text-xs text-text-secondary">
								{formatMomentTime(item.created_at)}
								{#if item.expires_at} · 有效期至 {formatMomentTime(item.expires_at)}{/if}
							</p>
							<div class="mt-1 flex flex-wrap items-center gap-2">
								<Button
									type="button"
									variant="soft"
									size="sm"
									shape="pill"
									class="hit-slop-44 h-7 gap-1 px-2.5 text-xs"
									disabled={availability !== 'active'}
									onclick={() => void copy(item)}
								>
									<Link2 class="size-3.5" strokeWidth={1.75} />
									复制链接
								</Button>
								<Button
									type="button"
									variant="ghost"
									size="sm"
									shape="pill"
									class="hit-slop-44 h-7 gap-1 px-2.5 text-xs text-text-secondary"
									disabled={availability !== 'active'}
									onclick={() => (revokeTarget = item)}
								>
									<Trash2 class="size-3.5" strokeWidth={1.75} />
									撤销
								</Button>
							</div>
						</div>
					</div>
				</ListItem>
			{/each}
		</ListContainer>
		{#if nextCursor}
			<div class="py-2">
				<InfiniteSentinel
					hasMore={Boolean(nextCursor)}
					loading={loadingMore}
					error={Boolean(loadError)}
					onLoadMore={() => void loadPage(false)}
				/>
			</div>
		{:else}
			<ListEnd />
		{/if}
	{/if}
</StandardPageLayout>

<ConfirmDialog
	open={Boolean(revokeTarget)}
	title="撤销这条分享？"
	description="撤销后分享链接立即失效，已复制出去的链接也无法再打开；此操作不可恢复。"
	icon={Trash2}
	confirmLabel="撤销分享"
	loading={revokeLoading}
	onCancel={() => (revokeTarget = null)}
	onConfirm={() => void confirmRevoke()}
/>
