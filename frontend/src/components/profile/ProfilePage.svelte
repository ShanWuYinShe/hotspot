<script lang="ts">
	import { goto } from '$app/navigation';
	import { ChevronRight, LayoutGrid, Pencil, Settings, Sparkles, UserRound, Users, UsersRound } from '@lucide/svelte';
	import { platform } from '@/stores/platform.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
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
	import SectionMoments from '@/components/profile/SectionMoments.svelte';
	import SectionGroups from '@/components/profile/SectionGroups.svelte';
	import { useAccountData } from '@/lib/profile-data.svelte';
	import { smartBack } from '@/lib/smart-back';
	import { ROUTES, ME_SECTION_PATHS, type ProfileSection } from '@/lib/routes';

	// 个人主页（/me）——全设备使用：
	// 总览 = 身份封面卡（IdentityHero，桌面/移动同构）+ 我的内容入口 + 资料状态，
	// 定位为「内容与关系导航落地页」，与编辑页去重（bio/位置只出现一次）；
	// 分区模式 = PageHeader + Section 组件（moments/groups）。
	// 设备分派：mobile 总览走 ProfileMobile；tablet/desktop 总览与全部 me 分区页
	// （任意设备）走本组件。relationships 聚合分区由 /me/relationships 独立渲染
	// （RelationshipsPage），不走本组件。

	let { section }: { section: string | null } = $props();

	// useAccountData 返回 getter 对象：解构会拿到非响应式快照（UI 永不更新），
	// 必须保留对象本体，用 $derived 逐字段桥接给模板。
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

	const SECTION_META: Record<string, { title: string }> = {
		moments: { title: '我的瞬间' },
		groups: { title: '我的群组' }
	};
	const meta = $derived(section && SECTION_META[section] ? SECTION_META[section] : null);
	const device = $derived(platform.platform);

	// 指标只做统计展示：页面下方「我的内容」列表才是导航入口，避免同一目标同屏两条入口。
	const metricItems = $derived([
		{ label: '瞬间', value: stats.moments_count },
		{ label: '粉丝', value: stats.followers_count },
		{ label: '群组', value: stats.groups_count }
	]);

	// ── 桌面端灵动岛（PageHeader islandMeta / islandTrailing）────────────────
	// 总览：岛上第二行 =「关注 X · 粉丝 Y」（复用 account.stats 的真实数据，非 mock）；
	// 分区模式：显示该分区计数（moments/groups 同样来自 account.stats）；
	// 加载中 / 未登录 / 错误态不显示 meta，避免岛上出现误导性的 0。
	const overviewIslandMeta = $derived(
		loading || needsAuth || error
			? undefined
			: `关注 ${stats.following_count} · 粉丝 ${stats.followers_count}`
	);
	const sectionIslandMeta = $derived(
		loading || needsAuth || error
			? undefined
			: section === 'moments'
				? `共 ${stats.moments_count} 条`
				: section === 'groups'
					? `共 ${stats.groups_count} 个`
					: undefined
	);

	type EntryItem = {
		id: string;
		href: string;
		title: string;
		description: string;
		count?: string;
		icon: typeof Users;
	};

	// 我的内容入口：图标砖 + 标题/说明 + 右对齐计数 + chevron（与移动端 ProfileMobile 同构）
	const ENTRY_ITEMS = $derived<EntryItem[]>([
		{
			id: 'moments',
			href: ME_SECTION_PATHS.moments,
			title: '我的瞬间',
			description: '记录城市里的光影',
			count: `${stats.moments_count} 条`,
			icon: Sparkles
		},
		{
			id: 'groups',
			href: ME_SECTION_PATHS.groups,
			title: '我的群组',
			description: '和同好一起探索',
			count: `${stats.groups_count} 个`,
			icon: UsersRound
		},
		{
			id: 'relationships',
			href: ME_SECTION_PATHS.relationships,
			title: '关系',
			description: '关注与粉丝都在这里',
			count: `${stats.friends_count} 位好友`,
			icon: Users
		}
	]);

	// 资料完善度：与资料接口实际返回的字段对齐（头像/昵称/简介/位置/群组）。
	const COMPLETENESS = $derived<CompletenessItem[]>([
		{ id: 'avatar', label: '上传一张头像', done: !!avatar, href: ROUTES.meEdit },
		{ id: 'nickname', label: '设置昵称', done: !!nickname, href: ROUTES.meEdit },
		{ id: 'bio', label: '写一句个人简介', done: !!bio, href: ROUTES.meEdit },
		{ id: 'location', label: '选择所在位置', done: !!locationName, href: ROUTES.meEdit },
		{ id: 'group', label: '加入一个群组', done: stats.groups_count > 0, href: ROUTES.nearbyGroups }
	]);
