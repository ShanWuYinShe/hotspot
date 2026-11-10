<script lang="ts">
	import { goto } from '$app/navigation';
	import {
		MessageCircle,
		MoreHorizontal,
		Share2,
		UserCheck,
		UserPlus,
		UserX
	} from '@lucide/svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import SplitPageLayout from '@/components/layout/page/SplitPageLayout.svelte';
	import DetailHeader from '@/components/layout/page/DetailHeader.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import IdentityHero from '@/components/profile/IdentityHero.svelte';
	import Button from '@/components/ui/button.svelte';
	import { ROUTES } from '@/lib/routes';
	import { smartBack } from '@/lib/smart-back';
	import { platform } from '@/stores/platform.svelte';

	import { useUserProfileData } from './shared/useUserProfileData.svelte';
	import UserGalleryGrid from './shared/UserGalleryGrid.svelte';
	import UserRail from './shared/UserRail.svelte';
	import UserOverlays from './shared/UserOverlays.svelte';

	let { params }: { params: { userId: string } } = $props();

	const page = useUserProfileData(() => params.userId);
	const user = $derived(page.user);
	const isMobile = $derived(platform.platform === 'mobile');
</script>

<svelte:head>
	<title>用户主页 · HotSpot</title>
</svelte:head>

{#snippet userHeaderActions()}
	<div class="flex items-center gap-3">
		<ActionIconButton
			icon={Share2}
			label={page.shareCopied ? '链接已复制' : '分享资料'}
			variant="ghost"
			size={36}
			onClick={page.copyProfileLink}
		/>
		{#if !page.isSelf}
			<ActionIconButton
				icon={MoreHorizontal}
				label="更多操作"
				variant="ghost"
				size={36}
				onClick={() => {
					page.remarkDraft = page.remark;
					page.moreOpen = true;
				}}
			/>
		{/if}
	</div>
{/snippet}

<SplitPageLayout
	contentMaxWidth="var(--cst-content-detail)"
	contentClassName="pt-1 pb-6"
>
	{#snippet header()}
		<DetailHeader
			category="用户"
			entityName={user.name ? `@${user.name}` : '用户资料'}
			author={user.name && !page.loading ? {
				id: params.userId,
				name: user.name,
				avatar: user.avatar,
				location: user.location,
				time: page.handleLine,
				role: '用户'
			} : undefined}
			onBack={() => smartBack()}
			trailing={userHeaderActions}
		/>
	{/snippet}

	{#if page.loading}
		<InlineLoading label="加载资料…" />
	{:else if page.loadError}
		<LoadError message={page.loadError} onRetry={() => page.refresh()} />
	{:else}
		<IdentityHero
			mode="other"
			titleTag="h1"
			name={user.name}
			handle={page.handleLine}
			bio={user.bio || '这个人还没有填写简介。'}
			avatar={user.avatar}
			location={user.location || '未设置位置'}
			tags={user.tags}
			online={page.presence === null ? undefined : page.presence === 'online'}
			metrics={page.metrics}
		>
			{#snippet actions()}
				{#if !page.isSelf}
					{#if page.followed}
						<Button
							type="button"
							variant="primary"
							size="md"
							class="hit-slop-44 flex-1 gap-1.5 sm:flex-none"
							disabled={page.blocked}
							onclick={() => !page.blocked && void goto(ROUTES.message(params.userId))}
						>
							<MessageCircle class="size-4" strokeWidth={1.75} />
							私信
						</Button>
						<Button
							type="button"
							variant="soft"
							size="md"
							class="hit-slop-44 flex-1 gap-1.5 sm:flex-none"
							aria-pressed={page.followed}
							disabled={page.followPending || page.blocked}
							onclick={() => void page.toggleFollow()}
						>
							<UserCheck class="size-4" strokeWidth={1.75} />
							{page.followed ? '已关注' : '关注'}
						</Button>
					{:else}
						<Button
							type="button"
							variant="primary"
							size="md"
							class="hit-slop-44 flex-1 gap-1.5 sm:flex-none"
							aria-pressed={page.followed}
							disabled={page.followPending || page.blocked}
							onclick={() => void page.toggleFollow()}
						>
							<UserPlus class="size-4" strokeWidth={1.75} />
							关注
						</Button>
						<Button
							type="button"
							variant="soft"
							size="md"
							class="hit-slop-44 flex-1 gap-1.5 sm:flex-none"
							disabled={page.blocked}
							onclick={() => !page.blocked && void goto(ROUTES.message(params.userId))}
						>
							<MessageCircle class="size-4" strokeWidth={1.75} />
							私信
						</Button>
					{/if}
				{/if}
			{/snippet}

			{#snippet children()}
				{#if page.blocked}
					<p
						class="flex items-center gap-2 rounded-md bg-danger-bg/60 px-3 py-2 text-sm text-text-status-danger-on-soft"
					>
						<UserX size={14} />
						你已屏蔽此用户；关注和私信已停用。
					</p>
				{/if}
			{/snippet}
		</IdentityHero>

		<UserGalleryGrid
			userName={user.name}
			galleryMoments={user.galleryMoments}
			galleryBspLayouts={page.galleryBspLayouts}
		/>
	{/if}

	{#snippet rail()}
		<UserRail
			location={user.location}
			tags={user.tags}
			mutualFriends={page.mutualFriends}
		/>
	{/snippet}
</SplitPageLayout>

<UserOverlays
	userName={user.name}
	userId={params.userId}
	bind:moreOpen={page.moreOpen}
	bind:remarkDraft={page.remarkDraft}
	followed={page.followed}
	savingRemark={page.savingRemark}
	blocked={page.blocked}
	bind:confirmBlockOpen={page.confirmBlockOpen}
	bind:confirmUnblockOpen={page.confirmUnblockOpen}
	bind:reportOpen={page.reportOpen}
	{isMobile}
	onSaveRemark={() => void page.saveRemark()}
	onToggleBlock={() => void page.toggleBlock()}
/>
