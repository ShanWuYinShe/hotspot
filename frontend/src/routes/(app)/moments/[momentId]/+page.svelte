<script lang="ts">
	import { goto } from '$app/navigation';
	import { ArrowLeft, MoreHorizontal, Link2, PencilLine, Trash2, Flag, EyeOff } from '@lucide/svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import Skeleton from '@/components/data-display/Skeleton.svelte';
	import Button from '@/components/ui/button.svelte';
	import SkeletonList from '@/components/data-display/SkeletonList.svelte';
	import SplitPageLayout from '@/components/layout/page/SplitPageLayout.svelte';
	import * as DropdownMenu from '@/components/ui/dropdown-menu';
	import { platform } from '@/stores/platform.svelte';
	import { ROUTES } from '@/lib/routes';
	import MomentPostBody from './MomentPostBody.svelte';
	import MomentDetailRail from './shared/MomentDetailRail.svelte';
	import DeleteMomentDialog from './shared/DeleteMomentDialog.svelte';
	import CommentInputBar from './shared/CommentInputBar.svelte';
	import MomentEditDialog from '@/components/overlays/MomentEditDialog.svelte';
	import ReportDialog from '@/components/overlays/ReportDialog.svelte';
	import ConfirmDialog from '@/components/overlays/ConfirmDialog.svelte';
	import DetailHeader from '@/components/layout/page/DetailHeader.svelte';
	import { contentFade } from '@/lib/motion';
	import { smartBack } from '@/lib/smart-back';
	import { useMomentDetailData } from './shared/useMomentDetailData.svelte';

	// 动态详情：点赞/收藏/评论/复制链接/更多菜单 + 桌面右 Rail。
	// 数据状态与异步操作收敛至 useMomentDetailData。

	let { params }: { params: { momentId: string } } = $props();

	const d = useMomentDetailData(() => params.momentId);

	let moreMenuEl = $state<HTMLElement | null>(null);
	let commentInputEl = $state<HTMLInputElement | null>(null);

	const focusComments = () => {
		document.getElementById('moment-comments')?.scrollIntoView({ behavior: 'smooth', block: 'start' });
		commentInputEl?.focus();
	};

	const replyTo = (user: string) => {
		d.input = `@${user} `;
		commentInputEl?.focus();
	};
</script>

<svelte:head>
	<title>瞬间详情 · HotSpot</title>
</svelte:head>

<h1 class="sr-only">{d.moment.user ? `${d.moment.user} 的瞬间` : '瞬间详情'}</h1>

