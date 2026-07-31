<script lang="ts">
	import { goto } from '$app/navigation';
	import { Share2, MapPin, Camera, Pencil } from '@lucide/svelte';
	import { platformStore } from '@/stores/platform.svelte';
	import AppearanceMenu from '@/components/layout/AppearanceMenu.svelte';
	import ContextRail from '@/components/layout/ContextRail.svelte';
	import TabletRail from '@/components/layout/tablet-rail/TabletRail.svelte';
	import StreakCard from '@/components/layout/tablet-rail/StreakCard.svelte';
	import VisitorsCard from '@/components/layout/tablet-rail/VisitorsCard.svelte';
	import MoodCheckinCard from '@/components/layout/tablet-rail/MoodCheckinCard.svelte';
	import AchievementCard from '@/components/layout/tablet-rail/AchievementCard.svelte';
	import SuggestedFollowsCard from '@/components/layout/tablet-rail/SuggestedFollowsCard.svelte';
	import VibeCard from '@/components/layout/tablet-rail/VibeCard.svelte';
	import ActionIconButton from '@/components/shared/ActionIconButton.svelte';
	import ProfileTabs, { type Tab } from '@/components/profile/ProfileTabs.svelte';
	import GalleryGrid from '@/components/profile/GalleryGrid.svelte';
	import { useAccountData, useGalleryPosts, useReactedPosts, useFavoritePosts, type GalleryPost } from '@/lib/profile-data.svelte';
	import { ROUTES } from '@/lib/routes';

	// 移动/平板个人主页：Hero 身份卡 + Tab 画廊 + 平板右侧陪伴栏。
	// 迁移自旧版 ProfileFeature 内 ProfileMobile。

	const { avatar, nickname, stats } = useAccountData();
	const myPosts = useGalleryPosts('my');
	const reacted = useReactedPosts();
	const savedP = useFavoritePosts();

	let activeTab = $state<Tab>('posts');
	const postsMap = $derived({ posts: myPosts.posts, liked: reacted.posts, saved: savedP.posts } as Record<Tab, GalleryPost[]>);

	const isTablet = $derived(platformStore.platform === 'tablet');
	const tabletShellStyle = $derived(isTablet ? 'max-width:884px;width:100%;' : '');
	const mainColStyle = $derived(isTablet ? 'max-width:560px;' : '');
	const containerStyle = $derived(
		isTablet ? 'max-width:560px;margin-left:auto;margin-right:auto;width:100%;' : ''
	);
</script>

