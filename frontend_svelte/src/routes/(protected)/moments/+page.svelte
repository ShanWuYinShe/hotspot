<script lang="ts">
	import { onMount } from 'svelte';
	import { momentsService } from '$lib/api/services';
	import { usePagination } from '$lib/hooks/use-pagination.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Card } from '$lib/components/ui/card';
	import PageHeader from '$lib/components/layout/page-header.svelte';
	import ListSkeleton from '$lib/components/common/list-skeleton.svelte';
	import EmptyState from '$lib/components/common/empty-state.svelte';
	import LoadMoreButton from '$lib/components/common/load-more-button.svelte';
	import CreateMomentModal from '$lib/components/common/create-moment-modal.svelte';
	import type { MomentItem } from '$lib/types/api';
	import Plus from 'lucide-svelte/icons/plus';
	import Image from 'lucide-svelte/icons/image';
	import RefreshCw from 'lucide-svelte/icons/refresh-cw';
	import AlertTriangle from 'lucide-svelte/icons/triangle-alert';

	const pagination = usePagination<MomentItem>({
		uniqueKey: 'moment_id'
	});
	let showCreateMoment = $state(false);

	async function loadMoments(refresh = false) {
		await pagination.load(
			(cursor) => momentsService.getMyMoments({ pagination: { limit: 20, cursor } }),
			refresh
		);
	}

	onMount(() => {
		loadMoments(true);
	});
</script>

<svelte:head>
	<title>我的瞬间 - HotSpot</title>
</svelte:head>

<div class="space-y-5">
	<PageHeader title="我的瞬间" description="记录和管理你发布的动态内容">
		{#snippet actions()}
			<div class="flex gap-2">
				<Button
					variant="outline"
					size="sm"
					class="h-10 rounded-2xl border-border/60 bg-background/70"
					onclick={() => loadMoments(true)}
					disabled={pagination.loading}
				>
					<RefreshCw class="mr-2 h-4 w-4 {pagination.loading ? 'animate-spin' : ''}" />
					刷新
				</Button>
				<Button size="sm" class="h-10 rounded-2xl" onclick={() => (showCreateMoment = true)}>
					<Plus class="mr-2 h-4 w-4" />
					发布
				</Button>
			</div>
		{/snippet}
	</PageHeader>

	{#if pagination.loading}
		<ListSkeleton type="moment" count={3} layout="grid" />
	{:else if pagination.error}
		<EmptyState 
			icon={AlertTriangle} 
			title="加载失败" 
			description={pagination.error} 
			actionText="重试"
			onAction={() => loadMoments(true)}
		/>
	{:else if pagination.items.length === 0}
		<EmptyState 
			icon={Image} 
			title="还没有瞬间" 
			description="发布你的第一个瞬间吧" 
			actionText="发布瞬间"
			onAction={() => (showCreateMoment = true)}
		/>
	{:else}
		<div class="space-y-4">
			<div class="columns-1 gap-4 sm:columns-2 xl:columns-3">
				{#each pagination.items as moment (moment.moment_id)}
					<div class="mb-4 break-inside-avoid">
						<Card class="overflow-hidden rounded-3xl border border-border/60 bg-card/85 shadow-sm backdrop-blur-md">
							<a href="/moments/{moment.moment_id}" class="block p-4 hover:bg-accent/50">
								<p class="line-clamp-3 whitespace-pre-wrap">{moment.description}</p>
								{#if moment.images && moment.images.length > 0}
									<div class="mt-3 flex gap-2 overflow-x-auto">
										{#each moment.images.slice(0, 4) as image, i (image.url)}
											<img
												src={image.url}
												alt="图片 {i + 1}"
												class="h-20 w-20 rounded object-cover"
											/>
										{/each}
										{#if moment.images.length > 4}
											<div class="flex h-20 w-20 items-center justify-center rounded bg-muted text-sm text-muted-foreground">
												+{moment.images.length - 4}
											</div>
										{/if}
									</div>
								{/if}
								<div class="mt-3 flex items-center gap-4 text-sm text-muted-foreground">
									<span>{moment.likes_count} 点赞</span>
									<span>{moment.comments_count} 评论</span>
									<span>{moment.shares_count} 分享</span>
									<span>{new Date(moment.created_at).toLocaleDateString('zh-CN')}</span>
								</div>
							</a>
						</Card>
					</div>
				{/each}
			</div>

			<LoadMoreButton
				loading={pagination.loadingMore}
				hasMore={pagination.hasMore}
				onLoadMore={() => loadMoments(false)}
			/>
		</div>
	{/if}
</div>

<CreateMomentModal bind:open={showCreateMoment} onCreated={() => loadMoments(true)} />
