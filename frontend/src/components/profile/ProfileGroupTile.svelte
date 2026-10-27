<script lang="ts">
	import { goto } from '$app/navigation';
	import { ChevronRight, Clock, LogOut, MoreHorizontal, Shield, Users } from '@lucide/svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import * as DropdownMenu from '@/components/ui/dropdown-menu';
	import NearbyImageCard from '@/components/data-display/NearbyImageCard.svelte';
	import ConfirmDialog from '@/components/overlays/ConfirmDialog.svelte';
	import { leaveGroup } from '@/api';
	import { showToast } from '@/stores/toast.svelte';
	import { platform } from '@/stores/platform.svelte';
	import { ROUTES } from '@/lib/routes';
	import type { MappedGroup } from '@/lib/profile-data.svelte';

	// 我的群组卡（R13）：统一横排（图左 40% + 信息右）——「我的群组」是列表型内容，
	// 原先桌面 5 列图片网格在只有 2 个群时右侧留出 1/3 空场，横排列表既消除空岛也更好扫读。
	// 管理入口（R13 评审 P1-7 批准）：… → 退出群组（复用 leaveGroup + ConfirmDialog），
	// 成功后由父级从本地列表移除；桌面 hover 显现、触屏常显（desktop-reveal）。

	let {
		group,
		onLeft
	}: {
		group: MappedGroup;
		onLeft: () => void;
	} = $props();

	const isMobile = $derived(platform.platform === 'mobile');
	let moreOpen = $state(false);
	let menuAnchor = $state<HTMLElement | null>(null);
	let leaveOpen = $state(false);
	let leaving = $state(false);

	async function handleLeave() {
		if (leaving) return;
		leaving = true;
		try {
			const res = await leaveGroup(group.id);
			if (res.code === 0) {
				showToast('已退出群组', 'success');
				leaveOpen = false;
				onLeft();
			} else {
				showToast(res.message || '退出失败', 'danger');
			}
		} catch {
			showToast('退出失败，请重试', 'danger');
		} finally {
			leaving = false;
		}
	}
</script>

<div class="group relative">
	<NearbyImageCard
		imageSrc={group.banner || group.avatar}
		imageAlt={group.name}
		horizontal
		onClick={() => void goto(`${ROUTES.group(group.id)}?from=me`)}
		class="h-32 rounded-md sm:h-36"
	>
		{#snippet watermark()}
			<Users size={72} strokeWidth={1} />
		{/snippet}

		{#snippet placeholder()}
			<span class="text-accent-default text-3xl font-semibold">{Array.from(group.name)[0] ?? '群'}</span>
		{/snippet}

		<!-- 左上：角色身份徽章 -->
		<div
			class="absolute left-2.5 top-2.5 z-[2] inline-flex h-5 items-center justify-center gap-1 rounded-full bg-surface-overlay-strong px-2.5 text-xxs font-medium leading-none text-content-inverse"
			style="backdrop-filter:blur(var(--cst-blur-sm));"
		>
			{#if group.role === '管理员'}
				<Shield class="mr-0.5 inline size-2.5 text-accent-default" strokeWidth={2} />
			{/if}
			{group.role}
		</div>

		<!-- 右上：管理入口（desktop-reveal；展开时保持可见） -->
		<div class="desktop-reveal absolute right-2 top-2 z-20" style={moreOpen ? 'opacity:1' : undefined}>
			<DropdownMenu.Root bind:open={moreOpen}>
				<div bind:this={menuAnchor}>
					<ActionIconButton
						icon={MoreHorizontal}
						label="管理这个群组"
						variant="overlay"
						tone="inverse"
						size={32}
						onClick={() => (moreOpen = !moreOpen)}
					/>
				</div>
				<DropdownMenu.Content customAnchor={menuAnchor}>
					<DropdownMenu.Item
						class="text-status-danger hover:text-text-status-danger-on-soft data-[highlighted]:text-text-status-danger-on-soft hover:bg-danger-bg data-[highlighted]:bg-danger-bg"
						onSelect={() => {
							moreOpen = false;
							leaveOpen = true;
						}}
					>
						<LogOut class="size-3.5" strokeWidth={1.75} />
						退出群组
					</DropdownMenu.Item>
				</DropdownMenu.Content>
			</DropdownMenu.Root>
		</div>

		{#snippet footer({ horizontal }: { onImage: boolean; horizontal: boolean })}
			{@const onCover = false}
			<div class="p-4">
				<div class="flex items-center justify-between gap-2">
					<p class="truncate text-md font-semibold {onCover ? 'text-content-inverse' : 'text-text-primary'}">{group.name}</p>
					<ChevronRight class="size-3.5 shrink-0 {onCover ? 'text-content-inverse opacity-emphasis' : 'text-text-secondary'}" strokeWidth={1.75} />
				</div>
				{#if group.desc}
					<p class="mt-1 line-clamp-2 text-xs {onCover ? 'text-content-inverse opacity-emphasis' : 'text-text-secondary'}">{group.desc}</p>
				{/if}
				<div class="mt-2 flex items-center justify-between gap-2 text-xxs {onCover ? 'text-content-inverse opacity-emphasis' : 'text-text-secondary'}">
					<span class="flex items-center gap-1">
						<Users class="size-3" strokeWidth={1.5} />
						{group.members} 位成员
					</span>
					<span class="flex items-center gap-1">
						{#if group.online > 0}
							<span class="size-1.5 rounded-full bg-status-online"></span>
							{group.online} 在线
						{:else if group.lastActive}
							<Clock class="size-3" strokeWidth={1.5} />
							{group.lastActive}
						{/if}
					</span>
				</div>
			</div>
		{/snippet}
	</NearbyImageCard>

	<ConfirmDialog
		open={leaveOpen}
		tone="danger"
		icon={LogOut}
		title={`退出「${group.name}」？`}
		description="退出后需要重新申请才能加入，群内消息将不再接收。"
		confirmLabel="退出群组"
		layout={isMobile ? 'stacked' : 'row'}
		loading={leaving}
		onCancel={() => (leaveOpen = false)}
		onConfirm={() => void handleLeave()}
	/>
</div>