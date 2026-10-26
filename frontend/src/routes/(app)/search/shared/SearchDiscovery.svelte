<script lang="ts">
	import { Clock, ArrowUpRight, Flame, TrendingUp, Users, UsersRound } from '@lucide/svelte';
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

<!-- 宽屏双列：左（最近搜索 / 热门话题 / 人物推荐）+ 右（热门动态）。
     此前全部单列堆在内容列里，宽屏时内容只占中间一条、右侧大片空白（用户截图反馈）。
     窄屏自动回落单列，顺序与原先一致。断点用 `lg`（项目自定义 lg=1280 / xl=1536，
     1397px 常见笔记本宽度只到 lg）。
     纵向节奏（Round 9）：页顶留白只由页面 contentClassName pt-4 提供（原根顶距 20 与它重复，
     已删）；分区之间 24 = 左列 `flex flex-col gap-6` + 移动端列间 `gap-y-6`（父 gap 单侧提供，
     不再用子块顶距——否则首个渲染的分区会多叠一层顶距）。 -->
<!-- 超宽屏（2xl ≥1792）改「发现区横向 4 分栏 + 热门动态整行 4 卡」两段式：
     lg 双列时右栏热门动态远高于左栏，2560 下左栏底部大片留白、纵向重心偏右。
     改为两段整行后左右高度接近（发现区 ~484 + 热门动态 ~560），内容横向铺满不塌中间。 -->
