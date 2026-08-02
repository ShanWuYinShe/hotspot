<script lang="ts">
	import { goto } from '$app/navigation';
	import { Camera, ChevronRight, Heart, ImageIcon, MapPin, Pencil, Settings, UserCheck, UserPlus, Users } from '@lucide/svelte';
	import AppearanceMenu from '@/components/layout/AppearanceMenu.svelte';
	import ContentColumn from '@/components/layout/page/ContentColumn.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import { useAccountData } from '@/lib/profile-data.svelte';
	import { ROUTES, type ProfileSection } from '@/lib/routes';

	// 移动/平板个人主页（对齐 Figma Make ProfileCompact 契约）：
	// 资料页眉 + 身份卡（68px 头像 + 编辑按钮 + stats 横条）+ 个人数据 5 行链接（无 Tab/画廊）。

	const { avatar, nickname, stats } = useAccountData();

	const DATA_ITEMS: { section: ProfileSection; label: string; detail: string; icon: typeof ImageIcon }[] = [
		{ section: 'moments', label: '我的瞬间', detail: '查看已发布的记录', icon: ImageIcon },
		{ section: 'groups', label: '我的群组', detail: `${stats.groups_count} 个正在参与`, icon: Users },
		{ section: 'friends', label: '好友', detail: `${stats.friends_count} 位好友`, icon: Heart },
		{ section: 'following', label: '关注', detail: `${stats.following_count} 人`, icon: UserPlus },
		{ section: 'fans', label: '粉丝', detail: `${stats.followers_count} 人`, icon: UserCheck }
	];

	const statsList = [
		{ label: '帖子', value: stats.moments_count, section: 'moments' as const },
		{ label: '关注者', value: stats.followers_count, section: 'fans' as const },
		{ label: '足迹', value: 27, section: null as const }
	];
</script>

<StandardPageLayout contentClassName="pt-5 pb-8">
	{#snippet header()}
		<!-- 资料页眉（对齐 Figma Make：eyebrow + 设置入口） -->
		<div class="border-b border-[var(--neu-border)]" style="background:var(--neu-bg);">
			<ContentColumn class="flex items-center justify-between py-3">
				<p class="text-[11px] font-medium uppercase tracking-[0.14em] text-[var(--neu-accent-primary)]">个人资料</p>
				<div class="flex items-center gap-1.5">
					<AppearanceMenu />
					<ActionIconButton icon={Settings} label="设置" onClick={() => void goto(ROUTES.settings)} size={34} />
				</div>
			</ContentColumn>
		</div>
	{/snippet}

	<!-- 身份卡（compact） -->
	<section class="rounded-panel border border-[var(--neu-border)] neu-surface-flat" aria-label="个人资料">
		<div class="p-5">
			<div class="flex items-start justify-between gap-5">
				<div class="flex min-w-0 items-center gap-4">
					<div class="relative shrink-0">
						<div class="overflow-hidden rounded-full border-2 border-[var(--neu-bg)] bg-[var(--neu-bg)] p-1">
							{#if avatar}
								<img src={avatar} alt={nickname || '用户'} class="rounded-full object-cover" style="width:68px;height:68px;" />
							{:else}
								<span class="flex items-center justify-center text-[var(--neu-text-muted)]" style="width:68px;height:68px;font-size:28px;">{(nickname || '?')[0]}</span>
							{/if}
						</div>
						<button
							type="button"
							aria-label="更换头像"
							onclick={() => void goto(ROUTES.meEdit)}
							class="neu-hover-btn absolute -bottom-1 -right-1 flex items-center justify-center rounded-full outline-none bg-[var(--neu-accent-primary)] text-white"
							style="width:26px;height:26px;border:2px solid var(--neu-bg);"
						>
							<Camera style="width:13px;height:13px;" strokeWidth={2} />
						</button>
					</div>
					<div class="min-w-0">
						<h1 class="text-[20px] font-medium tracking-[-0.01em] text-[var(--neu-text)]">{nickname || '鹿鸣'}</h1>
						<p class="mt-0.5 text-xs text-[var(--neu-text-muted)]">@elara.v</p>
						<p class="mt-2 flex items-center gap-1.5 text-xs text-[var(--neu-text-muted)]">
							<MapPin style="width:14px;height:14px;color:var(--neu-accent-primary);" strokeWidth={1.8} />
							江苏 · 徐州
						</p>
					</div>
				</div>
				<button
					type="button"
					onclick={() => void goto(ROUTES.meEdit)}
					class="neu-hover-btn inline-flex shrink-0 items-center gap-1.5 border border-[var(--neu-border)] bg-[var(--neu-bg)] px-3 py-2 text-xs font-medium text-[var(--neu-text)] outline-none transition-colors hover:bg-[var(--neu-accent-soft)]"
				>
					<Pencil style="width:14px;height:14px;" strokeWidth={1.8} />
					编辑资料
				</button>
			</div>
			<p class="mt-5 max-w-[430px] text-sm leading-6 text-[var(--neu-text-muted)]">
				触感思考者。在流动里收集安静，也在熟悉的街道寻找新的光。
			</p>
		</div>
		<div class="grid grid-cols-3 border-t border-[var(--neu-border)]">
			{#each statsList as stat, i}
				<button
					type="button"
					class="neu-hover-bg border-l border-[var(--neu-border)] px-4 py-3 text-left outline-none first:border-l-0"
					style="background:var(--neu-bg);"
					onclick={() => (stat.section ? void goto(ROUTES.meSection(stat.section)) : undefined)}
				>
					<p class="text-base font-medium tabular-nums text-[var(--neu-text)]">{stat.value}</p>
					<p class="mt-0.5 text-[11px] text-[var(--neu-text-muted)]">{stat.label}</p>
				</button>
			{/each}
		</div>
	</section>

	<!-- 个人数据 5 行链接 -->
	<section class="mt-5 rounded-panel border border-[var(--neu-border)] neu-surface-flat" aria-label="个人数据">
		<div class="flex items-center justify-between border-b border-[var(--neu-border)] px-5 py-3.5">
			<div>
				<p class="text-sm font-medium text-[var(--neu-text)]">个人数据</p>
				<p class="mt-0.5 text-[11px] text-[var(--neu-text-muted)]">管理你在 HotSpot 的内容与关系</p>
			</div>
		</div>
		{#each DATA_ITEMS as item (item.section)}
			<button
				type="button"
				class="neu-hover-bg flex w-full items-center gap-3 border-b border-[var(--neu-border)] px-5 py-3.5 text-left outline-none last:border-b-0"
				style="background:var(--neu-bg);"
				onclick={() => void goto(ROUTES.meSection(item.section))}
			>
				<item.icon style="width:17px;height:17px;color:var(--neu-accent-primary);flex-shrink:0;" strokeWidth={1.7} />
				<span class="min-w-0 flex-1">
					<span class="block text-sm text-[var(--neu-text)]">{item.label}</span>
					<span class="mt-0.5 block truncate text-[11px] text-[var(--neu-text-muted)]">{item.detail}</span>
				</span>
				<ChevronRight style="width:16px;height:16px;color:var(--neu-text-muted);flex-shrink:0;" strokeWidth={1.7} />
			</button>
		{/each}
	</section>
</StandardPageLayout>