</script>

{#if meta}
	<!-- 分区模式：StandardPageLayout + integrated PageHeader；
	     Section 组件为纯内容（不自建滚动/gutter），gutter 由 ContentColumn 统一提供 -->
	<StandardPageLayout contentMaxWidth="var(--cst-content-breath)" contentClassName="pt-4 pb-6">
		{#snippet header()}
			<PageHeader
				variant="integrated"
				title={meta.title}
				islandMeta={sectionIslandMeta}
				maxWidth="var(--cst-content-breath)"
				titleSize={device === 'desktop' ? 'xl' : '2xl'}
				onBack={() => smartBack(ROUTES.me)}
			>
				{#snippet islandIcon()}
					<UserRound class="size-3.5 text-accent-default" strokeWidth={1.75} />
				{/snippet}
			</PageHeader>
		{/snippet}
		{#if section === 'moments'}
			<SectionMoments />
		{:else if section === 'groups'}
			<SectionGroups />
		{/if}
	</StandardPageLayout>
{:else}
	<!-- 总览模式：身份封面卡 + 我的内容 + 资料状态。
	     总览页是长内容呼吸型布局，顶/底 padding 略大于标准列表页（pt-6 pb-8），
	     属于样板 pt-4 pb-6 的可登记变体。 -->
	<StandardPageLayout contentMaxWidth="var(--cst-content-breath)" contentClassName="pt-6 pb-8">
		{#snippet header()}
			<!-- 页头不放操作按钮：编辑资料/设置统一收在身份卡的动作区，
			     同一页同一功能只允许一个入口。外观控制不进身份卡：桌面顶栏已有常驻
			     「外观设置」，其余设备走设置页「外观与显示」。
			     R17：页头与正文共用 1280 宽屏呼吸档（此前正文在 1440/1920 居中收窄而页头
			     全宽，标题左缘比身份卡左移 46/158px，违反「同一内容轴」）。 -->
			<PageHeader
				variant="integrated"
				title="个人中心"
				subtitle="你的公开资料、内容与关系的入口"
				islandMeta={overviewIslandMeta}
				maxWidth="var(--cst-content-breath)"
				titleSize={device === 'desktop' ? 'xl' : '2xl'}
			>
				{#snippet islandIcon()}
					<UserRound class="size-3.5 text-accent-default" strokeWidth={1.75} />
				{/snippet}
				{#snippet islandTrailing()}
					<!-- 滚动入岛后「编辑资料」常驻；数据未就绪/未登录/加载失败时不给不可用入口 -->
					{#if !loading && !needsAuth && !error}
						<Button
							variant="primary"
							size="sm"
							shape="pill"
							class="hit-slop gap-1 px-2.5"
							aria-label="编辑资料"
							onclick={() => void goto(ROUTES.meEdit)}
						>
							<Pencil class="size-3.5" strokeWidth={1.75} />
							编辑资料
						</Button>
					{/if}
				{/snippet}
			</PageHeader>
		{/snippet}

		{#if loading}
			<InlineLoading label="加载个人资料…" />
		{:else if needsAuth}
			<div class="w-full">
				<!-- 访客身份封面卡：展示个人中心全貌，提供引导与从容空间感 -->
				<div class="mb-6">
					<IdentityHero
						mode="self"
						name="开启你的城市漫游档案"
						handle="访客浏览中"
						bio="记录街头巷尾的瞬间，发现同频的伙伴，建立属于你的城市足迹。"
						location="附近漫游中"
						metrics={[
							{ label: '瞬间', value: 0 },
							{ label: '群组', value: 0 },
							{ label: '好友', value: 0 }
						]}
					>
						{#snippet actions()}
							<Button
								variant="primary"
								size="md"
								class="hit-slop-44 flex-1 gap-1.5 sm:flex-none"
								onclick={() => void goto(ROUTES.login)}
							>
								<UserRound class="size-4" strokeWidth={1.75} />
								登录 / 注册开启档案
							</Button>
							<Button
								variant="soft"
								size="md"
								class="hit-slop-44 flex-1 gap-1.5 sm:flex-none"
								onclick={() => void goto(ROUTES.nearbyMoments)}
							>
								先去附近逛逛
							</Button>
						{/snippet}
					</IdentityHero>
				</div>

				<!-- 桌面端 3 列 Bento 展示网格：充实 1280px 呼吸型宽屏，富有艺术质感与探索导流 -->
				<div class="grid grid-cols-1 gap-5 md:grid-cols-3">
					<!-- 卡片 1: 瞬间画廊 -->
					<div class="flex flex-col justify-between rounded-lg bg-surface-island p-5 border border-border-default/60 shadow-xs">
						<div>
							<div class="flex size-10 items-center justify-center rounded-md bg-accent-soft text-text-accent-on-soft">
								<Sparkles class="size-5" strokeWidth={1.75} />
							</div>
							<h3 class="mt-4 text-base font-semibold text-text-primary">记录城市瞬间</h3>
							<p class="mt-1.5 text-xs leading-relaxed text-text-secondary">
								随时发布身边的光影与心情，在地图上留下专属足迹，与同城伙伴分享生活。
							</p>
							<div class="mt-4 flex flex-wrap gap-1.5">
								<span class="rounded-full bg-surface-subtle px-2.5 py-1 text-xxs font-medium text-text-secondary">#街角光影</span>
								<span class="rounded-full bg-surface-subtle px-2.5 py-1 text-xxs font-medium text-text-secondary">#城市手记</span>
								<span class="rounded-full bg-surface-subtle px-2.5 py-1 text-xxs font-medium text-text-secondary">#夜跑日常</span>
							</div>
						</div>
						<div class="mt-6 pt-4 border-t border-border-default/40">
							<Button
								variant="ghost"
								size="sm"
								class="hit-slop -ml-2 text-accent-default hover:underline group"
								onclick={() => void goto(ROUTES.nearbyMoments)}
							>
								探索附近瞬间
								<ChevronRight class="size-4 transition-transform group-hover:translate-x-0.5" strokeWidth={1.75} />
							</Button>
						</div>
					</div>

					<!-- 卡片 2: 同好群组 -->
					<div class="flex flex-col justify-between rounded-lg bg-surface-island p-5 border border-border-default/60 shadow-xs">
						<div>
							<div class="flex size-10 items-center justify-center rounded-md bg-accent-soft text-text-accent-on-soft">
								<UsersRound class="size-5" strokeWidth={1.75} />
							</div>
							<h3 class="mt-4 text-base font-semibold text-text-primary">发现同好群组</h3>
							<p class="mt-1.5 text-xs leading-relaxed text-text-secondary">
								寻找同一片街区的骑行队、读书会与摄影社团，加入真实的城市线下社区。
							</p>
							<div class="mt-4 flex flex-col gap-2">
								<div class="flex items-center justify-between rounded-md bg-surface-subtle/80 px-2.5 py-1.5 text-xxs">
									<span class="font-medium text-text-primary">云龙湖摄影社</span>
									<span class="text-text-secondary">142 成员</span>
								</div>
								<div class="flex items-center justify-between rounded-md bg-surface-subtle/80 px-2.5 py-1.5 text-xxs">
									<span class="font-medium text-text-primary">周末骑行俱乐部</span>
									<span class="text-text-secondary">89 成员</span>
								</div>
							</div>
						</div>
						<div class="mt-6 pt-4 border-t border-border-default/40">
							<Button
								variant="ghost"
								size="sm"
								class="hit-slop -ml-2 text-accent-default hover:underline group"
								onclick={() => void goto(ROUTES.nearbyGroups)}
							>
								发现附近群组
								<ChevronRight class="size-4 transition-transform group-hover:translate-x-0.5" strokeWidth={1.75} />
							</Button>
						</div>
					</div>

					<!-- 卡片 3: 社交雷达 -->
					<div class="flex flex-col justify-between rounded-lg bg-surface-island p-5 border border-border-default/60 shadow-xs">
						<div>
							<div class="flex size-10 items-center justify-center rounded-md bg-accent-soft text-text-accent-on-soft">
								<Users class="size-5" strokeWidth={1.75} />
							</div>
							<h3 class="mt-4 text-base font-semibold text-text-primary">真实社交雷达</h3>
							<p class="mt-1.5 text-xs leading-relaxed text-text-secondary">
								基于地理位置与漫游雷达，发现此刻正在这片区域的同频漫游者。
							</p>
							<div class="mt-4 flex flex-col gap-2">
								<div class="flex items-center gap-2 rounded-md bg-surface-subtle/80 px-2.5 py-1.5 text-xxs text-text-secondary">
									<span class="size-2 rounded-full bg-status-online shrink-0"></span>
									<span>实时距离感知 · 支持私密保护</span>
								</div>
								<div class="flex items-center gap-2 rounded-md bg-surface-subtle/80 px-2.5 py-1.5 text-xxs text-text-secondary">
									<span class="size-2 rounded-full bg-accent-default shrink-0"></span>
									<span>双向确认好友 · 私信畅聊</span>
								</div>
							</div>
						</div>
						<div class="mt-6 pt-4 border-t border-border-default/40">
							<Button
								variant="ghost"
								size="sm"
								class="hit-slop -ml-2 text-accent-default hover:underline group"
								onclick={() => void goto(ROUTES.nearbyPeople)}
							>
								开启漫游雷达
								<ChevronRight class="size-4 transition-transform group-hover:translate-x-0.5" strokeWidth={1.75} />
							</Button>
						</div>
					</div>
				</div>
			</div>
		{:else if error}
			<LoadError message={error} onRetry={retry} />
		{:else}
			<div class="w-full">
				<!-- 身份封面卡：封面 + 头像 + 身份 + 简介/位置 + 指标行 + 动作区。
				     页顶留白由 contentClassName 的 pt-6（呼吸型变体）单侧承担；
				     与下方双栏的区块间距 24 由本块 mb-6 单侧提供。 -->
				<div class="mb-6">
					<IdentityHero
						mode="self"
						name={nickname || '未设置昵称'}
						{bio}
						{avatar}
						{handle}
						location={locationName || '未设置位置'}
						metrics={metricItems}
					>
						{#snippet actions()}
							<Button
								variant="primary"
								size="md"
								class="hit-slop-44 flex-1 gap-1.5 sm:flex-none"
								onclick={() => void goto(ROUTES.meEdit)}
							>
								<Pencil class="size-4" strokeWidth={1.75} />
								编辑资料
							</Button>
							<Button
								variant="soft"
								size="md"
								class="hit-slop-44 flex-1 gap-1.5 sm:flex-none"
								aria-label="系统设置"
								onclick={() => void goto(ROUTES.settings)}
							>
								<Settings class="size-4 text-text-secondary" strokeWidth={1.75} />
								设置
							</Button>
						{/snippet}
					</IdentityHero>
				</div>

				<!-- 双栏：我的内容入口 + 资料状态（分区间距由上方身份卡 mb-6 单侧提供）。 -->
				<div class="grid gap-x-5 gap-y-6 lg:grid-cols-[minmax(0,1.35fr)_minmax(17.5rem,0.65fr)]">
					<section aria-label="我的内容">
						<SectionHead title="我的内容" subtitle="你的瞬间、群组与社交关系都在这里" icon={LayoutGrid} />
						<ListContainer bordered>
							{#each ENTRY_ITEMS as item (item.id)}
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
									{#if item.count}
										<span class="shrink-0 text-xs font-medium tabular-nums text-text-secondary">{item.count}</span>
									{/if}
									<ChevronRight class="size-4 shrink-0 text-text-secondary" strokeWidth={1.5} />
								</ListItem>
							{/each}
						</ListContainer>
					</section>

					<ProfileCompleteness items={COMPLETENESS} class="self-start" />
				</div>
			</div>
		{/if}
	</StandardPageLayout>
{/if}