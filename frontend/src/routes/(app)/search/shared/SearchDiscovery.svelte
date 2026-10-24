<script lang="ts">
	import { Clock, ArrowUpRight, Flame, TrendingUp, Users } from '@lucide/svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import TagPill from '@/components/forms/TagPill.svelte';
	import ActivityRow from '@/components/home/ActivityRow.svelte';
	import UserCardRow from '@/components/profile/UserCardRow.svelte';
	import OnlineDot from '@/components/data-display/OnlineDot.svelte';
	import type { MomentItem } from '@/api/types';

	// 搜索页「无查询发现态」：最近搜索 + 热门话题 + 人物推荐 + 热门动态。
	// 热门话题来自 moments/search 热门模式的 tags 聚合（页面传入）；人物推荐来自
	// users/search 距离排序（页面传入）；无数据时各自不渲染区块（诚实空态原则，
	// 不留死区块）。人物推荐补发现态下半屏空白（审查 P1-16：54% 空白）。
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

	let {
		recent,
		hotTopics = [],
		hotMoments = [],
		hotUsers = [],
		onChoose,
		onChooseTopic,
		onOpenMoment,
		onOpenUser,
		onSeeAllUsers,
		onClear
	}: {
		recent: string[];
		/** 热门话题（为空时不渲染区块） */
		hotTopics?: string[];
		/** 热门动态（为空时不渲染区块）：发现态本就拉取的热门响应复用，零额外请求 */
		hotMoments?: MomentItem[];
		/** 人物推荐（为空时不渲染区块）：users/search 距离排序取前 3 */
		hotUsers?: DiscoveryUser[];
		onChoose: (value: string) => void;
		/** 热门话题点击：按 tag 语义搜索（moments/search tag 参数），与关键词搜索分流 */
		onChooseTopic: (value: string) => void;
		/** 热门动态点击：跳动态详情 */
		onOpenMoment: (momentId: string) => void;
		/** 人物推荐行点击：跳用户资料页 */
		onOpenUser: (userId: string) => void;
		/** 「查看全部」：跳附近的人专区 */
		onSeeAllUsers: () => void;
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
<div class="grid gap-y-6 gap-x-8 lg:grid-cols-2">
<div class="flex flex-col gap-6">
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
		<ListContainer class="mt-3 divide-y divide-border-subtle/30">
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
		<!-- 热门话题：热门动态 tags 聚合，点击即搜 -->
		<section aria-label="热门话题">
			<div class="flex items-center gap-1.5">
				<Flame class="size-3.5 text-accent-default" strokeWidth={1.75} />
				<h2 class="eyebrow">热门话题</h2>
			</div>
			<div class="mt-3 flex flex-wrap gap-2">
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
			<div class="flex items-center justify-between">
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
			<ListContainer class="mt-3 divide-y divide-border-subtle/30">
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

	</div>
	{#if hotMoments.length > 0}
	<div>
		<!-- 热门动态：复用发现态已拉取的热门响应（C2 轻量充实），行原语与首页附近活动一致 -->
		<section aria-label="热门动态">
			<div class="flex items-center gap-1.5">
				<TrendingUp class="size-3.5 text-accent-default" strokeWidth={1.75} />
				<h2 class="eyebrow">热门动态</h2>
			</div>
			<ListContainer class="mt-3 divide-y divide-border-subtle/30">
				{#each hotMoments as m, i (m.moment_id)}
					<ActivityRow moment={m} index={i} onClick={() => onOpenMoment(m.moment_id)} />
				{/each}
			</ListContainer>
		</section>
	</div>
	{/if}
</div>
