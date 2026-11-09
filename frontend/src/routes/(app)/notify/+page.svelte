<script lang="ts">
	import { goto } from '$app/navigation';
	import { ArrowUp, Bell, CheckCheck, Heart, UserPlus, Users } from '@lucide/svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import SplitPageLayout from '@/components/layout/page/SplitPageLayout.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import SkeletonList from '@/components/data-display/SkeletonList.svelte';
	import { contentFade } from '@/lib/motion';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import InfiniteSentinel from '@/components/list/InfiniteSentinel.svelte';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import Button from '@/components/ui/button.svelte';
	import SectionHead from '@/components/home/SectionHead.svelte';
	import NeedsAuthNotice from '@/components/auth/NeedsAuthNotice.svelte';
	import { ROUTES } from '@/lib/routes';
	import { smartBack } from '@/lib/smart-back';

	import { useNotifyData, FILTER_TABS } from './shared/useNotifyData.svelte';
	import NotifyRow from './shared/NotifyRow.svelte';
	import InvitesPendingCard from './shared/InvitesPendingCard.svelte';
	import NotifyRail from './shared/NotifyRail.svelte';

	const data = useNotifyData();

	const unreadGroup = $derived(data.filteredNotifies.filter((n) => !n.read));
	const readGroup = $derived(data.filteredNotifies.filter((n) => n.read));
</script>

<svelte:head>
	<title>通知 · HotSpot</title>
</svelte:head>

