<script lang="ts">
	import { goto } from '$app/navigation';
	import { MapPin, Pencil, ShieldCheck, Sparkles, UsersRound, Users, ChevronRight } from '@lucide/svelte';
	import { platform } from '@/stores/platform.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import MetricRow from '@/components/data-display/MetricRow.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import SectionMoments from '@/components/profile/SectionMoments.svelte';
	import SectionGroups from '@/components/profile/SectionGroups.svelte';
	import SectionFriends from '@/components/profile/SectionFriends.svelte';
	import SectionFollowing from '@/components/profile/SectionFollowing.svelte';
	import SectionFans from '@/components/profile/SectionFans.svelte';
	import { useAccountData } from '@/lib/profile-data.svelte';
	import { ROUTES, ME_SECTION_PATHS, type ProfileSection } from '@/lib/routes';

	// 个人主页（/me）——全设备使用：
	// 总览 = 个人总览（身份卡 + 居中统计 + 我的内容入口 + 资料状态 aside），
	// 定位为「内容与关系导航落地页」，与编辑页去重（bio/位置只出现一次）；
	// 分区模式 = PageHeader + Section 组件（moments/groups/friends/following/fans）。
	// 设备分派：mobile 总览走 ProfileMobile；tablet/desktop 总览与全部 me 分区页
	// （任意设备）走本组件。relationships 聚合分区由 /me/relationships 独立渲染
	// （RelationshipsPage），不走本组件。

	let { section }: { section: string | null } = $props();

	const { avatar, nickname, stats, bio, handle, locationName, loading, error, retry } = useAccountData();

	const SECTION_META: Record<string, { title: string }> = {
		moments: { title: '我的瞬间' },
		groups: { title: '我的群组' },
		friends: { title: '好友' },
		following: { title: '关注' },
		fans: { title: '粉丝' }
	};
	const meta = $derived(section && SECTION_META[section] ? SECTION_META[section] : null);
	const device = $derived(platform.platform);

	const metricItems = $derived([
		{ label: '瞬间', value: stats.moments_count },
		{ label: '关注者', value: stats.followers_count },
		{ label: '群组', value: stats.groups_count }
	]);

	const ENTRY_ITEMS: { section: ProfileSection; title: string; description: string; icon: typeof Users }[] = [
		{ section: 'moments', title: '我的瞬间', description: '查看已发布的记录', icon: Sparkles },
		{ section: 'groups', title: '我的群组', description: `${stats.groups_count} 个正在参与`, icon: UsersRound },
		{ section: 'relationships', title: '关系', description: `好友 ${stats.friends_count} · 关注 ${stats.following_count} · 粉丝 ${stats.followers_count}`, icon: Users }
	];

	const filledCount = $derived(
		[nickname, avatar, bio, locationName, stats.groups_count > 0].filter(Boolean).length
	);
</script>

