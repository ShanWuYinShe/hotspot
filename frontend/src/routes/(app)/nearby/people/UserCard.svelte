<script lang="ts">
	import { MapPin, UserCheck, UserPlus } from '@lucide/svelte';
	import OnlineDot from '@/components/shared/OnlineDot.svelte';

	// 附近用户卡片：图片铺满整卡，标签/距离/关注等菜单以覆盖层显示在图片上。
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

<div
	class="neu-hover-card relative h-full w-full overflow-hidden rounded-[20px] aspect-[3/4] cursor-pointer neu-surface-flat neu-card-hover neu-focus-ring"
	style="background:var(--neu-bg);"
	role="button"
	tabindex="0"
	onclick={onNavigate}
	onkeydown={(e) => {
		if (e.target !== e.currentTarget) return;
		if (e.key === 'Enter' || e.key === ' ') {
			e.preventDefault();
			onNavigate();
		}
	}}
>
	{#if user.avatar && !avatarFailed}
		<img
			src={user.avatar}
			alt={user.name}
			class="absolute inset-0 h-full w-full object-cover"
			loading="lazy"
			onerror={() => (avatarFailed = true)}
		/>
	{:else}
		<!-- 无头像 / 加载失败：accent 主题渐变占位铺满整卡 -->
		<div
			class="absolute inset-0 flex items-center justify-center"
			style="background:linear-gradient(135deg, color-mix(in srgb, var(--neu-accent-primary) 24%, var(--neu-bg)) 0%, color-mix(in srgb, var(--neu-accent-primary) 8%, var(--neu-bg)) 100%);"
		>
			<span class="text-[var(--neu-accent-primary)] text-3xl font-semibold">{Array.from(user.name)[0] ?? '?'}</span>
		</div>
	{/if}

	<!-- 左上：兴趣标签 -->
	{#if user.tag}
		<span
			class="absolute top-2.5 left-2.5 px-2 py-0.5 rounded-full text-white text-[10px]"
			style="background:var(--overlay-strong);backdrop-filter:blur(8px);"
		>
			#{user.tag}
		</span>
	{/if}

	<!-- 右上：距离 -->
	<div class="absolute top-2.5 right-2.5 flex items-center gap-1 px-1.5 py-0.5 rounded-full bg-[var(--overlay-strong)]" style="backdrop-filter:blur(8px);">
		<MapPin class="w-[9px] h-[9px] text-white" strokeWidth={1.5} />
		<span class="text-white text-[10px]">{user.distance}</span>
	</div>

	<!-- 底部：名字 / 此刻状态 / 共同好友 / 关注按钮（半透明深色玻璃背景，有图无图都保证可读） -->
	<div class="absolute inset-x-0 bottom-0 bg-[var(--card-scrim)] px-3 pt-2 pb-2.5" style="backdrop-filter:blur(12px);">
		<div class="min-w-0">
			<p class="flex items-center gap-1.5 truncate text-white text-sm font-semibold">
				<span class="truncate">{user.name}</span>
				{#if user.status === '在线'}
					<OnlineDot size="sm" className="relative static" color="var(--online-on-scrim)" />
				{/if}
			</p>
			<p class="truncate text-white/78 text-[10px] mt-0.5">{user.note}</p>
		</div>
		<div class="mt-2 flex items-center justify-between gap-2">
			<span class="truncate text-white/70 text-[10px]">{user.mutual ? `${user.mutual} 位共同好友` : '等待认识'}</span>
			<button
				type="button"
				class="neu-hover-btn flex items-center justify-center rounded-full outline-none flex-shrink-0 w-[30px] h-[30px]"
				aria-label={followed ? `取消关注 ${user.name}` : `关注 ${user.name}`}
				style={followed
					? 'background:var(--neu-accent-soft);border:1px solid rgba(255,255,255,0.45);color:white;'
					: 'background:var(--neu-accent-primary);color:white;box-shadow:0 2px 8px var(--neu-glow);'}
				onclick={(e) => {
					e.stopPropagation();
					onFollow();
				}}
			>
				{#if followed}
					<UserCheck size={15} />
				{:else}
					<UserPlus size={15} />
				{/if}
			</button>
		</div>
	</div>
</div>
