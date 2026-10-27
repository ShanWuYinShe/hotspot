<script lang="ts">
	import { Clock, ChevronRight, Flame, TrendingUp, Users, UsersRound } from '@lucide/svelte';
	import SectionHead from '@/components/home/SectionHead.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import TagPill from '@/components/forms/TagPill.svelte';
	import HomeFeedItem from '@/components/home/HomeFeedItem.svelte';
	import UserCardRow from '@/components/profile/UserCardRow.svelte';
	import OnlineDot from '@/components/data-display/OnlineDot.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
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

<!-- 发现态布局：默认单列；md（960）起双列（窄平板 768–959 仍单列，避免两列被压成窄条），
     2xl 超宽屏改「发现区横向 4 分栏 + 热门动态整行 4 卡」两段式（左栏不再底部大片留白）。
     卡头统一 SectionHead（icon + 标题 + 可选动作），不再手写 eyebrow 标题行。 -->
<div class="grid gap-y-6 gap-x-8 md:grid-cols-2 2xl:flex 2xl:flex-col">
<div class="flex flex-col gap-6 2xl:grid 2xl:grid-cols-4 2xl:items-start">
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

	{#if hotTopics.length > 0}
		<!-- 热门话题：无界流式标签云（chip 本身即操作，不套卡片壳） -->
		<section aria-label="热门话题">
			<SectionHead title="热门话题" icon={Flame} />
			<div class="flex flex-wrap gap-2">
				{#each hotTopics as topic (topic)}
					<TagPill onClick={() => onChooseTopic(topic)}>#{topic}</TagPill>
				{/each}
			</div>
		</section>
	{/if}

	{#if hotUsers.length > 0}
		<!-- 人物推荐：users/search 距离排序前 3（头像角标 + #tag · 距离 + 尾部导航箭头） -->
		<section aria-label="人物推荐">
			<SectionHead title="人物推荐" icon={Users} action="查看全部" onAction={onSeeAllUsers} />
			<ListContainer bordered>
				{#each hotUsers as u (u.id)}
					<UserCardRow
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
					</UserCardRow>
				{/each}
			</ListContainer>
		</section>
	{/if}

	{#if hotGroups.length > 0}
		<!-- 推荐群组：与人物推荐同栅格语言，平衡发现态左右栏纵向节奏 -->
		<section aria-label="推荐群组">
			{#if onSeeAllGroups}
				<SectionHead title="推荐群组" icon={UsersRound} action="查看全部" onAction={onSeeAllGroups} />
			{:else}
				<SectionHead title="推荐群组" icon={UsersRound} />
			{/if}
			<ListContainer bordered>
				{#each hotGroups as g (g.id)}
					<ListItem
						onClick={() => onOpenGroup?.(g.id)}
						ariaLabel={`查看 ${g.name} 群组`}
						class="flex items-center gap-3 px-4 py-3"
					>
						<Avatar src={g.avatar} name={g.name} size="md" shape="rounded" />
						<div class="flex-1 min-w-0">
							<p class="truncate text-sm font-medium text-text-primary">{g.name}</p>
							<p class="mt-0.5 flex items-center gap-1.5 truncate text-xs">
								{#if g.category}<span class="text-accent-default">#{g.category}</span>{/if}
								{#if g.category && g.memberCount > 0}<span class="opacity-muted text-text-secondary">·</span>{/if}
								<span class="tabular-nums text-text-secondary">{g.memberCount} 成员</span>
							</p>
						</div>
						<ChevronRight class="size-4 text-text-secondary" strokeWidth={1.75} />
					</ListItem>
				{/each}
			</ListContainer>
		</section>
	{/if}
	</div>

	{#if hotMoments.length > 0}
	<div>
		<!-- 热门动态画廊：任何断点都 ≥2 张（避免单卡 + 大片空白）：
		     手机单列（与首页信息流同宽）→ 平板起 2 列（768 单列页下不再出现整屏大卡）→
		     lg 双列 2×2（与左栏发现区高度齐平）→ 2xl 与发现区同宽成 4 列一行 -->
		<section aria-label="热门动态">
			<SectionHead title="热门动态" icon={TrendingUp} />
			<div class="grid grid-cols-1 gap-3.5 sm:grid-cols-2 sm:gap-4 2xl:grid-cols-4">
				{#each hotMoments as m, i (m.moment_id)}
					<div>
						<HomeFeedItem moment={m} showDistance={false} priority={i < 2} onClick={() => onOpenMoment(m.moment_id)} />
					</div>
				{/each}
			</div>
		</section>
	</div>
	{/if}
</div>