{#snippet commentComposer()}
	{#if d.replyTarget}
		<div class="mb-2 flex items-center justify-between gap-3 rounded-md bg-surface-subtle/70 px-3 py-1.5">
			<p class="truncate text-xs text-text-secondary font-medium">回复 @{d.replyTarget.user}</p>
			<button
				type="button"
				class="link-btn shrink-0 text-xs text-accent-default hover:underline"
				onclick={() => d.cancelReply()}
			>取消回复</button>
		</div>
	{/if}
	<CommentInputBar
		bind:input={d.input}
		bind:inputEl={commentInputEl}
		onSend={() => void d.handleComment()}
		variant={platform.platform === 'mobile' ? 'docked' : 'inline'}
	/>
{/snippet}

{#snippet moreMenu()}
	<DropdownMenu.Root bind:open={d.moreOpen}>
		<div bind:this={moreMenuEl} class="relative">
			<ActionIconButton icon={MoreHorizontal} label="更多操作" variant="ghost" tone="current" ariaExpanded={d.moreOpen} ariaHasPopup="menu" onClick={() => (d.moreOpen = !d.moreOpen)} />
		</div>
		<DropdownMenu.Content customAnchor={moreMenuEl}>
			{#if d.isAuthor}
				<DropdownMenu.Item onSelect={() => void d.openEditDialog()}>
					<PencilLine class="size-3.5 text-accent-default" strokeWidth={1.75} />
					编辑动态
				</DropdownMenu.Item>
				<DropdownMenu.Item class="text-status-danger hover:text-text-status-danger-on-soft data-[highlighted]:text-text-status-danger-on-soft hover:bg-danger-bg data-[highlighted]:bg-danger-bg" onSelect={() => { d.deleteOpen = true; d.moreOpen = false; }}>
					<Trash2 class="size-3.5" strokeWidth={1.75} />
					删除动态
				</DropdownMenu.Item>
			{:else if d.moment.id}
				<DropdownMenu.Item class="text-status-danger hover:text-text-status-danger-on-soft data-[highlighted]:text-text-status-danger-on-soft hover:bg-danger-bg data-[highlighted]:bg-danger-bg" onSelect={() => { d.reportTarget = { target_type: 'moment', target_id: params.momentId }; d.moreOpen = false; }}>
					<Flag class="size-3.5" strokeWidth={1.75} />
					举报动态
				</DropdownMenu.Item>
			{/if}
		</DropdownMenu.Content>
	</DropdownMenu.Root>
{/snippet}

{#snippet headerActions()}
	<button
		type="button"
		class="relative flex size-9 items-center justify-center rounded-full text-text-primary hover:bg-interactive-hover focus-ring"
		aria-label={d.shareStatus === 'copied' ? '已复制动态链接' : '分享动态'}
		onclick={() => void d.copyMomentLink()}
	>
		<Link2 class="size-4" strokeWidth={1.75} />
		{#if d.shareStatus === 'copied'}
			<span class="absolute -top-7 right-0 rounded-sm bg-surface-primary px-2 py-0.5 text-xxs font-medium text-text-primary shadow-raised-sm whitespace-nowrap">
				已复制
			</span>
		{/if}
	</button>
	{@render moreMenu()}
{/snippet}

<SplitPageLayout
	contentMaxWidth="var(--cst-content-detail)"
	edgeToEdgeMobile={true}
	overlayHeaderMobile={!d.loading && d.moment.images.length > 0}
	contentClassName="flex flex-col flex-1 min-h-full pt-0 sm:pt-1 pb-0 sm:pb-6"
>
	{#snippet header()}
		<DetailHeader
			category="发现动态"
			entityName={d.moment.user ? `@${d.moment.user} 的瞬间` : '瞬间详情'}
			author={d.moment.id && !d.loading ? {
				id: d.moment.authorId,
				name: d.moment.user || '用户',
				avatar: d.moment.avatar,
				location: d.moment.location,
				time: d.moment.time
			} : undefined}
			onBack={() => smartBack()}
			onAuthorClick={() => {
				if (d.moment.authorId) void goto(ROUTES.user(d.moment.authorId));
			}}
			trailing={headerActions}
		/>
	{/snippet}

	<p aria-live="polite" class="sr-only">
		{d.shareStatus === 'copied' ? '动态链接已复制' : d.shareStatus === 'failed' ? '无法复制动态链接' : ''}
	</p>

	<div class="flex min-h-full flex-1 flex-col" use:contentFade={d.loading}>
		{#if d.loading}
			<div
				class="mx-auto w-full max-w-4xl flex-1 flex flex-col"
				role="status"
				aria-busy="true"
				aria-label="加载动态…"
				data-skeleton-root
			>
				<div class="overflow-hidden rounded-none sm:rounded-lg bg-surface-island shadow-none sm:shadow-xs flex-1 flex flex-col">
					<Skeleton class="aspect-[4/3] w-full" />
					<div class="p-4 sm:p-5 md:p-7">
						<div class="flex items-center gap-3 pb-4">
							<Skeleton class="size-11 shrink-0 rounded-full" />
							<div class="flex min-w-0 flex-1 flex-col gap-1.5">
								<Skeleton class="w-32 rounded-sm" style="height: 0.875rem;" />
								<Skeleton class="w-24 rounded-sm" style="height: 0.6875rem;" />
							</div>
						</div>
						<div class="flex flex-col gap-2 pb-4">
							<Skeleton class="w-full rounded-sm" style="height: 0.875rem;" />
							<Skeleton class="w-11/12 rounded-sm" style="height: 0.875rem;" />
							<Skeleton class="w-3/4 rounded-sm" style="height: 0.875rem;" />
						</div>
						<div class="flex items-center gap-6 py-4">
							<Skeleton class="h-4 w-16 rounded-sm" />
							<Skeleton class="h-4 w-12 rounded-sm" />
						</div>
						<div class="border-t border-border-subtle/40 pt-6">
							<Skeleton class="mb-3 h-4 w-24 rounded-sm" />
							<SkeletonList rows={3} leading={36} decorative />
						</div>
					</div>
				</div>
			</div>
		{:else if d.loadError}
			<LoadError message={d.loadError} onRetry={() => d.reload()} />
		{:else if !d.moment.id}
			<EmptyState
				spotlight
				iconTone="accent"
				title="动态不存在"
				description="这条动态可能已被删除，或你没有查看权限。"
			>
				{#snippet icon()}
					<EyeOff class="size-7" strokeWidth={1.5} />
				{/snippet}
				{#snippet action()}
					<Button variant="soft" size="md" class="mt-1" onclick={() => smartBack()}>返回上一页</Button>
				{/snippet}
			</EmptyState>
		{:else}
			<MomentPostBody
				moment={d.moment}
				comments={d.apiComments}
				commentCount={d.commentCount}
				hasMoreComments={d.commentsNextCursor !== null}
				loadingMoreComments={d.commentsLoadingMore}
				liked={d.liked}
				saved={d.saved}
				newCommentId={d.newCommentId}
				newReplyId={d.newReplyId}
				onToggleLike={() => void d.toggleLike()}
				onToggleSave={() => void d.toggleSave()}
				onCommentClick={focusComments}
				onReply={(user, commentId) => d.startReply(user, commentId)}
				onExpandReplies={(commentId) => void d.expandReplies(commentId)}
				onToggleCommentLike={(id: string) => void d.toggleCommentLike(id)}
				onReportComment={(id: string) => (d.reportTarget = { target_type: 'comment', target_id: id })}
				onDeleteComment={(id: string) => (d.commentDeleteTarget = id)}
				onLoadMoreComments={() => void d.loadMoreComments()}
				commentComposer={platform.platform === 'mobile' ? undefined : commentComposer}
			/>
		{/if}
	</div>

	{#snippet footer()}
		{#if platform.platform === 'mobile'}
			<div class="w-full border-t border-border-default/60 bg-surface-primary/95 pb-[calc(0.5rem+env(safe-area-inset-bottom,0px))] backdrop-blur-md">
				{@render commentComposer()}
			</div>
		{/if}
	{/snippet}

	{#snippet rail()}
		<MomentDetailRail moment={d.moment} nearbyNow={d.nearbyNow} relatedMoments={d.relatedMoments} />
	{/snippet}
</SplitPageLayout>

<DeleteMomentDialog
	open={d.deleteOpen}
	deleting={d.deleting}
	onCancel={() => (d.deleteOpen = false)}
	onConfirm={() => void d.handleDelete()}
/>

<ConfirmDialog
	open={d.commentDeleteTarget !== null}
	title="删除这条评论？"
	description="删除后将显示「该评论已删除」，其下的回复保留。"
	confirmLabel={d.commentDeletePending ? '删除中…' : '确认删除'}
	loading={d.commentDeletePending}
	onCancel={() => (d.commentDeleteTarget = null)}
	onConfirm={() => void d.handleCommentDelete()}
/>

<ReportDialog
	open={d.reportTarget !== null}
	targetType={d.reportTarget?.target_type ?? 'moment'}
	targetId={d.reportTarget?.target_id ?? ''}
	targetLabel={d.reportTarget?.target_type === 'comment' ? '这条评论' : '这条动态'}
	onClose={() => (d.reportTarget = null)}
/>

<MomentEditDialog
	open={d.editOpen}
	momentId={params.momentId}
	initial={d.momentRaw}
	onClose={() => (d.editOpen = false)}
	onUpdated={d.handleUpdated}
/>