{#if meta}
	<!-- 分区模式：StandardPageLayout + sticky PageHeader；
	     Section 组件为纯内容（不自建滚动/gutter），gutter 由 ContentColumn 统一提供 -->
	<StandardPageLayout contentClassName="animate-fade-in pt-4 pb-6">
		{#snippet header()}
			<PageHeader title={meta.title} sticky titleSize={device === 'desktop' ? 20 : device === 'tablet' ? 24 : 22} onBack={() => void goto(ROUTES.me)} />
		{/snippet}
		{#if section === 'moments'}
			<SectionMoments />
		{:else if section === 'groups'}
			<SectionGroups />
		{:else if section === 'friends'}
			<SectionFriends />
		{:else if section === 'following'}
			<SectionFollowing />
		{:else if section === 'fans'}
			<SectionFans />
		{/if}
	</StandardPageLayout>
{:else}
	<!-- 总览模式：个人总览（身份卡 + 统计 + 内容入口 + 资料状态）。
	     总览页是长内容呼吸型布局，顶/底 padding 略大于标准列表页（pt-6 pb-8），
	     属于样板 pt-4 pb-6 的可登记变体。 -->
	<StandardPageLayout contentClassName="animate-fade-in pt-6 pb-8">
		{#snippet header()}
			<PageHeader
				title="个人总览"
				subtitle="你的公开资料、内容与关系的入口。"
				titleSize={device === 'desktop' ? 20 : device === 'tablet' ? 24 : 22}
				contentClassName="pt-4 pb-0"
			>
				{#snippet trailing()}
					<button
						type="button"
						onclick={() => void goto(ROUTES.meEdit)}
						class="hover-btn inline-flex shrink-0 items-center gap-1.5 border border-border-default rounded-md bg-surface-primary px-3.5 py-2.5 text-sm font-medium text-text-primary focus-ring transition-colors hover:bg-accent-soft"
					>
						<Pencil class="size-4"  strokeWidth={1.75} />
						编辑资料
					</button>
				{/snippet}
			</PageHeader>
		{/snippet}

		{#if loading}
			<div class="flex flex-col items-center gap-3 py-20">
				<div class="spinner size-7" ></div>
				<p class="text-text-secondary text-sm">加载个人资料…</p>
			</div>
		{:else if error}
			<LoadError message={error} onRetry={retry} />
		{:else}
			<!-- 身份卡 + 居中统计（同一张卡，指标复用 MetricRow 原语） -->
			<section class="mt-6 overflow-hidden rounded-lg surface-flat" aria-label="公开个人资料">
				<div class="flex items-start gap-5 p-6">
					<Avatar src={avatar} name={nickname} size={80} ring class="shrink-0" />
					<div class="min-w-0 pt-1">
						<h2 class="text-2xl font-medium tracking-title text-text-primary">{nickname || '未设置昵称'}</h2>
						<p class="mt-1 text-sm text-text-secondary">{handle}</p>
						<p class="mt-4 max-w-115 text-sm leading-6 text-text-secondary">{bio || '这个人还没有填写简介。'}</p>
						<p class="mt-3 flex items-center gap-1.5 text-xs text-text-secondary">
							<MapPin class="size-3.5 text-accent-default"   strokeWidth={1.75} />
							{locationName || '未设置位置'}
						</p>
					</div>
				</div>
				<MetricRow items={metricItems} class="border-t border-border-default" />
			</section>

			<!-- 双栏：我的内容入口 + 资料状态 -->
			<div class="mt-5 grid gap-5 lg:grid-cols-[minmax(0,1.35fr)_minmax(280px,0.65fr)]">
				<section class="rounded-lg surface-flat" aria-label="我的内容">
					<div class="border-b border-border-default px-6 py-4">
						<h2 class="text-sm font-medium text-text-primary">我的内容</h2>
						<p class="mt-1 text-xs text-text-secondary">你的瞬间、群组与社交关系都在这里。</p>
					</div>
					<div class="p-3">
						{#each ENTRY_ITEMS as item (item.section)}
							<button
								type="button"
								onclick={() => void goto(ME_SECTION_PATHS[item.section])}
								class="hover-btn flex w-full items-center gap-3.5 rounded-md p-3 text-left focus-ring hover:bg-interactive-hover"
							>
								<span class="flex shrink-0 items-center justify-center rounded-full icon-badge size-9">
									<item.icon class="size-4.5 text-accent-default"  strokeWidth={1.75} />
								</span>
								<span class="min-w-0 flex-1">
									<span class="block text-sm font-medium text-text-primary">{item.title}</span>
									<span class="mt-0.5 block truncate text-xs text-text-secondary">{item.description}</span>
								</span>
								<ChevronRight class="size-4 shrink-0 text-text-secondary"  strokeWidth={1.75} />
							</button>
						{/each}
					</div>
				</section>

				<!-- 资料状态 -->
				<aside class="rounded-lg surface-flat p-5" aria-label="资料状态">
					<div class="flex items-center justify-between">
						<h2 class="text-sm font-medium text-text-primary">资料状态</h2>
						<ShieldCheck class="size-4 text-accent-default"   strokeWidth={1.75} />
					</div>
					<p class="mt-2 text-xs leading-5 text-text-secondary">你的基础资料已完善，可随时在编辑页更新。</p>
					<div class="mt-5 h-1.5 overflow-hidden rounded-full bg-border-default">
						<div class="h-full rounded-full bg-accent-default" style="width:{filledCount * 20}%;"></div>
					</div>
					<p class="mt-2 text-xs text-text-secondary">{filledCount} 项基础资料已填写</p>
					<button
						type="button"
						onclick={() => void goto(ROUTES.settings)}
						class="hover-btn mt-5 flex w-full items-center justify-center gap-2 border border-border-default rounded-md bg-surface-primary py-2.5 text-xs font-medium text-text-primary focus-ring transition-colors hover:bg-accent-soft"
					>
						账号与隐私设置
						<ChevronRight class="size-3.5"  strokeWidth={1.75} />
					</button>
				</aside>
			</div>
		{/if}
	</StandardPageLayout>
{/if}
