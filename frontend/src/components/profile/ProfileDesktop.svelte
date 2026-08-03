<script lang="ts">
	import { goto } from '$app/navigation';
	import { Camera, MapPin, Pencil, ShieldCheck, AtSign, FileText, UserCheck, ChevronRight } from '@lucide/svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import ContentColumn from '@/components/layout/page/ContentColumn.svelte';
	import SectionMoments from '@/components/profile/SectionMoments.svelte';
	import SectionGroups from '@/components/profile/SectionGroups.svelte';
	import SectionFriends from '@/components/profile/SectionFriends.svelte';
	import SectionFollowing from '@/components/profile/SectionFollowing.svelte';
	import SectionFans from '@/components/profile/SectionFans.svelte';
	import { useAccountData } from '@/lib/profile-data.svelte';
	import { ROUTES, type ProfileSection } from '@/lib/routes';

	// 桌面个人主页（对齐 Figma Make ProfilePage 契约）：
	// 总览 = 资料页眉 + 双栏（公开个人资料卡 + 资料状态 aside）+ 资料详情 dl；
	// 分区模式 = PageHeader + Section 组件（moments/groups/friends/following/fans）。

	let { section }: { section: string | null } = $props();

	const { avatar, nickname, stats } = useAccountData();

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
			{ label: '帖子', value: stats.moments_count, section: 'moments' },
			{ label: '关注者', value: stats.followers_count, section: 'fans' },
			{ label: '足迹', value: 27, section: null }
		] as { label: string; value: number; section: ProfileSection | null }[])
	);

	const detailRows = $derived([
		{ icon: AtSign, label: '用户名', value: '@elara.v' },
		{ icon: MapPin, label: '所在地', value: '江苏 · 徐州' },
		{ icon: FileText, label: '个人简介', value: '触感思考者 · 在流动中寻觅静止' },
		{ icon: UserCheck, label: '资料可见范围', value: '公开' }
	]);
</script>

