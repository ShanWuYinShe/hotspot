<script lang="ts">
	import { goto } from '$app/navigation';
	import { Camera, ChevronRight, Heart, ImageIcon, MapPin, Pencil, Settings, UserCheck, UserPlus, Users } from '@lucide/svelte';
import ContentColumn from '@/components/layout/page/ContentColumn.svelte';
import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
import BorderedList from '@/components/data-display/BorderedList.svelte';
import InteractiveListRow from '@/components/navigation/InteractiveListRow.svelte';
	import { useAccountData } from '@/lib/profile-data.svelte';
	import { ROUTES, type ProfileSection } from '@/lib/routes';

	// 移动/平板个人主页（对齐 Figma Make ProfileCompact 契约）：
	// 资料页眉 + 身份卡（68px 头像 + 编辑按钮 + stats 横条）+ 个人数据 5 行链接（无 Tab/画廊）。

	const { avatar, nickname, stats, bio, handle, locationName } = useAccountData();

	const DATA_ITEMS: { section: ProfileSection; label: string; detail: string; icon: typeof ImageIcon }[] = [
		{ section: 'moments', label: '我的瞬间', detail: '查看已发布的记录', icon: ImageIcon },
		{ section: 'groups', label: '我的群组', detail: `${stats.groups_count} 个正在参与`, icon: Users },
		{ section: 'friends', label: '好友', detail: `${stats.friends_count} 位好友`, icon: Heart },
		{ section: 'following', label: '关注', detail: `${stats.following_count} 人`, icon: UserPlus },
		{ section: 'fans', label: '粉丝', detail: `${stats.followers_count} 人`, icon: UserCheck }
	];

	const statsList: { label: string; value: number; section: ProfileSection | null }[] = [
		{ label: '瞬间', value: stats.moments_count, section: 'moments' },
		{ label: '关注者', value: stats.followers_count, section: 'fans' },
		{ label: '群组', value: stats.groups_count, section: 'groups' }
	];
</script>

<StandardPageLayout contentClassName="pt-5 pb-8">
	{#snippet header()}
		<!-- 资料页眉（对齐 Figma Make：eyebrow + 设置入口） -->
			<div class="border-b border-border-default" style="background:var(--surface-primary);">
				<ContentColumn class="flex items-center justify-between py-3">
					<p class="text-xs font-medium uppercase tracking-[0.14em] text-accent-default">个人资料</p>
					<ActionIconButton icon={Settings} label="设置" onClick={() => void goto(ROUTES.settings)} variant="ghost" size={32} />
				</ContentColumn>
			</div>
	{/snippet}

	<!-- 身份卡（compact） -->
	<section class="rounded-panel border border-border-default surface-flat" aria-label="个人资料">
		<div class="p-5">
			<div class="flex items-start justify-between gap-5">
				<div class="flex min-w-0 items-center gap-4">
					<div class="relative shrink-0">
						<div class="overflow-hidden rounded-full border-2 border-surface-primary bg-surface-primary p-1">
							{#if avatar}
								<img src={avatar} alt={nickname || '用户'} class="rounded-full object-cover" style="width:68px;height:68px;" />
							{:else}
								<span class="flex items-center justify-center text-text-secondary" style="width:68px;height:68px;font-size:28px;">{(nickname || '?')[0]}</span>
							{/if}
						</div>
						<button
							type="button"
							aria-label="更换头像"
							onclick={() => void goto(ROUTES.meEdit)}
							class="hover-btn absolute -bottom-1 -right-1 flex items-center justify-center rounded-full outline-none bg-accent-default text-white"
							style="width:24px;height:24px;border:2px solid var(--surface-primary);"
						>
							<Camera style="width:13px;height:13px;" strokeWidth={2} />
						</button>
					</div>
					<div class="min-w-0">
						<h1 class="text-[20px] font-medium tracking-[-0.01em] text-text-primary">{nickname || '未设置昵称'}</h1>
						<p class="mt-0.5 text-xs text-text-secondary">{handle}</p>
						<p class="mt-2 flex items-center gap-1.5 text-xs text-text-secondary">
							<MapPin style="width:14px;height:14px;color:var(--accent-default);" strokeWidth={1.8} />
							{locationName || '未设置位置'}
						</p>
					</div>
				</div>
				<button
					type="button"
					onclick={() => void goto(ROUTES.meEdit)}
					class="hover-btn inline-flex shrink-0 items-center gap-1.5 border border-border-default bg-surface-primary px-3 py-2 text-xs font-medium text-text-primary outline-none transition-colors hover:bg-accent-soft"
				>
					<Pencil style="width:14px;height:14px;" strokeWidth={1.8} />
					编辑资料
				</button>
			</div>
			<p class="mt-5 max-w-107.5 text-sm leading-6 text-text-secondary">
				{bio || '这个人还没有填写简介。'}
			</p>
		</div>
		<div class="grid grid-cols-3 border-t border-border-default">
			{#each statsList as stat, i}
				<div class="border-l border-border-default px-4 py-3 first:border-l-0">
					<p class="text-base font-medium tabular-nums text-text-primary">{stat.value}</p>
					<p class="mt-0.5 text-xs text-text-secondary">{stat.label}</p>
				</div>
			{/each}
		</div>
	</section>

	<!-- 个人数据 5 行链接 -->
	<BorderedList class="mt-5" ariaLabel="个人数据">
		<div class="flex items-center justify-between bg-surface-primary px-5 py-3.5">
			<div>
				<p class="text-sm font-medium text-text-primary">个人数据</p>
				<p class="mt-0.5 text-xs text-text-secondary">在这里查看你的内容与关系</p>
			</div>
		</div>
		{#each DATA_ITEMS as item (item.section)}
			<InteractiveListRow
				onClick={() => void goto(ROUTES.meSection(item.section))}
				ariaLabel={item.label}
				class="flex items-center gap-3 px-5 py-3.5"
			>
				<span class="flex shrink-0 items-center justify-center" style="width:32px;height:32px;background:var(--color-accent-bg);color:var(--accent-default);">
					<item.icon style="width:16px;height:16px;" strokeWidth={1.8} />
				</span>
				<span class="min-w-0 flex-1">
					<span class="block text-sm text-text-primary">{item.label}</span>
					<span class="mt-0.5 block truncate text-xs text-text-secondary">{item.detail}</span>
				</span>
				<ChevronRight style="width:16px;height:16px;color:var(--text-secondary);shrink:0;" strokeWidth={1.7} />
			</InteractiveListRow>
		{/each}
	</BorderedList>
</StandardPageLayout>
