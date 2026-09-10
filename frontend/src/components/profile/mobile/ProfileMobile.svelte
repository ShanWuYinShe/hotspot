<script lang="ts">
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';	import { goto } from '$app/navigation';
	import { ChevronRight, Heart, ImageIcon, MapPin, Pencil, Settings, UserCheck, UserPlus, Users } from '@lucide/svelte';
	import ContentColumn from '@/components/layout/page/ContentColumn.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
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
		{ id: 'moments', href: ME_SECTION_PATHS.moments, label: '我的瞬间', detail: '查看已发布的记录', icon: ImageIcon },
		{ id: 'groups', href: ME_SECTION_PATHS.groups, label: '我的群组', detail: `${stats.groups_count} 个正在参与`, icon: Users },
		// 好友/关注/粉丝直接深链到关系聚合页的对应 tab——不再各自维护一条独立路由
		{ id: 'friends', href: relationshipsHref('friends'), label: '好友', detail: `${stats.friends_count} 位好友`, icon: Heart },
		{ id: 'following', href: relationshipsHref('following'), label: '关注', detail: `${stats.following_count} 人`, icon: UserPlus },
		{ id: 'fans', href: relationshipsHref('fans'), label: '粉丝', detail: `${stats.followers_count} 人`, icon: UserCheck }
	]);

	const statsList = $derived<{ label: string; value: number }[]>([
		{ label: '瞬间', value: stats.moments_count },
		{ label: '关注者', value: stats.followers_count },
		{ label: '群组', value: stats.groups_count }
	]);
</script>

<StandardPageLayout contentClassName="pt-4 pb-6">
	{#snippet header()}
		<!-- 资料页眉（eyebrow；底色对齐标准 PageHeader 的 surface-page）。
			     不在此放设置/外观入口——它们统一收在下方身份卡的拇指操作区，
			     同一页只保留一处设置入口；外观走设置页「外观与显示」，
			     避免与「设置」按钮形成同一功能两条路径。 -->
			<div class="border-b border-border-default bg-surface-page" >
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
		<!-- 身份卡（compact）：头像复用 Avatar，统计横条居中（对齐桌面 MetricRow 原语）。
		     卡片壳与桌面 ProfilePage 统一为「白卡 + 发丝描边 + 主题投影」语言
		     （原 card-raised 纯投影，两端不一致）。 -->
		<section class="rounded-lg border border-border-default bg-surface-primary shadow-raised-sm" aria-label="个人资料">
			<div class="p-5">
				<div class="flex items-start justify-between gap-5">
					<div class="flex min-w-0 items-center gap-4">
						<!-- 头像不再挂「更换头像」相机按钮：它与下方「编辑资料」跳同一个页面，
						     属同屏重复入口；且它只做跳转、并不直接改头像，affordance 有误导。
						     头像更换统一走「编辑资料」。 -->
						<Avatar src={avatar} name={nickname} size={68} ring class="shrink-0" />
						<div class="min-w-0">
							<h1 class="text-xl font-semibold tracking-title text-text-primary">{nickname || '未设置昵称'}</h1>
							<p class="mt-0.5 text-xs text-text-secondary">{handle}</p>
							<p class="mt-2 flex items-center gap-1.5 text-xs text-text-secondary">
								<MapPin class="size-3.5 text-accent-default"   strokeWidth={1.75} />
								{locationName || '未设置位置'}
							</p>
						</div>
					</div>
				</div>
				{#if bio}
					<p class="mt-3 max-w-107.5 text-xs leading-relaxed text-text-secondary">
						{bio}
					</p>
				{/if}

				<!-- 移动端单手黄金操作区：大拇指无需移位，直达高频设置与编辑 -->
				<div class="mt-4 flex items-center gap-2 pt-1">
					<button
						type="button"
						onclick={() => void goto(ROUTES.meEdit)}
						class="hover-btn inline-flex flex-1 items-center justify-center gap-1.5 border border-border-default rounded-md bg-surface-primary py-2 px-3 text-xs font-medium text-text-primary shadow-raised-sm focus-ring transition-colors hover:bg-accent-soft"
					>
						<Pencil class="size-3.5 text-text-secondary" strokeWidth={1.75} />
						编辑资料
					</button>
					<button
						type="button"
						onclick={() => void goto(ROUTES.settings)}
						class="hover-btn inline-flex items-center justify-center gap-1.5 border border-border-default rounded-md bg-surface-primary py-2 px-3 text-xs font-medium text-text-primary shadow-raised-sm focus-ring transition-colors hover:bg-accent-soft"
						aria-label="系统设置"
					>
						<Settings class="size-3.5 text-text-secondary" strokeWidth={1.75} />
						<span>设置</span>
					</button>
				</div>
			</div>
			<!-- 统计横条：仅保留顶部分隔线，列间用留白（竖线在小屏三列里显得切割局促）。
			     数字 display 化（大号粗体 + 紧字距），对齐 Figma stats 模式 -->
			<div class="grid grid-cols-3 border-t border-border-default">
				{#each statsList as stat}
					<div class="px-4 py-3.5 text-center">
						<p class="text-2xl font-bold tabular-nums tracking-tight text-text-primary">{stat.value}</p>
						<p class="mt-0.5 text-xs text-text-secondary">{stat.label}</p>
					</div>
				{/each}
			</div>
		</section>

		<!-- 个人数据 5 行链接（唯一分组卡 ListContainer bordered：内缩圆角行） -->
		<div class="mt-5">
			<p class="eyebrow px-1 mb-2">个人数据</p>
			<p class="px-1 mb-2 text-xs text-text-secondary">在这里查看你的内容与关系</p>
			<ListContainer bordered ariaLabel="个人数据">
			{#each DATA_ITEMS as item (item.id)}
				<ListItem
					onClick={() => void goto(item.href)}
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
		</div>
	{/if}
</StandardPageLayout>
