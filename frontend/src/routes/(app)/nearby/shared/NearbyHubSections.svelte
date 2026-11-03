<script lang="ts">
	import { goto } from '$app/navigation';
	import { ROUTES } from '@/lib/routes';
	import { Sparkles, Users, UsersRound } from '@lucide/svelte';
	import { platform } from '@/stores/platform.svelte';
	import SectionHead from '@/components/home/SectionHead.svelte';
	import AdaptiveCardGrid from '@/components/layout/AdaptiveCardGrid.svelte';
	import NearbyUserRows from '@/components/home/NearbyUserRows.svelte';
	import UserCard from '@/components/data-display/UserCard.svelte';
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
		momentPreview,
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
	// 瞬间预览按栅格列数给：2560 下 6 列、3 条会留下半行空白；平板 4、手机 3。
	const momentPreviewCount = $derived(
		momentPreview ?? (devicePlatform === 'desktop' ? 6 : devicePlatform === 'tablet' ? 4 : 3)
	);
	// 枢纽页只是 6 条预览，不做 BSP 大跨度：2x2/2x1 与固定预览条数组合会在宽屏
	// 留下整行/整列空洞。统一 1x1，一行铺满即止；完整 BSP 留给 /nearby/people。
</script>

{#if users.length > 0}
	<div data-hub-section="people">
		<!-- 三段分区卡头统一 SectionHead（icon + count + 查看全部）；标题→内容间距由卡头 mb-3 承担 -->
		<SectionHead
			title="附近的人"
			icon={Users}
			count={users.length}
			action={users.length > previewCount ? '查看全部' : undefined}
			onAction={users.length > previewCount ? onSeeAllUsers : undefined}
		/>
		{#if !useRows}
			<AdaptiveCardGrid
				count={Math.min(users.length, previewCount)}
				minTrackRem={18.75}
				gapRem={[0.875, 1, 1]}
				class="grid-flow-dense"
				style="grid-auto-rows: clamp(12.1875rem, 20vw, 17.1875rem);"
			>
				{#each users.slice(0, previewCount) as user, index (user.id)}
					<div class="h-full w-full">
						<UserCard
							{user}
							priority={index < 2}
							followed={isFollowed(user.id)}
							pending={isPending(user.id)}
							onFollow={() => onToggleFollow(user.id)}
							onNavigate={() => (onOpenUser ? onOpenUser(user.id) : void goto(ROUTES.user(user.id)))}
							class="h-full w-full"
						/>
					</div>
				{/each}
			</AdaptiveCardGrid>
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
		<SectionHead
			title="附近瞬间"
			icon={Sparkles}
			count={moments.length}
			action={moments.length > momentPreviewCount ? '查看全部' : undefined}
			onAction={moments.length > momentPreviewCount ? onSeeAllMoments : undefined}
		/>
		<AdaptiveCardGrid
			count={Math.min(moments.length, momentPreviewCount)}
			minTrackRem={21.25}
			gapRem={[0.875, 1, 1]}
		>
			{#each moments.slice(0, momentPreviewCount) as moment (moment.moment_id)}
				<HomeFeedItem
					{moment}
					showDistance={true}
					onClick={() => onOpenMoment(moment.moment_id)}
				/>
			{/each}
		</AdaptiveCardGrid>
	</div>
{/if}

{#if groups.length > 0}
	<div data-hub-section="groups">
		<!-- 「附近群组」标题由轮播组件自带（SectionHead + 查看全部），此处不再重复标题 -->
		<GroupCarousel
			{groups}
			icon={UsersRound}
			count={groups.length}
			actionLabel="查看全部"
			showAction={groups.length > 6}
			onSeeAll={onSeeAllGroups}
		/>
	</div>
{/if}
