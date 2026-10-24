<script lang="ts">
	import SectionHead from '@/components/home/SectionHead.svelte';
	import NearbyUserRows from '@/components/home/NearbyUserRows.svelte';
	import GroupCarousel, { type NearbyGroup } from '@/components/home/GroupCarousel.svelte';
	import HomeFeedItem from '@/components/home/HomeFeedItem.svelte';
	import type { MomentItem } from '@/api/types';
	import type { NearbyUser } from '@/components/home/nearby-users';

	// 「附近」枢纽页的三段分区（2026-10-19 IA 调整：由首页迁来）。
	// 纯展示：数据、关注态与导航回调全部由页面注入，便于用例锁定分区与预览条数。
	// 段序按「谁在附近 → 附近在发生什么 → 有哪些圈子」：附近的人是本 Tab 的历史落点
	// 与 LBS 主场景，故列首；三个叶子页仍是完整列表/筛选的归属，此处只做预览 + 入口。
	// 空数据段整段不渲染（与首页/搜索发现态同口径，不留空框）。

	let {
		users = [],
		moments = [],
		groups = [],
		userPreview = 4,
		momentPreview = 3,
		emptyHint,
		isFollowed,
		isPending,
		onToggleFollow,
		onSeeAllUsers,
		onSeeAllMoments,
		onSeeAllGroups,
		onOpenMoment
	}: {
		users?: NearbyUser[];
		moments?: MomentItem[];
		groups?: NearbyGroup[];
		userPreview?: number;
		momentPreview?: number;
		emptyHint?: string;
		isFollowed: (id: string) => boolean;
		isPending: (id: string) => boolean;
		onToggleFollow: (id: string) => void;
		onSeeAllUsers: () => void;
		onSeeAllMoments: () => void;
		onSeeAllGroups: () => void;
		onOpenMoment: (momentId: string) => void;
	} = $props();
</script>

{#if users.length > 0}
	<div data-hub-section="people">
		<SectionHead title="附近的人" action="查看全部" onAction={onSeeAllUsers} />
		<NearbyUserRows
			users={users.slice(0, userPreview)}
			{isFollowed}
			{isPending}
			onToggle={onToggleFollow}
			{emptyHint}
		/>
	</div>
{/if}

{#if moments.length > 0}
	<div data-hub-section="moments">
		<SectionHead title="附近瞬间" action="查看全部" onAction={onSeeAllMoments} />
		<div class="flex flex-col gap-3.5 sm:gap-4 mt-2">
			{#each moments.slice(0, momentPreview) as moment (moment.moment_id)}
				<HomeFeedItem
					{moment}
					showDistance={true}
					onClick={() => onOpenMoment(moment.moment_id)}
				/>
			{/each}
		</div>
	</div>
{/if}

{#if groups.length > 0}
	<div data-hub-section="groups">
		<!-- 「附近群组」标题由轮播组件自带（含「更多」入口），此处不再重复标题 -->
		<GroupCarousel {groups} onSeeAll={onSeeAllGroups} />
	</div>
{/if}
