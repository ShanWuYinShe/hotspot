<script lang="ts">
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';	import { goto } from '$app/navigation';
	import { Camera, ChevronRight, Heart, ImageIcon, MapPin, Pencil, Settings, UserCheck, UserPlus, Users } from '@lucide/svelte';
	import ContentColumn from '@/components/layout/page/ContentColumn.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
import NeedsAuthNotice from '@/components/auth/NeedsAuthNotice.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import { useAccountData } from '@/lib/profile-data.svelte';
	import { ROUTES, ME_SECTION_PATHS, type ProfileSection } from '@/lib/routes';

	// 移动个人主页：资料页眉 + 身份卡（68px 头像 + 编辑按钮 + stats 横条居中）
	// + 个人数据 5 行链接（无 Tab/画廊）。仅 mobile 使用；tablet 归 ProfilePage。

	// getter 对象禁止解构（快照不响应，UI 永不更新），$derived 桥接给模板
	const account = useAccountData();
	const avatar = $derived(account.avatar);
	const nickname = $derived(account.nickname);
	const stats = $derived(account.stats);
	const bio = $derived(account.bio);
	const handle = $derived(account.handle);
	const locationName = $derived(account.locationName);
	const loading = $derived(account.loading);
	const error = $derived(account.error);
	const needsAuth = $derived(account.needsAuth);
	const retry = account.retry;

	const DATA_ITEMS = $derived<{ section: ProfileSection; label: string; detail: string; icon: typeof ImageIcon }[]>([
		{ section: 'moments', label: '我的瞬间', detail: '查看已发布的记录', icon: ImageIcon },
		{ section: 'groups', label: '我的群组', detail: `${stats.groups_count} 个正在参与`, icon: Users },
		{ section: 'friends', label: '好友', detail: `${stats.friends_count} 位好友`, icon: Heart },
		{ section: 'following', label: '关注', detail: `${stats.following_count} 人`, icon: UserPlus },
		{ section: 'fans', label: '粉丝', detail: `${stats.followers_count} 人`, icon: UserCheck }
	]);

	const statsList = $derived<{ label: string; value: number }[]>([
		{ label: '瞬间', value: stats.moments_count },
		{ label: '关注者', value: stats.followers_count },
		{ label: '群组', value: stats.groups_count }
	]);
</script>

<StandardPageLayout contentClassName="pt-4 pb-6">
	{#snippet header()}
		<!-- 资料页眉（eyebrow + 设置入口；底色对齐标准 PageHeader 的 surface-page） -->
			<div class="border-b border-border-default bg-surface-page" >
				<ContentColumn class="flex items-center justify-between py-3">
					<p class="eyebrow font-medium text-accent-default">个人中心</p>
					{#if !needsAuth}
						<!-- 未登录时设置页无意义，隐藏入口 -->
						<ActionIconButton icon={Settings} label="设置" onClick={() => void goto(ROUTES.settings)} variant="ghost" size={32} />
					{/if}
				</ContentColumn>
			</div>
	{/snippet}

	{#if loading}
		<InlineLoading label="加载个人资料…" />
	{:else if needsAuth}
		<NeedsAuthNotice title="登录后查看" description="登录后即可管理资料、查看互动数据与收藏内容" />
{:else if error}
		<LoadError message={error} onRetry={retry} />
	{:else}
		<!-- 身份卡（compact）：头像复用 Avatar，统计横条居中（对齐桌面 MetricRow 原语） -->
		<section class="rounded-lg surface-flat" aria-label="个人资料">
			<div class="p-5">
				<div class="flex items-start justify-between gap-5">
					<div class="flex min-w-0 items-center gap-4">
						<div class="relative shrink-0">
							<Avatar src={avatar} name={nickname} size={68} ring />
							<button
								type="button"
								aria-label="更换头像"
								onclick={() => void goto(ROUTES.meEdit)}
								class="hover-btn absolute -bottom-1 -right-1 flex items-center justify-center rounded-full focus-ring bg-accent-default text-text-on-accent size-6"
								 style="border:var(--cst-border-width-strong) solid var(--cst-surface-primary);"
							>
								<Camera class="size-3.5"  strokeWidth={2} />
							</button>
						</div>
						<div class="min-w-0">
							<h1 class="text-xl font-medium tracking-title text-text-primary">{nickname || '未设置昵称'}</h1>
							<p class="mt-0.5 text-xs text-text-secondary">{handle}</p>
							<p class="mt-2 flex items-center gap-1.5 text-xs text-text-secondary">
								<MapPin class="size-3.5 text-accent-default"   strokeWidth={1.75} />
								{locationName || '未设置位置'}
							</p>
						</div>
					</div>
					<button
						type="button"
						onclick={() => void goto(ROUTES.meEdit)}
						class="hover-btn inline-flex shrink-0 items-center gap-1.5 border border-border-default rounded-md bg-surface-primary px-3 py-2 text-xs font-medium text-text-primary focus-ring transition-colors hover:bg-accent-soft"
					>
						<Pencil class="size-3.5"  strokeWidth={1.75} />
						编辑资料
					</button>
				</div>
				<p class="mt-5 max-w-107.5 text-sm leading-6 text-text-secondary">
					{bio || '这个人还没有填写简介。'}
				</p>
			</div>
			<div class="grid grid-cols-3 border-t border-border-default">
				{#each statsList as stat}
					<div class="border-l border-border-default px-4 py-3 text-center first:border-l-0">
						<p class="text-sm font-medium tabular-nums text-text-primary">{stat.value}</p>
						<p class="mt-0.5 text-xs text-text-secondary">{stat.label}</p>
					</div>
				{/each}
			</div>
		</section>

		<!-- 个人数据 5 行链接 -->
		<ListContainer bordered class="mt-5" ariaLabel="个人数据">
			<div class="flex items-center justify-between bg-surface-primary px-5 py-3.5">
				<div>
					<p class="text-sm font-medium text-text-primary">个人数据</p>
					<p class="mt-0.5 text-xs text-text-secondary">在这里查看你的内容与关系</p>
				</div>
			</div>
			{#each DATA_ITEMS as item (item.section)}
				<ListItem
					onClick={() => void goto(ME_SECTION_PATHS[item.section])}
					ariaLabel={item.label}
					class="flex items-center gap-3 px-5 py-3.5"
				>
					<span class="flex shrink-0 items-center justify-center rounded-full icon-badge size-8">
						<item.icon class="size-4 text-accent-default"  strokeWidth={1.75} />
					</span>
					<span class="min-w-0 flex-1">
						<span class="block text-sm text-text-primary">{item.label}</span>
						<span class="mt-0.5 block truncate text-xs text-text-secondary">{item.detail}</span>
					</span>
					<ChevronRight class="size-4 shrink-0 text-text-secondary"   strokeWidth={1.75} />
				</ListItem>
			{/each}
		</ListContainer>
	{/if}
</StandardPageLayout>