<div class="grid gap-y-6 gap-x-8 lg:grid-cols-2 2xl:flex 2xl:flex-col">
<div class="flex flex-col gap-6 2xl:grid 2xl:grid-cols-4 2xl:items-start">
	<!-- 最近搜索：页面直出（eyebrow 小标 + 行动），无卡片壳——空态/少内容时
	     一张几乎空白的浮起卡会放大页面的空洞感。
	     无记录时整块隐藏（Gemini 审查 r11：干瘪的「暂无记录」文案无信息量），
	     首次搜索后自然出现 -->
	{#if recent.length > 0}
	<section aria-label="最近搜索">
		<div class="flex items-center justify-between">
			<!-- 标题图标与「热门话题」Flame 对称（Gemini 审查 r11） -->
			<h2 class="eyebrow flex items-center gap-1.5"><Clock class="size-3.5 text-accent-default" strokeWidth={1.75} />最近搜索</h2>
			<button
				type="button"
				onclick={onClear}
				class="link-btn hit-slop -my-1 px-2 py-1 text-xs text-accent-default"
			>
				清除
			</button>
		</div>
		<ListContainer bordered class="mt-3 rounded-2xl border-border-subtle/50 shadow-xs overflow-hidden">
			{#each recent as item (item)}
				<ListItem
					onClick={() => onChoose(item)}
					ariaLabel={`重新搜索 ${item}`}
					class="flex items-center gap-3 px-4 py-3"
				>
					<Clock class="size-4 text-text-secondary" strokeWidth={1.75} />
					<span class="flex-1 text-sm text-text-primary">{item}</span>
					<ArrowUpRight class="size-4 text-text-secondary" strokeWidth={1.75} />
				</ListItem>
			{/each}
		</ListContainer>
	</section>
	{/if}

	{#if hotTopics.length > 0}
		<!-- 热门话题：无界流式标签云，点击即搜，告别生硬方框束缚 -->
		<section aria-label="热门话题">
			<div class="flex items-center gap-1.5 mb-2.5">
				<Flame class="size-3.5 text-accent-default" strokeWidth={1.75} />
				<h2 class="eyebrow">热门话题</h2>
			</div>
			<div class="flex flex-wrap gap-2 pt-1">
				{#each hotTopics as topic (topic)}
					<TagPill onClick={() => onChooseTopic(topic)}>#{topic}</TagPill>
				{/each}
			</div>
		</section>
	{/if}

	{#if hotUsers.length > 0}
		<!-- 人物推荐：users/search 距离排序前 3（补发现态下半屏空白，P1-16）；
		     行原语与首页附近用户行同语言（头像角标 + #tag · 距离副标题），尾部
		     ArrowUpRight 与本页最近搜索/结果行导航语义一致 -->
		<section aria-label="人物推荐">
			<div class="flex items-center justify-between mb-2.5">
				<div class="flex items-center gap-1.5">
					<Users class="size-3.5 text-accent-default" strokeWidth={1.75} />
					<h2 class="eyebrow">人物推荐</h2>
				</div>
				<button
					type="button"
					onclick={onSeeAllUsers}
					class="link-btn hit-slop -my-1 px-2 py-1 text-xs text-accent-default"
				>
					查看全部
				</button>
			</div>
			<ListContainer bordered class="mt-3 rounded-2xl border-border-subtle/50 shadow-xs overflow-hidden">
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
								<p class="mt-1 flex items-center gap-1.5 text-xxs">
									{#if hasTag}<span class="text-accent-default">#{u.tag}</span>{/if}
									{#if hasTag && hasDistance}<span class="opacity-muted text-text-secondary">·</span>{/if}
									{#if hasDistance}<span class="text-text-secondary">{u.distance}</span>{/if}
								</p>
							{/if}
						{/snippet}
						{#snippet actions()}
							<ArrowUpRight class="size-4 text-text-secondary" strokeWidth={1.75} />
						{/snippet}
					</UserCardRow>
				{/each}
			</ListContainer>
		</section>
	{/if}

	{#if hotGroups.length > 0}
		<!-- 推荐群组：丰富发现态维度，平衡左右栏纵向节奏 -->
		<section aria-label="推荐群组">
			<div class="flex items-center justify-between mb-2.5">
				<div class="flex items-center gap-1.5">
					<UsersRound class="size-3.5 text-accent-default" strokeWidth={1.75} />
					<h2 class="eyebrow">推荐群组</h2>
				</div>
				{#if onSeeAllGroups}
					<button
						type="button"
						onclick={onSeeAllGroups}
						class="link-btn hit-slop -my-1 px-2 py-1 text-xs text-accent-default"
					>
						查看全部
					</button>
				{/if}
			</div>
			<ListContainer bordered class="mt-3 rounded-2xl border-border-subtle/50 shadow-xs overflow-hidden">
				{#each hotGroups as g (g.id)}
					<ListItem
						onClick={() => onOpenGroup?.(g.id)}
						ariaLabel={`查看 ${g.name} 群组`}
						class="flex items-center gap-3 px-4 py-3"
					>
						<Avatar src={g.avatar} name={g.name} size="md" shape="rounded" />
						<div class="flex-1 min-w-0">
							<p class="text-sm font-medium text-text-primary truncate">{g.name}</p>
							<p class="text-xxs text-text-secondary mt-0.5 flex items-center gap-1.5 truncate">
								{#if g.category}<span class="text-accent-default">#{g.category}</span>{/if}
								{#if g.category && g.memberCount > 0}<span class="opacity-muted text-text-secondary">·</span>{/if}
								<span>{g.memberCount} 成员</span>
							</p>
						</div>
						<ArrowUpRight class="size-4 text-text-secondary" strokeWidth={1.75} />
					</ListItem>
				{/each}
			</ListContainer>
		</section>
	{/if}

	</div>
	{#if hotMoments.length > 0}
	<div>
		<!-- 热门动态：独立画廊卡片流，宽屏下响应式双列布局与左栏高度对齐 -->
		<section aria-label="热门动态">
			<div class="flex items-center gap-1.5 mb-2.5">
				<TrendingUp class="size-3.5 text-accent-default" strokeWidth={1.75} />
				<h2 class="eyebrow">热门动态</h2>
			</div>
			<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-1 xl:grid-cols-2 2xl:grid-cols-4 gap-3.5 sm:gap-4 mt-2">
				{#each hotMoments as m (m.moment_id)}
					<HomeFeedItem moment={m} showDistance={false} onClick={() => onOpenMoment(m.moment_id)} />
				{/each}
			</div>
		</section>
	</div>
	{/if}
</div>
