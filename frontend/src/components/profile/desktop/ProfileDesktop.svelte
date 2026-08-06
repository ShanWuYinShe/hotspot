<script lang="ts">
	import { goto } from '$app/navigation';
	import { Camera, MapPin, Pencil, ShieldCheck, AtSign, FileText, ChevronRight } from '@lucide/svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import SectionMoments from '@/components/profile/SectionMoments.svelte';
	import SectionGroups from '@/components/profile/SectionGroups.svelte';
	import SectionFriends from '@/components/profile/SectionFriends.svelte';
	import SectionFollowing from '@/components/profile/SectionFollowing.svelte';
	import SectionFans from '@/components/profile/SectionFans.svelte';
	import { useAccountData } from '@/lib/profile-data.svelte';
	import { ROUTES, ME_SECTION_PATHS, type ProfileSection } from '@/lib/routes';

	// 桌面个人主页（对齐 Figma Make ProfilePage 契约）：
	// 总览 = 资料页眉 + 双栏（公开个人资料卡 + 资料状态 aside）+ 资料详情 dl；
	// 分区模式 = PageHeader + Section 组件（moments/groups/friends/following/fans）。
	// 注意：relationships 聚合分区由 /me/relationships 独立渲染（RelationshipsPage），不走本组件。

	let { section }: { section: string | null } = $props();

	const { avatar, nickname, stats, bio, handle, locationName } = useAccountData();

	const SECTION_META: Record<string, { title: string }> = {
		moments: { title: '我的瞬间' },
		groups: { title: '我的群组' },
		friends: { title: '好友' },
		following: { title: '关注' },
		fans: { title: '粉丝' }
	};
	const meta = $derived(section && SECTION_META[section] ? SECTION_META[section] : null);

	const statsList = $derived(
		([
			{ label: '瞬间', value: stats.moments_count, section: 'moments' },
			{ label: '关注者', value: stats.followers_count, section: 'fans' },
			{ label: '群组', value: stats.groups_count, section: 'groups' }
		] as { label: string; value: number; section: ProfileSection | null }[])
	);

	const detailRows = $derived([
		{ icon: AtSign, label: '用户名', value: handle || '未设置' },
		{ icon: MapPin, label: '所在地', value: locationName || '未设置' },
		{ icon: FileText, label: '个人简介', value: bio || '未填写' }
	]);

	const filledCount = $derived(
		[nickname, avatar, bio, locationName, stats.groups_count > 0].filter(Boolean).length
	);
</script>

