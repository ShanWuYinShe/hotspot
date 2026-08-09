<script lang="ts">
	import ListItem from '@/components/list/ListItem.svelte';
	import { TYPE_CONFIG, type NotificationItem } from './notification-data';

	// 通知行（对齐 Figma Make NotificationsPage 行契约：ListItem
	// hover 背景填充 + tap opacity 0.84，无 scale 位移）。

	let {
		notif,
		onRead,
		onRemove
	}: {
		notif: NotificationItem;
		onRead: (id: string) => void;
		onRemove: (id: string) => void;
	} = $props();

	const cfg = $derived(TYPE_CONFIG[notif.type] ?? TYPE_CONFIG.reaction);
</script>

<ListItem
	onClick={() => onRead(notif.id)}
	ariaLabel={`通知：${notif.text}`}
	class="flex items-center gap-3.5 px-3 py-3.5"
>
	<!-- Avatar + icon badge -->
	<div class="relative shrink-0">
		{#if notif.user.avatar}
			<div class="rounded-full overflow-hidden size-11" >
		<img src={notif.user.avatar} alt={notif.user.name} loading="lazy" class="w-full h-full object-cover" />
			</div>
		{:else}
			<div class="rounded-full flex items-center justify-center size-11" style="background:{cfg.bg};">
				<cfg.icon class="size-4.5" style="color:{cfg.color};" strokeWidth={1.75} />
			</div>
		{/if}
		{#if notif.user.avatar}
			<div class="absolute -bottom-0.5 -right-0.5 flex items-center justify-center rounded-full size-4.5" style="background:{cfg.color};border:2px solid var(--cst-surface-primary);">
				<cfg.icon class="size-2.5 text-text-on-accent"   strokeWidth={2.5} />
			</div>
		{/if}
	</div>

	<!-- Text -->
	<div class="flex-1 min-w-0">
		<p class="truncate text-text-primary text-sm">
			<span class="{notif.read ? 'font-normal' : 'font-medium'}">{notif.user.name}</span>
			{' '}{notif.text}
		</p>
		<p class="text-text-secondary text-xs mt-0.5">{notif.time}</p>
	</div>

	<!-- Moment thumb or unread dot -->
	<div class="flex items-center gap-2 shrink-0">
		{#if notif.momentImage}
			<div class="rounded-sm overflow-hidden size-9.5" >
		<img src={notif.momentImage} alt="" loading="lazy" class="w-full h-full object-cover" />
			</div>
		{/if}
		{#if !notif.read}
			<div class="rounded-full size-2" style="background:{cfg.color};"></div>
		{/if}
	</div>
</ListItem>