{#if meta}
	<!-- 分区模式 -->
	<div class="flex flex-col h-full">
		<PageHeader title={meta.title} sticky onBack={() => void goto(ROUTES.me)} />
		<div class="flex-1 min-h-0 overflow-hidden flex">
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
		</div>
	</div>
{:else}
	<!-- 总览模式（对齐 Figma Make ProfileDesktop：单列资料页） -->
	<ContentColumn class="py-7 lg:py-8">
		<!-- 资料页眉 -->
		<header class="flex items-start justify-between gap-6 border-b border-[var(--neu-border)] pb-5">
			<div>
				<p class="text-[11px] font-medium uppercase tracking-[0.14em] text-[var(--neu-accent-primary)]">个人资料</p>
				<h1 class="mt-2 text-[24px] font-medium tracking-[-0.015em] text-[var(--neu-text)]">管理你的公开信息</h1>
				<p class="mt-1.5 text-sm text-[var(--neu-text-muted)]">这些资料会展示给访问你主页的人。</p>
			</div>
			<button
				type="button"
				onclick={() => void goto(ROUTES.meEdit)}
				class="neu-hover-btn inline-flex shrink-0 items-center gap-1.5 border border-[var(--neu-border)] bg-[var(--neu-bg)] px-3.5 py-2.5 text-sm font-medium text-[var(--neu-text)] outline-none transition-colors hover:bg-[var(--neu-accent-soft)]"
			>
				<Pencil style="width:16px;height:16px;" strokeWidth={1.8} />
				编辑资料
			</button>
		</header>

		<!-- 双栏：公开资料卡 + 资料状态 -->
		<div class="mt-6 grid gap-5 lg:grid-cols-[minmax(0,1.35fr)_minmax(280px,0.65fr)]">
			<section class="rounded-panel border border-[var(--neu-border)] neu-surface-flat" aria-label="公开个人资料">
				<div class="flex items-start gap-5 p-6">
					<div class="relative shrink-0">
						<div class="overflow-hidden rounded-full border-2 border-[var(--neu-bg)] bg-[var(--neu-bg)] p-1">
							{#if avatar}
								<img src={avatar} alt={nickname || '用户'} class="rounded-full object-cover" style="width:82px;height:82px;" />
							{:else}
								<span class="flex items-center justify-center text-[var(--neu-text-muted)]" style="width:82px;height:82px;font-size:32px;">{(nickname || '?')[0]}</span>
							{/if}
						</div>
						<button
							type="button"
							aria-label="更换头像"
							onclick={() => void goto(ROUTES.meEdit)}
							class="neu-hover-btn absolute -bottom-1 -right-1 flex items-center justify-center rounded-full outline-none bg-[var(--neu-accent-primary)] text-white"
							style="width:28px;height:28px;border:2px solid var(--neu-bg);"
						>
							<Camera style="width:14px;height:14px;" strokeWidth={2} />
						</button>
					</div>
					<div class="min-w-0 pt-1">
						<h2 class="text-[22px] font-medium tracking-[-0.01em] text-[var(--neu-text)]">{nickname || '鹿鸣'}</h2>
						<p class="mt-1 text-sm text-[var(--neu-text-muted)]">@elara.v</p>
						<p class="mt-4 max-w-[460px] text-sm leading-6 text-[var(--neu-text-muted)]">触感思考者。在流动里收集安静，也在熟悉的街道寻找新的光。</p>
						<p class="mt-3 flex items-center gap-1.5 text-xs text-[var(--neu-text-muted)]">
							<MapPin style="width:14px;height:14px;color:var(--neu-accent-primary);" strokeWidth={1.8} />
							江苏 · 徐州
						</p>
					</div>
				</div>
				<div class="grid grid-cols-3 border-t border-[var(--neu-border)]">
					{#each statsList as stat, i}
						<button
							type="button"
							class="neu-hover-bg border-l border-[var(--neu-border)] px-5 py-4 text-left outline-none first:border-l-0"
							style="background:var(--neu-bg);"
							onclick={() => (stat.section ? void goto(ROUTES.meSection(stat.section)) : undefined)}
						>
							<p class="text-lg font-medium tabular-nums text-[var(--neu-text)]">{stat.value}</p>
							<p class="mt-0.5 text-xs text-[var(--neu-text-muted)]">{stat.label}</p>
						</button>
					{/each}
				</div>
			</section>

			<!-- 资料状态 -->
			<aside class="rounded-panel border border-[var(--neu-border)] neu-surface-flat p-5" aria-label="资料状态">
				<div class="flex items-center justify-between">
					<h2 class="text-sm font-medium text-[var(--neu-text)]">资料状态</h2>
					<ShieldCheck style="width:16px;height:16px;color:var(--neu-accent-primary);" strokeWidth={1.8} />
				</div>
				<p class="mt-2 text-xs leading-5 text-[var(--neu-text-muted)]">你的基础资料已完善，可随时在编辑页更新。</p>
				<div class="mt-5 h-1.5 overflow-hidden bg-[var(--neu-border)]">
					<div class="h-full w-full bg-[var(--neu-accent-primary)]"></div>
				</div>
				<p class="mt-2 text-[11px] text-[var(--neu-text-muted)]">4 项基础资料已填写</p>
				<button
					type="button"
					onclick={() => void goto(ROUTES.settings)}
					class="neu-hover-btn mt-5 flex w-full items-center justify-center gap-2 border border-[var(--neu-border)] bg-[var(--neu-bg)] py-2.5 text-xs font-medium text-[var(--neu-text)] outline-none transition-colors hover:bg-[var(--neu-accent-soft)]"
				>
					账号与隐私设置
					<ChevronRight style="width:14px;height:14px;" strokeWidth={1.8} />
				</button>
			</aside>
		</div>

		<!-- 资料详情 -->
		<section class="mt-5 rounded-panel border border-[var(--neu-border)] neu-surface-flat" aria-label="资料详情">
			<div class="border-b border-[var(--neu-border)] px-6 py-4">
				<h2 class="text-sm font-medium text-[var(--neu-text)]">资料详情</h2>
				<p class="mt-1 text-xs text-[var(--neu-text-muted)]">以下内容将显示在你的个人主页。</p>
			</div>
			<dl class="grid md:grid-cols-2">
				{#each detailRows as row, index}
					<div
						class="flex items-start gap-3 px-6 py-4 {index > 1 ? 'border-t border-[var(--neu-border)]' : ''} {index % 2 === 1 ? 'md:border-l md:border-[var(--neu-border)]' : ''}"
					>
						<row.icon style="width:16px;height:16px;margin-top:2px;color:var(--neu-accent-primary);flex-shrink:0;" strokeWidth={1.7} />
						<div class="min-w-0">
							<dt class="text-[11px] text-[var(--neu-text-muted)]">{row.label}</dt>
							<dd class="mt-1 text-sm text-[var(--neu-text)]">{row.value}</dd>
						</div>
					</div>
				{/each}
			</dl>
		</section>
	</ContentColumn>
{/if}
