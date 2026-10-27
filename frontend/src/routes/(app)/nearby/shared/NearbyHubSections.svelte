<script lang="ts">
	import { goto } from '$app/navigation';
	import { ROUTES } from '@/lib/routes';
	import { Sparkles, Users, UsersRound } from '@lucide/svelte';
	import { platform } from '@/stores/platform.svelte';
	import SectionHead from '@/components/home/SectionHead.svelte';
	import NearbyUserRows from '@/components/home/NearbyUserRows.svelte';
	import UserCard from '@/routes/(app)/nearby/people/UserCard.svelte';
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
		userPreview,
		momentPreview = 3,
		emptyHint,
		isFollowed,
		isPending,
		onToggleFollow,
		onSeeAllUsers,
		onSeeAllMoments,
		onSeeAllGroups,
		onOpenMoment,
		onOpenUser
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
		onOpenUser?: (userId: string) => void;
	} = $props();

	// 只有手机用「全宽行」（宽度有限，行式信息密度最高）；平板/桌面一律卡片网格，
	// 否则单行横跨 1100px（名字贴左、关注贴右）会读成一排空岛（设计评审 P1-4）。
	const devicePlatform = $derived(platform.platform);
	const useRows = $derived(devicePlatform === 'mobile');
	// 桌面 6（宽屏一行铺满）/ 平板与手机 4（移动端行式列表、平板卡片网格都取 4 条预览）
	const defaultUserPreview = $derived(devicePlatform === 'desktop' ? 6 : 4);
	const previewCount = $derived(userPreview ?? defaultUserPreview);
</script>

{#if users.length > 0}
	<div data-hub-section="people">
		<!-- 三段分区卡头统一 SectionHead（icon + count + 查看全部）；标题→内容间距由卡头 mb-3 承担 -->
		<SectionHead title="附近的人" icon={Users} count={users.length} action="查看全部" onAction={onSeeAllUsers} />
		{#if !useRows}
			<div class="grid grid-cols-2 gap-3.5 sm:grid-cols-3 sm:gap-4 lg:grid-cols-4 xl:grid-cols-6">
				{#each users.slice(0, previewCount) as user (user.id)}
					<UserCard
						{user}
						followed={isFollowed(user.id)}
						pending={isPending(user.id)}
						onFollow={() => onToggleFollow(user.id)}
						onNavigate={() => (onOpenUser ? onOpenUser(user.id) : void goto(ROUTES.user(user.id)))}
					/>
				{/each}
			</div>
		{:else}
			<NearbyUserRows
				users={users.slice(0, previewCount)}
				{isFollowed}
				{isPending}
				onToggle={onToggleFollow}
				onOpenUser={onOpenUser}
				{emptyHint}
			/>
		{/if}
	</div>
{/if}

{#if moments.length > 0}
	<div data-hub-section="moments">
		<SectionHead title="附近瞬间" icon={Sparkles} count={moments.length} action="查看全部" onAction={onSeeAllMoments} />
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3.5 sm:gap-4">
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
		<!-- 「附近群组」标题由轮播组件自带（SectionHead + 查看全部），此处不再重复标题 -->
		<GroupCarousel {groups} icon={UsersRound} count={groups.length} actionLabel="查看全部" onSeeAll={onSeeAllGroups} />
	</div>
{/if}
