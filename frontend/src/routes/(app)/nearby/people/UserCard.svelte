<script lang="ts">
	import { MapPin, UserCheck, UserPlus } from '@lucide/svelte';
	import NearbyImageCard from '@/components/data-display/NearbyImageCard.svelte';

	// 附近用户卡片：基于 NearbyImageCard 统一图片优先原语。
	// 标签/距离/关注等菜单以覆盖层显示在图片上。
	// 桌面/平板 BSP 布局中高度由布局单元决定；移动网格中由 aspect-[3/4] 撑起。

	export interface NearbyUser {
		id: string;
		name: string;
		distance: string;
		avatar: string;
		status: string;
		tag: string;
		mutual: number;
		note: string;
	}

	let {
		user,
		followed,
		onFollow,
		onNavigate
	}: {
		user: NearbyUser;
		followed: boolean;
		onFollow: () => void;
		onNavigate: () => void;
	} = $props();

	// 头像缺失或加载失败时的占位状态；头像地址变化后重置
	let avatarFailed = $state(false);
	$effect(() => {
		void user.avatar;
		avatarFailed = false;
	});
</script>

<NearbyImageCard
	imageSrc={user.avatar}
	imageAlt={user.name}
	onClick={onNavigate}
	onImageError={() => (avatarFailed = true)}
	class="rounded-panel aspect-[3/4] neu-surface-flat"
>
	<!-- 无头像 / 加载失败：accent 主题渐变占位覆盖在图片层之上 -->
	{#if avatarFailed}
		<div
			class="absolute inset-0 z-[1] flex items-center justify-center"
			style="background:linear-gradient(135deg, color-mix(in srgb, var(--neu-accent-primary) 24%, var(--neu-bg)) 0%, color-mix(in srgb, var(--neu-accent-primary) 8%, var(--neu-bg)) 100%);"
		>
			<span class="text-[var(--neu-accent-primary)] text-3xl font-semibold">{Array.from(user.name)[0] ?? '?'}</span>
		</div>
	{/if}

	<!-- 右上：距离（单份，消除历史重复徽章 bug） -->
	<div class="absolute top-2.5 right-2.5 z-[2] flex items-center gap-1 px-1.5 py-0.5 rounded-full bg-[var(--overlay-strong)]" style="backdrop-filter:blur(8px);">
		<MapPin class="w-[9px] h-[9px] text-white" strokeWidth={1.5} />
		<span class="text-white text-[10px]">{user.distance}</span>
	</div>

	{#snippet footer()}
		<div class="px-3 pt-2.5 pb-2.5">
			<div class="flex items-center justify-between gap-2">
				<div class="min-w-0">
					<p class="flex items-center gap-1.5 truncate text-[var(--neu-text)] text-[14px]" style="font-weight:600;">
						<span class="truncate">{user.name}</span>
					</p>
					<p class="truncate text-[var(--neu-text-muted)] text-[10px] mt-0.5">{user.note}</p>
				</div>
				{#if user.status === '在线'}
					<span
						class="shrink-0 rounded-full"
						title="在线"
						style="width:9px;height:9px;background:var(--status-online);box-shadow:0 0 0 2px var(--surface-card-menu);"
					></span>
				{/if}
			</div>
			<div class="mt-2 flex items-center justify-between gap-2">
				{#if user.tag}
					<span class="rounded-full px-1.5 py-0.5 text-[10px]" style="background:var(--neu-accent-soft);color:var(--neu-accent-primary);border:1px solid var(--neu-accent-soft-border);">#{user.tag}</span>
				{:else}
					<span></span>
				{/if}
				<button
					type="button"
					class="neu-hover-btn neu-tap-90 flex items-center justify-center rounded-full outline-none flex-shrink-0"
					style="width:28px;height:28px;{followed
						? 'background:var(--neu-accent-soft);border:1px solid var(--neu-accent-soft-border);color:var(--neu-accent-primary);'
						: 'background:var(--neu-accent-primary);color:var(--text-on-accent);box-shadow:0 2px 8px var(--neu-glow);'}"
					aria-label={followed ? `取消关注 ${user.name}` : `关注 ${user.name}`}
					onclick={(e) => {
						e.stopPropagation();
						onFollow();
					}}
				>
					{#if followed}
						<UserCheck size={14} />
					{:else}
						<UserPlus size={14} />
					{/if}
				</button>
			</div>
		</div>
	{/snippet}
</NearbyImageCard>
