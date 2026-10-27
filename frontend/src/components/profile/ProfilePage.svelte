<script lang="ts">
	import { goto } from '$app/navigation';
	import { ChevronRight, LayoutGrid, Pencil, Settings, Sparkles, Users, UsersRound } from '@lucide/svelte';
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

	// 指标行：数字是资料页的节奏锚点（MetricRow strip：大数字 + 小标签 + 发丝分隔）
	const metricItems = $derived([
		{ label: '瞬间', value: stats.moments_count, onClick: () => void goto(ME_SECTION_PATHS.moments) },
		{ label: '粉丝', value: stats.followers_count, onClick: () => void goto(ME_SECTION_PATHS.relationships) },
		{ label: '群组', value: stats.groups_count, onClick: () => void goto(ME_SECTION_PATHS.groups) }
	]);

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
	<!-- 分区模式：StandardPageLayout + sticky PageHeader；
	     Section 组件为纯内容（不自建滚动/gutter），gutter 由 ContentColumn 统一提供 -->
	<StandardPageLayout contentClassName="animate-fade-in pt-4 pb-6">
		{#snippet header()}
			<PageHeader title={meta.title} sticky titleSize={device === 'desktop' ? 'xl' : '2xl'} onBack={() => smartBack(ROUTES.me)} />
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
	<StandardPageLayout contentClassName="animate-fade-in pt-6 pb-8">
		{#snippet header()}
			<!-- 页头不放操作按钮：编辑资料/设置统一收在身份卡的动作区，
			     同一页同一功能只允许一个入口。外观控制不进身份卡：桌面顶栏已有常驻
			     「外观设置」，其余设备走设置页「外观与显示」。 -->
			<PageHeader
				title="个人中心"
				subtitle="你的公开资料、内容与关系的入口"
				titleSize={device === 'desktop' ? 'xl' : '2xl'}
			/>
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
			<div class="mx-auto w-full max-w-5xl xl:max-w-6xl 2xl:max-w-7xl">
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
				<div class="grid gap-x-5 gap-y-6 lg:grid-cols-[minmax(0,1.35fr)_minmax(280px,0.65fr)]">
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