<script lang="ts">
	import { goto } from '$app/navigation';
	import {
		ChevronRight,
		Heart,
		ImageIcon,
		LayoutGrid,
		Pencil,
		Settings,
		Sparkles,
		UserCheck,
		UserPlus,
		Users,
		UsersRound
	} from '@lucide/svelte';
	import ContentColumn from '@/components/layout/page/ContentColumn.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import NeedsAuthNotice from '@/components/auth/NeedsAuthNotice.svelte';
	import SectionHead from '@/components/home/SectionHead.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import Button from '@/components/ui/button.svelte';
	import IdentityHero from '@/components/profile/IdentityHero.svelte';
	import ProfileCompleteness, { type CompletenessItem } from '@/components/profile/ProfileCompleteness.svelte';
	import { useAccountData } from '@/lib/profile-data.svelte';
	import { ROUTES, ME_SECTION_PATHS } from '@/lib/routes';
	import { relationshipsHref } from '@/lib/relationships';

	// 移动个人主页（R13 与桌面 ProfilePage 同构）：
	// 身份封面卡（IdentityHero，h1=昵称，动作整行在卡片拇指区）
	// + 个人数据 5 行链接（内容与关系资产）+ 资料状态卡。
	// 仅 mobile 使用；tablet/desktop 归 ProfilePage。页面信息架构（5 行关系入口）保持不变。

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

	type DataItem = { id: string; href: string; title: string; description: string; count: string; icon: typeof ImageIcon };

	// 副标题统一为「量词 + 数字」（2026-10-19 UI 审查），计数右对齐展示
	const DATA_ITEMS = $derived<DataItem[]>([
		{
			id: 'moments',
			href: ME_SECTION_PATHS.moments,
			title: '我的瞬间',
			description: '记录城市里的光影',
			count: `${stats.moments_count} 条`,
			icon: ImageIcon
		},
		{
			id: 'groups',
			href: ME_SECTION_PATHS.groups,
			title: '我的群组',
			description: '和同好一起探索',
			count: `${stats.groups_count} 个`,
			icon: Users
		},
		// 好友/关注/粉丝直接深链到关系聚合页的对应 tab——不再各自维护一条独立路由
		{
			id: 'friends',
			href: relationshipsHref('friends'),
			title: '好友',
			description: '可以直接聊天的人',
			count: `${stats.friends_count} 位`,
			icon: Heart
		},
		{
			id: 'following',
			href: relationshipsHref('following'),
			title: '关注',
			description: '你订阅的创作者',
			count: `${stats.following_count} 人`,
			icon: UserPlus
		},
		{
			id: 'fans',
			href: relationshipsHref('fans'),
			title: '粉丝',
			description: '关注你的人',
			count: `${stats.followers_count} 人`,
			icon: UserCheck
		}
	]);

	const statsList = $derived<{ label: string; value: number; onClick: () => void }[]>([
		{ label: '瞬间', value: stats.moments_count, onClick: () => void goto(ME_SECTION_PATHS.moments) },
		{ label: '粉丝', value: stats.followers_count, onClick: () => void goto(relationshipsHref('fans')) },
		{ label: '群组', value: stats.groups_count, onClick: () => void goto(ME_SECTION_PATHS.groups) }
	]);

	const COMPLETENESS = $derived<CompletenessItem[]>([
		{ id: 'avatar', label: '上传一张头像', done: !!avatar, href: ROUTES.meEdit },
		{ id: 'nickname', label: '设置昵称', done: !!nickname, href: ROUTES.meEdit },
		{ id: 'bio', label: '写一句个人简介', done: !!bio, href: ROUTES.meEdit },
		{ id: 'location', label: '选择所在位置', done: !!locationName, href: ROUTES.meEdit },
		{ id: 'group', label: '加入一个群组', done: stats.groups_count > 0, href: ROUTES.nearbyGroups }
	]);
</script>

<StandardPageLayout contentClassName="animate-fade-in pt-4 pb-6">
	{#snippet header()}
		<!-- 页眉只留页级 eyebrow：入口（编辑资料/设置）统一收在身份卡拇指区，不重复；
		     不做常驻底色，避免在连续画布上形成一条「白带」（Round 9 页头契约）。 -->
		<ContentColumn class="flex items-center justify-between py-2.5">
			<p class="eyebrow">个人中心</p>
		</ContentColumn>
	{/snippet}

	{#if loading}
		<InlineLoading label="加载个人资料…" />
	{:else if needsAuth}
		<NeedsAuthNotice
			spotlight
			title="登录后查看"
			description="登录后即可管理资料、查看互动数据与收藏内容"
			highlights={[
				{ icon: Pencil, text: '完善头像、昵称与简介，让附近的人认识你' },
				{ icon: Sparkles, text: '整理你的瞬间与收藏，随时回看' },
				{ icon: UsersRound, text: '管理好友、关注与群组关系' }
			]}
		/>
	{:else if error}
		<LoadError message={error} onRetry={retry} />
	{:else}
		<!-- 身份封面卡（h1=昵称；移动端页面无 PageHeader，标题层级由身份卡承担） -->
		<div class="mb-6">
			<IdentityHero
				mode="self"
				titleTag="h1"
				name={nickname || '未设置昵称'}
				{bio}
				{avatar}
				{handle}
				location={locationName || '未设置位置'}
				metrics={statsList}
			>
				{#snippet actions()}
					<Button
						variant="primary"
						size="md"
						class="hit-slop-44 flex-1 gap-1.5"
						onclick={() => void goto(ROUTES.meEdit)}
					>
						<Pencil class="size-4" strokeWidth={1.75} />
						编辑资料
					</Button>
					<Button
						variant="soft"
						size="md"
						class="hit-slop-44 flex-1 gap-1.5"
						aria-label="系统设置"
						onclick={() => void goto(ROUTES.settings)}
					>
						<Settings class="size-4 text-text-secondary" strokeWidth={1.75} />
						设置
					</Button>
				{/snippet}
			</IdentityHero>
		</div>

		<!-- 个人数据 5 行链接：唯一分组软岛 ListContainer bordered -->
		<section aria-label="个人数据">
			<SectionHead title="个人数据" subtitle="内容与关系资产" icon={LayoutGrid} />
			<ListContainer bordered>
				{#each DATA_ITEMS as item (item.id)}
					<ListItem
						onClick={() => void goto(item.href)}
						ariaLabel={item.title}
						class="flex items-center gap-3.5 px-3 py-3"
					>
						<span class="flex size-9.5 shrink-0 items-center justify-center rounded-sm bg-accent-soft text-text-accent-on-soft">
							<item.icon class="size-4.5" strokeWidth={1.75} />
						</span>
						<span class="min-w-0 flex-1">
							<span class="block text-md font-medium text-text-primary">{item.title}</span>
							<span class="mt-0.5 block truncate text-xs text-text-secondary">{item.description}</span>
						</span>
						<span class="shrink-0 text-xs font-medium tabular-nums text-text-secondary">{item.count}</span>
						<ChevronRight class="size-4 shrink-0 text-text-secondary" strokeWidth={1.5} />
					</ListItem>
				{/each}
			</ListContainer>
		</section>

		<div class="mt-6">
			<ProfileCompleteness items={COMPLETENESS} />
		</div>
	{/if}
</StandardPageLayout>