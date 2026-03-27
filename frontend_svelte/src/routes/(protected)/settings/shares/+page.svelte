<script lang="ts">
	import { onMount } from 'svelte';
	import { shareService } from '$lib/api/services';
	import { usePagination } from '$lib/hooks/use-pagination.svelte';
	import { formatDateTime, formatRelativeTime } from '$lib/utils/format';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent } from '$lib/components/ui/card';
	import { Badge } from '$lib/components/ui/badge';
	import PageHeader from '$lib/components/layout/page-header.svelte';
	import EmptyState from '$lib/components/common/empty-state.svelte';
	import ListSkeleton from '$lib/components/common/list-skeleton.svelte';
	import LoadMoreButton from '$lib/components/common/load-more-button.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { isSuccess } from '$lib/api/client';
	import { SHARE_TYPE_BADGE_CLASS, SHARE_TYPE_LABEL } from '$lib/utils/content-labels';
	import type { ShareListItem, ShareType } from '$lib/types/api';
	import RefreshCw from 'lucide-svelte/icons/refresh-cw';
	import Share2 from 'lucide-svelte/icons/share-2';
	import AlertTriangle from 'lucide-svelte/icons/triangle-alert';
	import Ban from 'lucide-svelte/icons/ban';
	import ExternalLink from 'lucide-svelte/icons/external-link';
	import Loader2 from 'lucide-svelte/icons/loader-2';

	const shareTypeOptions: Array<{ value: ShareType | 'all'; label: string }> = [
		{ value: 'all', label: '全部' },
		{ value: 'user', label: SHARE_TYPE_LABEL.user },
		{ value: 'group', label: SHARE_TYPE_LABEL.group },
		{ value: 'moment', label: SHARE_TYPE_LABEL.moment },
		{ value: 'location', label: SHARE_TYPE_LABEL.location }
	];

	const pagination = usePagination<ShareListItem>({
		uniqueKey: (item) => item.share_id
	});

	let activeType = $state<ShareType | 'all'>('all');
	let revokingShareId = $state<string | null>(null);

	function getShareTitle(item: ShareListItem): string {
		return item.preview?.title || `${SHARE_TYPE_LABEL[item.share_type]}分享`;
	}

	function getShareSubtitle(item: ShareListItem): string {
		return item.preview?.subtitle || `分享码 ${item.share_code}`;
	}

	function getShareHref(item: ShareListItem): string {
		return `/share/${item.share_code}`;
	}

	async function loadShares(refresh = false) {
		await pagination.load(
			(cursor) =>
				shareService.listMyShares({
					share_type: activeType === 'all' ? undefined : activeType,
					pagination: { limit: 20, cursor }
				}),
			refresh
		);
	}

	async function revokeShare(item: ShareListItem) {
		if (item.is_revoked || revokingShareId) return;
		revokingShareId = item.share_id;
		try {
			const response = await shareService.revoke({ share_id: item.share_id });
			if (isSuccess(response)) {
				pagination.updateItem((current) =>
					current.share_id === item.share_id ? { ...current, is_revoked: true } : current
				);
				toast.success('已撤销分享');
			} else {
				toast.error(response.message || '撤销失败');
			}
		} catch {
			toast.error('撤销失败');
		} finally {
			revokingShareId = null;
		}
	}

	onMount(() => {
		loadShares(true);
	});
</script>

<svelte:head>
	<title>我的分享 - HotSpot</title>
</svelte:head>

