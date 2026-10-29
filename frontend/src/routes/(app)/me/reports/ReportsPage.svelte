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
	import LoadError from '@/components/data-display/LoadError.svelte';
	import NeedsAuthNotice from '@/components/auth/NeedsAuthNotice.svelte';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import { listReports } from '@/api/client';
	import type { ReportItem, ReportStatus } from '@/api/types';
	import { formatMomentTime } from '@/lib/utils';
	import { ShieldCheck } from '@lucide/svelte';
	import { smartBack } from '@/lib/smart-back';
	import { ROUTES } from '@/lib/routes';

	// 我的举报（/me/reports）：消费 /api/v1/reports/fetch（spec api_spec_report.md），
	// 让举报人看到处理进度与结果（resolution）。此前举报提交后无任何结果可见入口，
	// 属功能完整性缺口。样板对齐 nearby/moments / relationships：
	// StandardPageLayout + integrated PageHeader + below TabBar（?tab= 深链）。

	type ReportsTab = 'all' | ReportStatus;

	// 状态胶囊 tone：待处理 warning 软底（需要用户/管理员处理）、已处理 online 软底（安全语义）、
	// 已驳回中性；颜色全部由 Badge tone 语义档表达（R12 口径：禁止淡底 + 同色语义字直配）。
	const STATUS_TONE: Record<ReportStatus, 'warning' | 'online' | 'neutral'> = {
		pending: 'warning',
		resolved: 'online',
		dismissed: 'neutral'
	};

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
	let needsAuth = $state(false);
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
			needsAuth = false;
			try {
				const res = await listReports({ status: statusParam, pagination: { limit: 20 } });
				if (cancelled) return;
				if (res.code === 10100) {
					// 未登录：给登录引导而非裸技术报错（对齐 notify/messages 鉴权表现）
					needsAuth = true;
					items = [];
					nextCursor = null;
					return;
				}
				if (res.code !== 0) {
					// 不向用户透出「Access Token」等技术术语（r70）：鉴权类改通俗文案
					const msg = res.message ?? '加载失败，请稍后重试';
					loadError = /token|令牌/i.test(msg) ? '登录状态已过期，请重新登录' : msg;
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

	// 鉴权失效类报错（token 过期/术语型 message）：与 10100 同走登录引导，
	// 避免「重试」死循环（Gemini UI 审查 r70——过期后点重试依然失败）
	const authExpired = $derived(
		needsAuth || (loadError !== null && /token|令牌|重新登录|登录状态/i.test(loadError))
	);

	const switchTab = (tab: string) => {
		if (tab === activeTab) return;
		void goto(tab === 'all' ? ROUTES.meReports : `${ROUTES.meReports}?tab=${tab}`);
	};
</script>

<svelte:head>
	<title>我的举报 · HotSpot</title>
</svelte:head>

<StandardPageLayout contentMaxWidth={1280} contentClassName="animate-fade-in pt-4 pb-6">
	{#snippet header()}
		<PageHeader
			variant="integrated"
			title="我的举报"
			maxWidth={1280}
			onBack={() => smartBack(ROUTES.me)}
		>
			{#snippet trailing()}
				{#if !loading && !loadError && !authExpired}
					<span class="text-text-secondary text-xs" aria-live="polite">{items.length} 条</span>
				{/if}
			{/snippet}
			{#snippet below()}
				<!-- 未登录/鉴权失效隐藏状态标签（Gemini UI 审查 r15/r70：展示不可用控件与其余个人子页鉴权表现不一） -->
				{#if !authExpired}
				<div>
					<TabBar
						items={TABS}
						value={activeTab}
						onChange={(id) => switchTab(id)}
					/>
				</div>
				{/if}
			{/snippet}
		</PageHeader>
	{/snippet}

	{#if loading}
		<InlineLoading label="加载举报记录…" />
	{:else if needsAuth}
		<!-- 10100 未登录：登录引导而非接口报错（r15：与其余个人子页鉴权表现一致） -->
		<div class="flex min-h-[40vh] items-center justify-center">
			<NeedsAuthNotice spotlight title="登录后查看" description="你提交的举报与处理进度会在这里同步。" />
		</div>
	{:else if authExpired && loadError}
		<!-- token 过期：去登录而非死循环重试（r70） -->
		<div class="flex min-h-[40vh] items-center justify-center">
			<NeedsAuthNotice
				spotlight
				title="登录状态已过期"
				description="你的登录状态已失效，重新登录后即可继续查看举报与处理进度。"
			/>
		</div>
	{:else if loadError}
		<LoadError message={loadError} onRetry={() => reloadKey++} />
	{:else if items.length === 0}
		<EmptyState
			spotlight
			iconTone="accent"
			title="暂无举报记录"
			description="在动态、评论或聊天中遇到违规内容时，可通过「举报」提交，处理结果会在这里同步。"
		>
			{#snippet icon()}
				<ShieldCheck class="size-7" strokeWidth={1.5} />
			{/snippet}
		</EmptyState>
	{:else}
		<ListContainer bordered class="overflow-hidden" ariaLabel="举报记录列表">
			{#each items as item (item.report_id)}
				<ListItem class="px-3 py-3" ariaLabel={`${STATUS_LABELS[item.status]}：举报${TARGET_LABELS[item.target_type]}`}>
					<div class="flex min-w-0 flex-1 flex-col gap-1 py-1">
						<!-- 标题 + 状态胶囊：胶囊固定行尾（ml-auto），不随标题长度漂移（R13 评审 P2-9） -->
						<div class="flex items-center gap-2">
							<span class="min-w-0 flex-1 truncate text-sm font-medium text-text-primary">
								举报{TARGET_LABELS[item.target_type]} · {REASON_LABELS[item.reason]}
							</span>
							<Badge class="ml-auto shrink-0" tone={STATUS_TONE[item.status]}>{STATUS_LABELS[item.status]}</Badge>
						</div>
						{#if item.description}
							<p class="text-xs text-text-secondary line-clamp-2">{item.description}</p>
						{/if}
						{#if item.resolution}
							<p class="text-xs leading-relaxed text-text-secondary">
								处理结果：
								<span class="font-medium text-text-primary">
									{item.resolution.action === 'resolve' ? '已确认处理' : '未构成违规'}
								</span>
								{#if item.resolution.note} · {item.resolution.note}{/if}
							</p>
						{/if}
						<!-- 时间行：原为 text-xxs(10px) + opacity-dim 且用绝对时间戳，与全站
						     时间戳档位（11px）与相对时间口径都不一致（全站复审 s3 P2-9）。 -->
						<p class="text-xs text-text-secondary">
							{formatMomentTime(item.created_at)}
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
			{/if}
		</ListContainer>
		{#if !nextCursor}
			<ListEnd />
		{/if}
	{/if}
</StandardPageLayout>