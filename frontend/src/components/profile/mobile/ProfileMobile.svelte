<script lang="ts">
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';	import { goto } from '$app/navigation';
	import { ChevronRight, Heart, ImageIcon, MapPin, Pencil, Settings, UserCheck, UserPlus, Users } from '@lucide/svelte';
	import ContentColumn from '@/components/layout/page/ContentColumn.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import MetricRow from '@/components/data-display/MetricRow.svelte';
import NeedsAuthNotice from '@/components/auth/NeedsAuthNotice.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import { useAccountData } from '@/lib/profile-data.svelte';
	import { ROUTES, ME_SECTION_PATHS } from '@/lib/routes';
	import { relationshipsHref } from '@/lib/relationships';

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

	const DATA_ITEMS = $derived<{ id: string; href: string; label: string; detail: string; icon: typeof ImageIcon }[]>([
		// 副标题统一为「量词 + 数字」（2026-10-19 UI 审查）：原先第二行是动词短语（"查看已发布的
		// 记录"）、好友行又重复标题（"好友 / 6 位好友"），五行读起来不成体系。
		{ id: 'moments', href: ME_SECTION_PATHS.moments, label: '我的瞬间', detail: `${stats.moments_count} 条`, icon: ImageIcon },
		{ id: 'groups', href: ME_SECTION_PATHS.groups, label: '我的群组', detail: `${stats.groups_count} 个`, icon: Users },
		// 好友/关注/粉丝直接深链到关系聚合页的对应 tab——不再各自维护一条独立路由
		{ id: 'friends', href: relationshipsHref('friends'), label: '好友', detail: `${stats.friends_count} 位`, icon: Heart },
		{ id: 'following', href: relationshipsHref('following'), label: '关注', detail: `${stats.following_count} 人`, icon: UserPlus },
		{ id: 'fans', href: relationshipsHref('fans'), label: '粉丝', detail: `${stats.followers_count} 人`, icon: UserCheck }
	]);

	const statsList = $derived<{ label: string; value: number; onClick: () => void }[]>([
		{ label: '瞬间', value: stats.moments_count, onClick: () => void goto(ME_SECTION_PATHS.moments) },
		{ label: '关注者', value: stats.followers_count, onClick: () => void goto(relationshipsHref('fans')) },
		{ label: '群组', value: stats.groups_count, onClick: () => void goto(ME_SECTION_PATHS.groups) }
	]);
</script>

<StandardPageLayout contentClassName="pt-4 pb-6">
	{#snippet header()}
		<!-- 资料页眉：毛玻璃微透，与全局天际线微背景自然互融 -->
		<div class="border-b border-border-subtle/40 bg-surface-primary/75 backdrop-blur-md">
			<ContentColumn class="flex items-center justify-between py-3">
				<p class="eyebrow">个人中心</p>
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
		<!-- 身份卡（compact）：头像复用 Avatar，统计胶囊居中（对齐桌面 MetricRow capsule 原语）。
		     卡片壳与桌面 ProfilePage 统一为 Bento 微岛生态。 -->
		<section class="mb-6 rounded-2xl border border-border-subtle/50 bg-surface-primary/85 backdrop-blur-xs shadow-xs overflow-hidden" aria-label="个人资料">
			<!-- 顶部身份信息区 -->
			<div class="relative p-5">
				<div class="flex items-start justify-between gap-5">
					<div class="flex min-w-0 items-center gap-4">
						<Avatar src={avatar} name={nickname} size="xl" ring class="shrink-0 ring-2 ring-accent-default/20 shadow-xs" />
						<div class="min-w-0">
							<h1 class="text-xl font-bold tracking-title text-text-primary">{nickname || '未设置昵称'}</h1>
							<p class="mt-0.5 text-xs text-text-secondary">{handle}</p>
							<p class="mt-2 flex items-center gap-1.5 text-xs text-text-secondary">
								<MapPin class="size-3.5 text-accent-default" strokeWidth={1.75} />
								{locationName || '未设置位置'}
							</p>
						</div>
					</div>
				</div>
				{#if bio}
					<p class="mt-3.5 max-w-107.5 text-xs leading-relaxed text-text-secondary">
						{bio}
					</p>
				{/if}

				<!-- 移动端单手黄金操作区：大拇指无需移位，直达高频设置与编辑 -->
				<div class="mt-4 flex items-center gap-2.5 pt-1">
					<button
						type="button"
						onclick={() => void goto(ROUTES.meEdit)}
						class="hover-btn hit-slop inline-flex flex-1 items-center justify-center gap-1.5 border border-border-subtle/80 rounded-full bg-surface-primary/90 py-2 px-3 text-xs font-medium text-text-primary shadow-xs focus-ring transition-all hover:bg-accent-soft hover:border-accent-border/60 active:scale-[0.98]"
					>
						<Pencil class="size-3.5 text-text-secondary" strokeWidth={1.75} />
						编辑资料
					</button>
					<button
						type="button"
						onclick={() => void goto(ROUTES.settings)}
						class="hover-btn hit-slop inline-flex items-center justify-center gap-1.5 border border-border-subtle/80 rounded-full bg-surface-primary/90 py-2 px-4 text-xs font-medium text-text-primary shadow-xs focus-ring transition-all hover:bg-accent-soft hover:border-accent-border/60 active:scale-[0.98]"
						aria-label="系统设置"
					>
						<Settings class="size-3.5 text-text-secondary" strokeWidth={1.75} />
						<span>设置</span>
					</button>
				</div>
			</div>
			<!-- 统计胶囊：复用 MetricRow capsule 原语 -->
			<MetricRow variant="capsule" class="px-5 pb-5 pt-0" items={statsList} />
		</section>

		<!-- 个人数据 5 行链接：使用唯一分组微岛卡 ListContainer bordered。 -->
		<div>
			<div class="px-1 mb-2.5 flex items-center justify-between">
				<p class="eyebrow">个人数据</p>
				<span class="text-xxs text-text-secondary">内容与关系资产</span>
			</div>
			<ListContainer bordered ariaLabel="个人数据" class="shadow-xs rounded-2xl border-border-subtle/50 bg-surface-primary/85 backdrop-blur-xs overflow-hidden">
			{#each DATA_ITEMS as item (item.id)}
				<ListItem
					onClick={() => void goto(item.href)}
					ariaLabel={item.label}
					class="flex items-center gap-3.5 px-4 py-3.5"
				>
					<span class="flex shrink-0 items-center justify-center rounded-xl bg-accent-soft text-accent-default size-8.5 transition-transform group-hover:scale-105">
						<item.icon class="size-4" strokeWidth={1.75} />
					</span>
					<span class="min-w-0 flex-1">
						<span class="block text-sm font-medium text-text-primary">{item.label}</span>
						<span class="mt-0.5 block truncate text-xs text-text-secondary">{item.detail}</span>
					</span>
					<ChevronRight class="size-4 shrink-0 text-text-secondary" strokeWidth={1.75} />
				</ListItem>
				{/each}
			</ListContainer>
		</div>
	{/if}
</StandardPageLayout>
