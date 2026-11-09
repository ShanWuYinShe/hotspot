<script lang="ts">
	import { Clock, ChevronRight, Flame, TrendingUp, Users, UsersRound } from '@lucide/svelte';
	import SectionHead from '@/components/home/SectionHead.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import TagPill from '@/components/forms/TagPill.svelte';
	import HomeFeedItem from '@/components/home/HomeFeedItem.svelte';
	import EntityRow from '@/components/list/EntityRow.svelte';
	import OnlineDot from '@/components/data-display/OnlineDot.svelte';
	import Skeleton from '@/components/data-display/Skeleton.svelte';
	import SkeletonFeed from '@/components/data-display/SkeletonFeed.svelte';
	import SkeletonList from '@/components/data-display/SkeletonList.svelte';
	import type { MomentItem } from '@/api/types';

	// 搜索页「无查询发现态」：最近搜索 + 热门话题 + 人物推荐 + 推荐群组 + 热门动态。
	// 热门话题来自 moments/search 热门模式的 tags 聚合（页面传入）；人物推荐来自
	// users/search 距离排序（页面传入）；推荐群组来自 groups/search 热门排序（页面传入）；
	// 无数据时各自不渲染区块（诚实空态原则，不留死区块）。
	// 页面私有组件（colocate），由 search/+page.svelte 渲染。

	/** 人物推荐列表项（users/search 距离排序，页面映射后传入） */
	export interface DiscoveryUser {
		id: string;
		name: string;
		avatar: string;
		/** 首个兴趣标签（可为空） */
		tag: string;
		/** formatDistance 格式化后的距离文案（可为空） */
		distance: string;
		isOnline: boolean;
	}

	/** 推荐群组列表项（groups/search 热门排序，页面映射后传入） */
	export interface DiscoveryGroup {
		id: string;
		name: string;
		avatar: string;
		category: string;
		memberCount: number;
	}

	let {
		recent,
		hotTopics = [],
		hotMoments = [],
		hotUsers = [],
		hotGroups = [],
		topicsLoading = false,
		momentsLoading = false,
		usersLoading = false,
		groupsLoading = false,
		onChoose,
		onChooseTopic,
		onOpenMoment,
		onOpenUser,
		onOpenGroup,
		onSeeAllUsers,
		onSeeAllGroups,
		onClear
	}: {
		recent: string[];
		/** 热门话题（为空时不渲染区块） */
		hotTopics?: string[];
		/** 热门动态（为空时不渲染区块）：发现态本就拉取的热门响应复用，零额外请求 */
		hotMoments?: MomentItem[];
		/** 人物推荐（为空时不渲染区块）：users/search 距离排序取前 3 */
		hotUsers?: DiscoveryUser[];
		/** 推荐群组（为空时不渲染区块）：groups/search 热门排序取前 3 */
		hotGroups?: DiscoveryGroup[];
		/** 加载期骨架（R17 CLS 整改）：数据落定前先在最终位置渲染与真实内容等高的骨架，
		 *  数据到达只替换块内内容（分区不再按到达顺序插入/整段下移）。为 false 且数据
		 *  为空时该区块不渲染（诚实空态，不留死骨架）。 */
		topicsLoading?: boolean;
		momentsLoading?: boolean;
		usersLoading?: boolean;
		groupsLoading?: boolean;
		onChoose: (value: string) => void;
		/** 热门话题点击：按 tag 语义搜索（moments/search tag 参数），与关键词搜索分流 */
		onChooseTopic: (value: string) => void;
		/** 热门动态点击：跳动态详情 */
		onOpenMoment: (momentId: string) => void;
		/** 人物推荐行点击：跳用户资料页 */
		onOpenUser: (userId: string) => void;
		/** 推荐群组行点击：跳群组资料页 */
		onOpenGroup?: (groupId: string) => void;
		/** 「查看全部」：跳附近的人专区 */
		onSeeAllUsers: () => void;
		/** 「查看全部」：跳附近的群组专区 */
		onSeeAllGroups?: () => void;
		onClear: () => void;
	} = $props();
</script>