<div class="sticky top-0 z-20 py-3 bg-[var(--neu-bg)]" style="border-bottom:1px solid var(--neu-border);">
	<div class="{isTablet ? 'mx-auto px-0 flex gap-6' : 'px-5'}" style={tabletShellStyle}>
		<div class="{isTablet ? 'mx-auto w-full lg:mx-0 min-w-0 px-5' : 'w-full px-0'}" style={mainColStyle}>
			<div class="flex items-center justify-between">
				<div>
					<p class="text-[var(--neu-text)] text-lg">鹿鸣</p>
					<p class="text-[var(--neu-text-muted)] text-[11px] mt-px">@elara.v</p>
				</div>
				<div class="flex gap-2">
					<ActionIconButton icon={Share2} label="分享个人主页" size={36} />
					<AppearanceMenu />
				</div>
			</div>
		</div>
		{#if isTablet}
			<div class="hidden lg:block flex-shrink-0" style="width:var(--context-rail-width);"></div>
		{/if}
	</div>
</div>

<div class="animate-fade-in {isTablet ? 'mx-auto px-0 flex gap-6 pb-6 pt-5' : 'flex flex-col w-full pb-6'}" style={tabletShellStyle}>
	<!-- Main column -->
	<div class="{isTablet ? 'mx-auto w-full lg:mx-0 min-w-0 flex flex-col' : 'flex flex-col w-full'}" style={mainColStyle}>
		<!-- Hero identity card -->
		<div class={isTablet ? 'px-5' : 'px-5 pt-5'}>
			<div class="neu-ambient-glow">
				<div class="neu-surface-hero rounded-[24px] flex flex-col items-center" style="padding:24px 20px 18px;">
					<!-- Avatar -->
					<div class="relative mb-3">
						<div
							class="rounded-full overflow-hidden"
							style="width:92px;height:92px;border:3px solid var(--neu-bg);box-shadow:0 6px 18px var(--neu-glow);"
						>
							{#if avatar}
								<img src={avatar} alt={nickname || '用户'} class="w-full h-full object-cover" />
							{:else}
								<span class="text-[var(--neu-text-muted)] text-[32px]">{(nickname || '?')[0]}</span>
							{/if}
						</div>
						<button
							type="button"
							class="neu-hover-btn absolute bottom-0 right-0 flex items-center justify-center rounded-full outline-none bg-[var(--neu-accent-primary)]"
							style="width:26px;height:26px;border:2.5px solid var(--neu-bg);"
						>
							<Camera style="width:12px;height:12px;color:white;" strokeWidth={2} />
						</button>
					</div>

					<!-- Identity -->
					<h2 class="text-[var(--neu-text)] text-[22px]">鹿鸣</h2>
					<span class="text-[var(--neu-text-muted)] text-xs mt-0.5">@elara.v</span>
					<p class="text-[var(--neu-text-muted)] text-xs text-center pt-2.5 px-5 pb-1 leading-relaxed">
						触感思考者 · 在流动中寻觅静止
					</p>
					<div class="flex items-center gap-1 mb-4">
						<MapPin style="width:11px;height:11px;color:var(--neu-accent-primary);" strokeWidth={1.5} />
						<span class="text-[var(--neu-text-muted)] text-[11px]">江苏·徐州</span>
					</div>

					<!-- Dual action -->
					<div class="flex w-full gap-2 mb-5">
						<button
							type="button"
							class="neu-hover-btn flex-1 flex items-center justify-center gap-1.5 py-2.5 rounded-[14px] outline-none bg-[var(--neu-accent-primary)] text-white text-[13px] font-medium"
							style="box-shadow:0 3px 10px var(--neu-glow);"
							onclick={() => void goto(ROUTES.meEdit)}
						>
							<Pencil style="width:13px;height:13px;" strokeWidth={2} />
							编辑资料
						</button>
						<button
							type="button"
							class="neu-hover-btn flex-1 flex items-center justify-center gap-1.5 py-2.5 rounded-[14px] outline-none text-[var(--neu-text)] text-[13px]"
							style="background:transparent;border:1px solid var(--neu-border-pressed);transition:background 0.18s ease;"
						>
							<Share2 style="width:13px;height:13px;" strokeWidth={1.75} />
							分享主页
						</button>
					</div>

					<!-- Stats — flat, integrated INTO the hero card -->
					<div class="grid grid-cols-3 w-full rounded-[14px] bg-[var(--neu-border)] overflow-hidden" style="gap:1px;box-shadow:0 0 0 1px var(--neu-border);">
						<button
							type="button"
							class="neu-hover-bg flex flex-col items-center justify-center py-2.5 outline-none"
							style="background:var(--neu-bg);"
						>
							<span class="text-[var(--neu-text)] text-xl font-semibold leading-none">48</span>
							<span class="text-[var(--neu-text-muted)] text-[10px] mt-1.5">帖子</span>
						</button>
						<button
							type="button"
							class="neu-hover-bg flex flex-col items-center justify-center py-2.5 outline-none"
							style="background:var(--neu-bg);"
							onclick={() => void goto(`${ROUTES.meSection('relationships')}?tab=followers`)}
						>
							<span class="text-[var(--neu-text)] text-xl font-semibold leading-none">312</span>
							<span class="text-[var(--neu-text-muted)] text-[10px] mt-1.5">关注者</span>
						</button>
						<button
							type="button"
							class="neu-hover-bg flex flex-col items-center justify-center py-2.5 outline-none"
							style="background:var(--neu-bg);"
						>
							<span class="text-[var(--neu-text)] text-xl font-semibold leading-none">27</span>
							<span class="text-[var(--neu-text-muted)] text-[10px] mt-1.5">足迹</span>
						</button>
					</div>
				</div>
			</div>
		</div>

		<!-- Tabs + content -->
		<div class="mt-6">
			<ProfileTabs activeTab={activeTab} setActiveTab={(t) => (activeTab = t)} />
			<div class="px-4 mt-3">
				<GalleryGrid posts={postsMap[activeTab]} {activeTab} />
			</div>
		</div>
	</div>

	<!-- Right rail — tablet landscape only -->
	{#if isTablet}
		<ContextRail className="hidden lg:block">
			<TabletRail>
				<StreakCard />
				<VisitorsCard />
				<MoodCheckinCard />
				<AchievementCard />
				<SuggestedFollowsCard />
				<VibeCard />
			</TabletRail>
		</ContextRail>
	{/if}
</div>