<SplitPageLayout contentClassName="pt-3 lg:pt-2 pb-6">
	{#snippet header()}
		<PageHeader
			title="通知"
			subtitle={!data.needsAuth && data.unread > 0 ? `${data.unread} 条未读消息` : '暂无新通知'}
			islandMeta={!data.needsAuth ? (data.unread > 0 ? `${data.unread} 条未读` : '全部已读') : ''}
			titleSize="xl"
			onBack={() => smartBack()}
		>
			{#snippet islandIcon()}
				<Bell class="size-3.5 text-accent-default" strokeWidth={1.75} />
			{/snippet}
			{#snippet trailing()}
				{#if !data.needsAuth && data.unread > 0}
					<Button
						type="button"
						variant="soft"
						size="sm"
						class="hit-slop text-xs"
						onclick={() => data.markAllRead()}
					>
						<CheckCheck size={14} class="text-text-secondary" />
						全部已读
					</Button>
				{/if}
			{/snippet}
			{#snippet islandTrailing()}
				{#if !data.needsAuth && data.unread > 0}
					<ActionIconButton
						icon={CheckCheck}
						label="全部已读"
						variant="ghost"
						tone="muted"
						size={32}
						onClick={() => data.markAllRead()}
					/>
				{/if}
			{/snippet}
			{#snippet islandExpanded()}
				<div class="flex flex-col gap-2.5">
					<div class="flex items-center justify-between text-xs pb-0.5">
						<span class="font-medium text-text-primary">通知分类</span>
						<span class="font-mono text-xxs text-accent-default">
							{!data.needsAuth ? (data.unread > 0 ? `${data.unread} 条未读` : '全部已读') : '请先登录'}
						</span>
					</div>
					{#if !data.needsAuth}
						<div class="w-full overflow-x-auto no-scrollbar py-0.5">
							<TabBar
								items={FILTER_TABS.map((t) => ({ id: t.id, label: t.label }))}
								value={data.filter}
								onChange={(id) => (data.filter = id)}
							/>
						</div>
						{#if data.unread > 0}
							<div class="flex items-center gap-2 pt-0.5">
								<Button
									type="button"
									variant="soft"
									size="sm"
									class="flex-1 hit-slop h-8"
									onclick={() => data.markAllRead()}
								>
									<CheckCheck class="size-3.5" />
									<span>全部标为已读</span>
								</Button>
							</div>
						{/if}
					{:else}
						<div class="flex items-center justify-between gap-3 rounded-md bg-surface-subtle/50 px-3 py-2">
							<div class="flex flex-col">
								<span class="text-xs font-medium text-text-primary">登录查看通知</span>
								<span class="text-xxs text-text-secondary">收到互动提醒与系统通知</span>
							</div>
							<Button variant="primary" size="sm" onclick={() => void goto(ROUTES.login)}>
								登录
							</Button>
						</div>
					{/if}
				</div>
			{/snippet}
			{#snippet below()}
				{#if !data.needsAuth}
					<div class="flex items-center justify-between gap-3">
						<div class="min-w-0 flex-1">
							<TabBar
								items={FILTER_TABS.map((t) => ({ id: t.id, label: t.label }))}
								value={data.filter}
								onChange={(id) => (data.filter = id)}
							/>
						</div>
						{#if data.unread > 0}
							<Button
								type="button"
								variant="soft"
								size="sm"
								class="hidden sm:inline-flex shrink-0 hit-slop h-7 text-xs"
								onclick={() => data.markAllRead()}
							>
								<CheckCheck size={13} class="text-text-secondary" />
								<span>全部已读</span>
							</Button>
						{/if}
					</div>
				{/if}
			{/snippet}
		</PageHeader>
	{/snippet}

	<div class="flex flex-col">
		<InvitesPendingCard
			invites={data.invites}
			invitePending={data.invitePending}
			onAccept={(id) => void data.handleAcceptInvite(id)}
			onReject={(id) => void data.handleRejectInvite(id)}
		/>

		{#if unreadGroup.length > 0}
			<SectionHead variant="eyebrow" title="新消息" icon={Bell} count={unreadGroup.length} />
			<ListContainer bordered class="mb-6">
				{#each unreadGroup as n (n.id)}
					<NotifyRow
						notif={n}
						actionPending={n.inviteId ? data.invitePending.has(n.inviteId) : false}
						onRead={data.markSingleRead}
						onRemove={data.remove}
						onAccept={(id) => void data.handleAcceptInvite(id)}
						onReject={(id) => void data.handleRejectInvite(id)}
					/>
				{/each}
			</ListContainer>
		{/if}

		{#if readGroup.length > 0}
			<SectionHead variant="eyebrow" title="已读" count={readGroup.length} />
			<ListContainer bordered class="">
				{#each readGroup as n (n.id)}
					<NotifyRow
						notif={n}
						actionPending={n.inviteId ? data.invitePending.has(n.inviteId) : false}
						onRead={data.markSingleRead}
						onRemove={data.remove}
						onAccept={(id) => void data.handleAcceptInvite(id)}
						onReject={(id) => void data.handleRejectInvite(id)}
					/>
				{/each}
			</ListContainer>
		{/if}

		{#if !data.loading && data.nextCursor !== null}
			<div class={readGroup.length > 0 ? 'mt-6' : ''}>
				<InfiniteSentinel
					hasMore={data.nextCursor !== null}
					loading={data.loadingMore}
					error={Boolean(data.loadError)}
					onLoadMore={() => void data.loadMoreNotifs()}
				/>
			</div>
		{:else if data.filteredNotifies.length > 0 && !data.loadError && data.nextCursor === null}
			<ListEnd />
		{/if}

		<div use:contentFade={data.loading}>
			{#if data.loading}
				<SkeletonList rows={6} leading={44} label="加载通知…" rowClass="px-4 py-3.5" />
			{:else if data.loadError}
				<LoadError message={data.loadError} onRetry={() => data.refresh()} />
			{:else if data.needsAuth}
				<div class="flex min-h-[50vh] items-center justify-center">
					<NeedsAuthNotice
						spotlight
						title="登录后查看通知"
						description="收到的点赞、评论与关注会出现在这里"
						highlights={[
							{ icon: Heart, text: '谁点赞了你的瞬间' },
							{ icon: UserPlus, text: '新的关注与互相关注' },
							{ icon: Users, text: '入群邀请与群申请结果' }
						]}
					/>
				</div>
			{:else if data.filteredNotifies.length === 0}
				<EmptyState
					spotlight
					iconTone="accent"
					title={data.filter === 'all' ? '暂无通知' : '该分类下暂无通知'}
					description={data.filter === 'all'
						? '收到的点赞、评论与关注会出现在这里。'
						: '换个分类看看，或去附近逛逛。'}
				>
					{#snippet icon()}
						<Bell class="size-7" strokeWidth={1.5} aria-hidden="true" />
					{/snippet}
					{#snippet extra()}
						<div class="mt-1 flex flex-wrap items-center justify-center gap-2">
							<Button
								type="button"
								variant="soft"
								size="sm"
								onclick={() => void goto(ROUTES.nearbyPeople)}
								class="hit-slop"
							>去附近看看</Button>
							<Button
								type="button"
								variant="soft"
								size="sm"
								onclick={() => void goto(ROUTES.nearbyMoments)}
								class="hit-slop"
							>逛逛热门瞬间</Button>
						</div>
					{/snippet}
				</EmptyState>
			{/if}
		</div>
	</div>

	{#snippet rail()}
		{#if !data.needsAuth}
			<NotifyRail
				notifies={data.notifies}
				unread={data.unread}
				meId={data.meId}
				followedBack={data.followedBack}
				onFollowBack={(n) => void data.followBack(n)}
			/>
		{/if}
	{/snippet}
</SplitPageLayout>
