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
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import { listReports } from '@/api/client';
	import type { ReportItem, ReportStatus } from '@/api/types';
	import { formatDateTime } from '@/lib/utils';
	import { smartBack } from '@/lib/smart-back';
	import { ROUTES } from '@/lib/routes';

	// 我的举报（/me/reports）：消费 /api/v1/reports/fetch（spec api_spec_report.md），
	// 让举报人看到处理进度与结果（resolution）。此前举报提交后无任何结果可见入口，
	// 属功能完整性缺口。样板对齐 nearby/moments / relationships：
	// StandardPageLayout + sticky PageHeader + below TabBar（?tab= 深链）。

	type ReportsTab = 'all' | ReportStatus;

	const TABS: { id: ReportsTab; label: string }[] = [
		{ id: 'all', label: '全部' },
		{ id: 'pending', label: '待处理' },
		{ id: 'resolved', label: '已处理' },
		{ id: 'dismissed', label: '已驳回' }
	];

	const TARGET_LABELS: Record<ReportItem['target_type'], string> = {
		user: '用户',
		moment: '瞬间',
		comment: '评论',
		message: '消息',
		group: '群组'
	};

	const REASON_LABELS: Record<ReportItem['reason'], string> = {
		spam: '垃圾信息',
		harassment: '骚扰',
		violence: '暴力',
		hate_speech: '仇恨言论',
		sexual_content: '色情内容',
		misinformation: '不实信息',
		impersonation: '冒充他人',
		self_harm: '自伤',
		other: '其他'
	};

	const STATUS_LABELS: Record<ReportStatus, string> = {
		pending: '待处理',
		resolved: '已处理',
		dismissed: '已驳回'
	};

	const device = $derived(platform.platform);

	const activeTab = $derived<ReportsTab>(
		isTab(page.url.searchParams.get('tab')) ? (page.url.searchParams.get('tab') as ReportsTab) : 'all'
	);
	function isTab(value: string | null): value is ReportsTab {
		return value === 'all' || value === 'pending' || value === 'resolved' || value === 'dismissed';
	}

	let items = $state<ReportItem[]>([]);
	let nextCursor = $state<string | null>(null);
	let loading = $state(true);
	let loadingMore = $state(false);
	let loadError = $state<string | null>(null);
	let reloadKey = $state(0);

	// status 过滤参数：all → 不传；其余 tab 直传对应状态
	const statusParam = $derived(activeTab === 'all' ? null : activeTab);

	$effect(() => {
		void reloadKey;
		void activeTab;
		let cancelled = false;
		async function fetchData() {
			loading = true;
			loadError = null;
			try {
				const res = await listReports({ status: statusParam, pagination: { limit: 20 } });
				if (cancelled) return;
				if (res.code !== 0) {
					loadError = res.message ?? '加载失败，请稍后重试';
					items = [];
					nextCursor = null;
					return;
				}
				items = res.data.items;
				nextCursor = res.data.next_cursor;
			} catch {
				if (!cancelled) {
					loadError = '加载失败，请稍后重试';
					items = [];
					nextCursor = null;
				}
			} finally {
				if (!cancelled) loading = false;
			}
		}
		void fetchData();
		return () => {
			cancelled = true;
		};
	});

	async function loadMore() {
		if (!nextCursor || loadingMore) return;
		loadingMore = true;
		try {
			const res = await listReports({
				status: statusParam,
				pagination: { limit: 20, cursor: nextCursor }
			});
			if (res.code === 0) {
				items = [...items, ...res.data.items];
				nextCursor = res.data.next_cursor;
			}
		} finally {
			loadingMore = false;
		}
	}

	const switchTab = (tab: string) => {
		if (tab === activeTab) return;
		void goto(tab === 'all' ? ROUTES.meReports : `${ROUTES.meReports}?tab=${tab}`);
	};
</script>

<svelte:head>
	<title>我的举报 · HotSpot</title>
</svelte:head>

<StandardPageLayout contentClassName="animate-fade-in pt-4 pb-6">
	{#snippet header()}
		<PageHeader
			title="我的举报"
			sticky
			titleSize={device === 'desktop' ? 'xl' : '2xl'}
			onBack={() => smartBack(ROUTES.me)}
		>
			{#snippet trailing()}
				{#if !loading && !loadError}
					<span class="text-text-secondary text-xs" aria-live="polite">{items.length} 条</span>
				{/if}
			{/snippet}
			{#snippet below()}
				<div class="mt-3">
					<TabBar
						items={TABS}
						value={activeTab}
						onChange={(id) => switchTab(id)}
					/>
				</div>
			{/snippet}
		</PageHeader>
	{/snippet}

	{#if loading}
		<InlineLoading label="加载举报记录…" />
	{:else if loadError}
		<LoadError message={loadError} onRetry={() => reloadKey++} />
	{:else if items.length === 0}
		<EmptyState
			bare
			title="暂无举报记录"
			description="在动态、评论或聊天中遇到违规内容时，可通过「举报」提交，处理结果会在这里同步。"
		/>
	{:else}
		<ListContainer bordered ariaLabel="举报记录列表">
			{#each items as item (item.report_id)}
				<ListItem ariaLabel={`${STATUS_LABELS[item.status]}：举报${TARGET_LABELS[item.target_type]}`}>
					<div class="flex min-w-0 flex-1 flex-col gap-1 py-1">
						<div class="flex items-center gap-2">
							<span class="text-sm font-medium text-text-primary truncate">
								举报{TARGET_LABELS[item.target_type]} · {REASON_LABELS[item.reason]}
							</span>
							<span
								class="inline-flex h-5 shrink-0 items-center justify-center px-2 leading-none rounded-full text-xxs {item.status ===
								'pending'
									? 'bg-accent-soft text-accent-default'
									: 'bg-surface-subtle text-text-secondary'}"
							>
								{STATUS_LABELS[item.status]}
							</span>
						</div>
						{#if item.description}
							<p class="text-xs text-text-secondary line-clamp-2">{item.description}</p>
						{/if}
						{#if item.resolution}
							<p class="text-xs text-text-secondary">
								处理结果：{item.resolution.action === 'resolve' ? '已确认处理' : '未构成违规'}{item.resolution.note
									? ` · ${item.resolution.note}`
									: ''}
							</p>
						{/if}
						<p class="text-xxs text-text-secondary opacity-dim">
							{formatDateTime(item.created_at)}
						</p>
					</div>
				</ListItem>
			{/each}
			{#if nextCursor}
				<ListItem
					onClick={() => void loadMore()}
					ariaLabel={loadingMore ? '加载中' : '加载更多'}
				>
					<div class="flex flex-1 items-center justify-center py-3">
						{#if loadingMore}
							<span class="spinner size-5"></span>
						{:else}
							<span class="text-sm text-text-secondary">加载更多</span>
						{/if}
					</div>
				</ListItem>
			{:else}
				<ListEnd />
			{/if}
		</ListContainer>
	{/if}
</StandardPageLayout>