{#if meta}
	<!-- 分区模式：StandardPageLayout + sticky PageHeader；
	     Section 组件为纯内容（不自建滚动/gutter），gutter 由 ContentColumn 统一提供 -->
	<StandardPageLayout contentClassName="animate-fade-in pt-4 pb-6">
		{#snippet header()}
			<PageHeader title={meta.title} sticky onBack={() => void goto(ROUTES.me)} />
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
	<!-- 总览模式（对齐 Figma Make ProfileDesktop：单列资料页） -->
	<StandardPageLayout contentClassName="py-7 lg:py-8">
		<!-- 资料页眉 -->
		<header class="flex items-start justify-between gap-6 border-b border-border-default pb-5">
			<div>
				<p class="text-xs font-medium uppercase tracking-[0.14em] text-accent-default">个人资料</p>
				<h1 class="mt-2 text-[24px] font-medium tracking-[-0.015em] text-text-primary">管理你的公开信息</h1>
				<p class="mt-1.5 text-sm text-text-secondary">这些资料会展示给访问你主页的人。</p>
			</div>
			<button
				type="button"
				onclick={() => void goto(ROUTES.meEdit)}
				class="hover-btn inline-flex shrink-0 items-center gap-1.5 border border-border-default bg-surface-primary px-3.5 py-2.5 text-sm font-medium text-text-primary outline-none transition-colors hover:bg-accent-soft"
			>
				<Pencil class="size-4"  strokeWidth={1.8} />
				编辑资料
			</button>
		</header>

		<!-- 双栏：公开资料卡 + 资料状态 -->
		<div class="mt-6 grid gap-5 lg:grid-cols-[minmax(0,1.35fr)_minmax(280px,0.65fr)]">
			<section class="rounded-panel border border-border-default surface-flat" aria-label="公开个人资料">
				<div class="flex items-start gap-5 p-6">
					<div class="relative shrink-0">
						<div class="overflow-hidden rounded-full border-2 border-surface-primary bg-surface-primary p-1">
							{#if avatar}
								<img src={avatar} alt={nickname || '用户'} class="rounded-full object-cover size-20.5"  />
							{:else}
								<span class="flex items-center justify-center text-text-secondary size-20.5 text-[32px]" >{(nickname || '?')[0]}</span>
							{/if}
						</div>
						<button
							type="button"
							aria-label="更换头像"
							onclick={() => void goto(ROUTES.meEdit)}
							class="hover-btn absolute -bottom-1 -right-1 flex items-center justify-center rounded-full outline-none bg-accent-default text-white"
							style="width:28px;height:28px;border:2px solid var(--surface-primary);"
						>
							<Camera class="size-3.5"  strokeWidth={2} />
						</button>
					</div>
					<div class="min-w-0 pt-1">
						<h2 class="text-[22px] font-medium tracking-[-0.01em] text-text-primary">{nickname || '未设置昵称'}</h2>
						<p class="mt-1 text-sm text-text-secondary">{handle}</p>
						<p class="mt-4 max-w-115 text-sm leading-6 text-text-secondary">{bio || '这个人还没有填写简介。'}</p>
						<p class="mt-3 flex items-center gap-1.5 text-xs text-text-secondary">
							<MapPin class="size-3.5"  style="color:var(--accent-default)" strokeWidth={1.8} />
							{locationName || '未设置位置'}
						</p>
					</div>
				</div>
				<div class="grid grid-cols-3 border-t border-border-default">
					{#each statsList as stat}
						<button
							type="button"
							class="hover-bg border-l border-border-default px-5 py-4 text-left outline-none first:border-l-0"
							style="background:var(--surface-primary);"
							onclick={() => (stat.section ? void goto(ME_SECTION_PATHS[stat.section]) : undefined)}
						>
							<p class="text-lg font-medium tabular-nums text-text-primary">{stat.value}</p>
							<p class="mt-0.5 text-xs text-text-secondary">{stat.label}</p>
						</button>
					{/each}
				</div>
			</section>

			<!-- 资料状态 -->
			<aside class="rounded-panel border border-border-default surface-flat p-5" aria-label="资料状态">
				<div class="flex items-center justify-between">
					<h2 class="text-sm font-medium text-text-primary">资料状态</h2>
					<ShieldCheck class="size-4"  style="color:var(--accent-default)" strokeWidth={1.8} />
				</div>
				<p class="mt-2 text-xs leading-5 text-text-secondary">你的基础资料已完善，可随时在编辑页更新。</p>
				<div class="mt-5 h-1.5 overflow-hidden bg-border-default">
					<div class="h-full bg-accent-default" style="width:{filledCount * 20}%;"></div>
				</div>
				<p class="mt-2 text-xs text-text-secondary">{filledCount} 项基础资料已填写</p>
				<button
					type="button"
					onclick={() => void goto(ROUTES.settings)}
					class="hover-btn mt-5 flex w-full items-center justify-center gap-2 border border-border-default bg-surface-primary py-2.5 text-xs font-medium text-text-primary outline-none transition-colors hover:bg-accent-soft"
				>
					账号与隐私设置
					<ChevronRight class="size-3.5"  strokeWidth={1.8} />
				</button>
			</aside>
		</div>

		<!-- 资料详情 -->
		<section class="mt-5 rounded-panel border border-border-default surface-flat" aria-label="资料详情">
			<div class="border-b border-border-default px-6 py-4">
				<h2 class="text-sm font-medium text-text-primary">资料详情</h2>
				<p class="mt-1 text-xs text-text-secondary">以下内容将显示在你的个人主页。</p>
			</div>
			<dl class="grid md:grid-cols-2">
				{#each detailRows as row, index}
					<div
						class="flex items-start gap-3 px-6 py-4 {index > 1 ? 'border-t border-border-default' : ''} {index % 2 === 1 ? 'md:border-l md:border-border-default' : ''}"
					>
						<row.icon class="size-4 mt-0.5 shrink-0"  style="color:var(--accent-default)" strokeWidth={1.7} />
						<div class="min-w-0">
							<dt class="text-xs text-text-secondary">{row.label}</dt>
							<dd class="mt-1 text-sm text-text-primary">{row.value}</dd>
						</div>
					</div>
				{/each}
			</dl>
		</section>
	</StandardPageLayout>
{/if}