<!-- 发现态布局：上下两段、每段各自网格——发现区（最近搜索/热门话题/人物/群组）在上，
     热门动态整行在下。此前 md–2xl 是「左发现 / 右动态」两列，左栏内容随数据结束
     （常见 y≈452）而右栏动态 2×2 更高，左栏底部留下 ~330px 空洞；改上下两段后
     两段都按容器宽度摊平，不再出现半屏空场。
     分栏数用 auto-fit 自适应：空分区（如 groups/search 无数据时不渲染推荐群组）轨道折叠为 0，
     由其余分区摊平分栏宽度，不再固定 4 栏留出 1 栏空洞（审计 shell-home F4）。
     卡头统一 SectionHead（icon + 标题 + 可选动作），不再手写 eyebrow 标题行。 -->
<div class="flex flex-col gap-6">
<div class="grid gap-6 md:grid-cols-[repeat(auto-fit,minmax(18rem,1fr))] md:items-start">
	<!-- 最近搜索：无记录时整块隐藏（干瘪的「暂无记录」无信息量），首次搜索后自然出现 -->
	{#if recent.length > 0}
	<section aria-label="最近搜索">
		<SectionHead title="最近搜索" icon={Clock} action="清除" onAction={onClear} />
		<ListContainer bordered>
			{#each recent as item (item)}
				<ListItem
					onClick={() => onChoose(item)}
					ariaLabel={`重新搜索 ${item}`}
					class="flex items-center gap-3 px-4 py-3"
				>
					<Clock class="size-4 text-text-secondary" strokeWidth={1.75} />
					<span class="flex-1 text-sm text-text-primary">{item}</span>
					<ChevronRight class="size-4 text-text-secondary" strokeWidth={1.75} />
				</ListItem>
			{/each}
		</ListContainer>
	</section>
	{/if}

	{#if topicsLoading || hotTopics.length > 0}
		<!-- 热门话题：无界流式标签云（chip 本身即操作，不套卡片壳）。加载期骨架复刻真实
		     chip 几何（h-7 / gap-2 / 约 55px 宽）：390 下折成 2 行、md+ 收成 1 行，与真实
		     标签云的换行结果一致，数据到达时区块高度不变。 -->
		<section aria-label="热门话题">
			<SectionHead title="热门话题" icon={Flame} />
			{#if topicsLoading}
				<div role="status" aria-busy="true" aria-label="加载热门话题…">
					<div class="flex flex-wrap gap-2" aria-hidden="true">
						{#each [0, 1, 2, 3, 4, 5] as i (i)}
							<Skeleton class="h-7 w-[3.4375rem] rounded-full" />
						{/each}
					</div>
				</div>
			{:else}
				<div class="flex flex-wrap gap-2">
					{#each hotTopics as topic (topic)}
						<TagPill onClick={() => onChooseTopic(topic)}>#{topic}</TagPill>
					{/each}
				</div>
			{/if}
		</section>
	{/if}

	{#if usersLoading || hotUsers.length > 0}
		<!-- 人物推荐：users/search 距离排序前 3（头像角标 + #tag · 距离 + 尾部导航箭头）。
		     加载期骨架复用 SkeletonList（Avatar 44 + 上下 12 = 行高 68、行间 2、软岛 p-2），
		     与 EntityRow 行几何一致 → 数据到达零位移。 -->
		<section aria-label="人物推荐">
			<SectionHead title="人物推荐" icon={Users} action="查看全部" onAction={onSeeAllUsers} />
			{#if usersLoading}
				<ListContainer bordered>
					<SkeletonList rows={3} leading={44} label="加载人物推荐…" rowClass="p-3 mb-0.5 last:mb-0" />
				</ListContainer>
			{:else}
				<ListContainer bordered>
					{#each hotUsers as u (u.id)}
						<EntityRow
							name={u.name}
							avatar={u.avatar}
							onClick={() => onOpenUser(u.id)}
							ariaLabel={`查看 ${u.name} 的资料`}
						>
							{#snippet avatarBadge()}
								{#if u.isOnline}
									<OnlineDot size="md" />
								{/if}
							{/snippet}
							{#snippet subtitle()}
								{@const hasTag = Boolean(u.tag)}
								{@const hasDistance = Boolean(u.distance)}
								{#if hasTag || hasDistance}
									<p class="mt-1 flex items-center gap-1.5 text-xs">
										{#if hasTag}<span class="text-accent-default">#{u.tag}</span>{/if}
										{#if hasTag && hasDistance}<span class="opacity-muted text-text-secondary">·</span>{/if}
										{#if hasDistance}<span class="text-text-secondary tabular-nums">{u.distance}</span>{/if}
									</p>
								{/if}
							{/snippet}
							{#snippet actions()}
								<ChevronRight class="size-4 text-text-secondary" strokeWidth={1.75} />
							{/snippet}
						</EntityRow>
					{/each}
				</ListContainer>
			{/if}
		</section>
	{/if}

	{#if groupsLoading || hotGroups.length > 0}
		<!-- 推荐群组：与人物推荐同栅格语言，平衡发现态左右栏纵向节奏。
		     加载期骨架复用 SkeletonList（Avatar 44 + p-3 = 行高 68、行间 2、软岛 p-2），
		     与 EntityRow 行几何一致 → 数据到达零位移。 -->
		<section aria-label="推荐群组">
			{#if onSeeAllGroups}
				<SectionHead title="推荐群组" icon={UsersRound} action="查看全部" onAction={onSeeAllGroups} />
			{:else}
				<SectionHead title="推荐群组" icon={UsersRound} />
			{/if}
			{#if groupsLoading}
				<ListContainer bordered>
					<SkeletonList rows={3} leading={44} label="加载推荐群组…" rowClass="p-3 mb-0.5 last:mb-0" />
				</ListContainer>
			{:else}
				<ListContainer bordered>
						{#each hotGroups as g (g.id)}
							<EntityRow
								avatarShape="rounded"
								name={g.name}
								avatar={g.avatar}
								ariaLabel={`查看 ${g.name} 群组`}
								onClick={() => onOpenGroup?.(g.id)}
							>
								{#snippet subtitle()}
									<p class="mt-1 flex items-center gap-1.5 truncate text-xs">
										{#if g.category}<span class="text-accent-default">#{g.category}</span>{/if}
										{#if g.category && g.memberCount > 0}<span class="opacity-muted text-text-secondary">·</span>{/if}
										<span class="tabular-nums text-text-secondary">{g.memberCount} 成员</span>
									</p>
								{/snippet}
								{#snippet actions()}
									<ChevronRight class="size-4 text-text-secondary" strokeWidth={1.75} />
								{/snippet}
							</EntityRow>
						{/each}
				</ListContainer>
			{/if}
		</section>
	{/if}
	</div>

	{#if momentsLoading || hotMoments.length > 0}
	<div>
		<!-- 热门动态画廊：任何断点都 ≥2 张（避免单卡 + 大片空白）：
		     手机单列（与首页信息流同宽）→ 平板起 2 列（768 单列页下不再出现整屏大卡）→
		     lg 双列 2×2（与左栏发现区高度齐平）→ 2xl 与发现区同宽成 4 列一行。
		     加载期骨架复用 SkeletonFeed（与真实卡同栅格/同 3:4·4:5 比例/同圆角），
		     数据到达只替换卡片内容 → 首屏高度不变。 -->
		<section aria-label="热门动态">
			<SectionHead title="热门动态" icon={TrendingUp} />
			{#if momentsLoading}
				<SkeletonFeed
					variant="feed"
					count={4}
					label="加载热门动态…"
					class="grid grid-cols-1 gap-3.5 sm:grid-cols-2 sm:gap-4 lg:grid-cols-3 xl:grid-cols-4"
					cardClass="aspect-[3/4] sm:aspect-[4/5]"
				/>
			{:else}
				<div class="grid grid-cols-1 gap-3.5 sm:grid-cols-2 sm:gap-4 lg:grid-cols-3 xl:grid-cols-4">
					{#each hotMoments as m, i (m.moment_id)}
						<div>
							<HomeFeedItem moment={m} showDistance={false} priority={i < 2} onClick={() => onOpenMoment(m.moment_id)} />
						</div>
					{/each}
				</div>
			{/if}
		</section>
	</div>
	{/if}
</div>