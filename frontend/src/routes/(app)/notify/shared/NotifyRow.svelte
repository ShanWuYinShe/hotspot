<script lang="ts">
	import { goto } from '$app/navigation';
	import ListItem from '@/components/list/ListItem.svelte';
	import { TYPE_CONFIG, type NotifyViewItem } from './notify-data';

	// 通知行（对齐 Figma Make NotifyPage 行契约：ListItem
	// hover 背景填充 + tap opacity 0.84，无 scale 位移）。

	let {
		notif,
		actionPending = false,
		onRead,
		onRemove,
		onAccept,
		onReject
	}: {
		notif: NotifyViewItem;
		/** 邀请接受/拒绝请求进行中（按钮禁点防重复提交） */
		actionPending?: boolean;
		onRead: (id: string) => void;
		onRemove: (id: string) => void;
		/** 入群邀请通知的可选操作（inviteId 存在时渲染接受/拒绝按钮）。 */
		onAccept?: (inviteId: string) => void;
		onReject?: (inviteId: string) => void;
	} = $props();

	const cfg = $derived(TYPE_CONFIG[notif.type] ?? TYPE_CONFIG.reaction);
</script>

<ListItem
	onClick={() => {
		onRead(notif.id);
		// 点击跳转（payload 上下文推导的 href；行内接受/拒绝按钮已 stopPropagation）
		if (notif.href) void goto(notif.href);
	}}
	ariaLabel={`通知：${notif.user.name}${notif.text}`}
	class="flex flex-wrap items-center gap-x-3.5 gap-y-2 px-4 py-3.5 transition-colors {notif.read ? '' : 'bg-accent-soft/15'}"
>
	<!-- Avatar + icon badge -->
	<div class="relative shrink-0">
		{#if notif.user.avatar}
			<div class="rounded-full overflow-hidden size-11 ring-1 ring-border-subtle/50">
				<img src={notif.user.avatar} alt={notif.user.name} loading="lazy" class="w-full h-full object-cover" />
			</div>
		{:else}
			<div class="rounded-full flex items-center justify-center size-11" style="background:{cfg.bg};">
				<cfg.icon class="size-4.5" style="color:{cfg.color};" strokeWidth={1.75} />
			</div>
		{/if}
		{#if notif.user.avatar}
			<div class="absolute -bottom-0.5 -right-0.5 flex items-center justify-center rounded-full size-4.5 shadow-xs" style="background:{cfg.color};border:var(--cst-border-width-strong) solid var(--cst-surface-primary);">
				<cfg.icon class="size-2.5 text-text-on-accent" strokeWidth={2.5} />
			</div>
		{/if}
	</div>

	<!-- Text -->
	<div class="flex-1 min-w-0">
		<p class="truncate text-text-primary text-sm">
			<span class="{notif.read ? 'font-normal text-text-secondary' : 'font-semibold text-text-primary'}">{notif.user.name}</span>
			{' '}{notif.text}
		</p>
		<p class="text-text-secondary text-xs mt-0.5">{notif.time}</p>
	</div>

	<!-- Moment thumb or unread dot；邀请行窄屏时动作组换行到第二行，
	     避免把正文挤成省略号、时间被拆行 -->
	<div class="flex items-center gap-2.5 shrink-0 {notif.inviteId ? 'w-full justify-end sm:w-auto' : ''}">
		{#if notif.inviteId}
			<!-- 入群邀请操作：行内接受/拒绝（stopPropagation 避免触发行已读） -->
			<button
				type="button"
				onclick={(e) => { e.stopPropagation(); onAccept?.(notif.inviteId!); }}
				disabled={actionPending}
				class="hover-btn hit-slop inline-flex h-8 items-center rounded-full bg-accent-default px-3.5 text-xs font-medium text-text-on-accent shadow-raised-sm focus-ring hover:shadow-raised-md disabled:pointer-events-none disabled:opacity-faint active:scale-95 transition-transform"
			>接受</button>
			<button
				type="button"
				onclick={(e) => { e.stopPropagation(); onReject?.(notif.inviteId!); }}
				disabled={actionPending}
				class="hover-btn hit-slop inline-flex h-8 items-center rounded-full bg-surface-subtle/60 px-3.5 text-xs text-text-primary focus-ring hover:bg-surface-subtle disabled:pointer-events-none disabled:opacity-faint active:scale-95 transition-transform"
			>拒绝</button>
		{/if}
		{#if notif.momentImage}
			<div class="rounded-lg overflow-hidden size-10 border border-border-subtle/50 shadow-xs">
				<img src={notif.momentImage} alt="" loading="lazy" class="w-full h-full object-cover" />
			</div>
		{/if}
		{#if !notif.read}
			<!-- 未读指示：实心微蓝呼吸光点，高质感辨识 -->
			<div
				class="rounded-full size-2 bg-accent-default shadow-[0_0_6px_var(--cst-accent-default)] animate-pulse"
				aria-hidden="true"
			></div>
			<span class="sr-only">未读</span>
		{/if}
	</div>
</ListItem>