<div class="mx-auto max-w-4xl space-y-5">
	<PageHeader title="我的分享" description="查看你创建过的分享，并管理其可访问状态">
		{#snippet actions()}
			<Button
				variant="outline"
				size="sm"
				class="rounded-2xl border-border/60 bg-background/70"
				onclick={() => loadShares(true)}
				disabled={pagination.loading}
			>
				<RefreshCw class="mr-2 h-4 w-4 {pagination.loading ? 'animate-spin' : ''}" />
				刷新
			</Button>
		{/snippet}
	</PageHeader>

	<div class="flex flex-wrap gap-2">
		{#each shareTypeOptions as option (option.value)}
			<Button
				variant={activeType === option.value ? 'default' : 'outline'}
				size="sm"
				class="rounded-full"
				disabled={pagination.loading}
				onclick={async () => {
					if (pagination.loading || activeType === option.value) return;
					activeType = option.value;
					await loadShares(true);
				}}
			>
				{option.label}
			</Button>
		{/each}
	</div>

	{#if pagination.loading}
		<ListSkeleton type="notification" count={4} layout="list" />
	{:else if pagination.error}
		<EmptyState
			icon={AlertTriangle}
			title="加载分享失败"
			description={pagination.error}
			actionText="重试"
			onAction={() => loadShares(true)}
		/>
	{:else if pagination.items.length === 0}
		<EmptyState
			icon={Share2}
			title="还没有分享记录"
			description={activeType === 'all' ? '你创建过的分享会显示在这里。' : `当前筛选下暂无${shareTypeOptions.find((option) => option.value === activeType)?.label ?? ''}分享。`}
		/>
	{:else}
		<div class="space-y-3">
			{#each pagination.items as item (item.share_id)}
				<Card class="rounded-3xl border border-border/60 bg-card/80 shadow-sm backdrop-blur-md">
					<CardContent class="space-y-4 p-4 sm:p-5">
						<div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
							<div class="min-w-0 flex-1 space-y-2">
								<div class="flex flex-wrap items-center gap-2">
									<Badge class={`rounded-full border ${SHARE_TYPE_BADGE_CLASS[item.share_type]}`}>
										{SHARE_TYPE_LABEL[item.share_type]}
									</Badge>
									{#if item.is_revoked}
										<Badge variant="outline" class="rounded-full border-destructive/30 text-destructive">
											已撤销
										</Badge>
									{:else}
										<Badge variant="outline" class="rounded-full">
											可访问
										</Badge>
									{/if}
								</div>
								<div>
									<h2 class="truncate text-base font-semibold text-foreground">{getShareTitle(item)}</h2>
									<p class="mt-1 text-sm text-muted-foreground">{getShareSubtitle(item)}</p>
								</div>
							</div>
							<div class="flex shrink-0 flex-wrap gap-2">
								<a href={getShareHref(item)} target="_blank" rel="noreferrer">
									<Button variant="outline" size="sm" class="rounded-2xl border-border/60 bg-background/70">
										<ExternalLink class="mr-2 h-4 w-4" />
										查看
									</Button>
								</a>
								<Button
									variant="outline"
									size="sm"
									class="rounded-2xl border-destructive/40 text-destructive hover:bg-destructive hover:text-destructive-foreground"
									onclick={() => revokeShare(item)}
									disabled={item.is_revoked || revokingShareId === item.share_id}
								>
									{#if revokingShareId === item.share_id}
										<Loader2 class="mr-2 h-4 w-4 animate-spin" />
									{:else}
										<Ban class="mr-2 h-4 w-4" />
									{/if}
									撤销
								</Button>
							</div>
						</div>

						<div class="grid gap-3 rounded-2xl border border-border/50 bg-background/60 px-4 py-3 text-sm sm:grid-cols-3">
							<div>
								<p class="text-xs text-muted-foreground">创建时间</p>
								<p class="mt-1 font-medium">{formatRelativeTime(item.created_at)}</p>
								<p class="mt-1 text-xs text-muted-foreground">{formatDateTime(item.created_at)}</p>
							</div>
							<div>
								<p class="text-xs text-muted-foreground">分享码</p>
								<p class="mt-1 font-mono text-sm font-medium uppercase tracking-[0.12em]">{item.share_code}</p>
							</div>
							<div>
								<p class="text-xs text-muted-foreground">有效期</p>
								<p class="mt-1 font-medium">{item.expires_at ? formatDateTime(item.expires_at) : '长期有效'}</p>
							</div>
						</div>
					</CardContent>
				</Card>
			{/each}
		</div>

		<LoadMoreButton
			loading={pagination.loadingMore}
			hasMore={pagination.hasMore}
			onLoadMore={() => loadShares(false)}
		/>
	{/if}
</div>
