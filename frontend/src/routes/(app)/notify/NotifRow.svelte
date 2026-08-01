<script lang="ts">
	import { TYPE_CONFIG, type Notif } from './notif-data';

	// 通知行（迁移自旧版 NotificationsPage 内 NotifRow）。

	let {
		notif,
		onRead,
		onRemove
	}: {
		notif: Notif;
		onRead: (id: string) => void;
		onRemove: (id: string) => void;
	} = $props();

	const cfg = $derived(TYPE_CONFIG[notif.type] ?? TYPE_CONFIG.reaction);
	const cfgTone = (amount: number) => `color-mix(in srgb, ${cfg.color} ${amount}%, transparent)`;
</script>

<button
	type="button"
	class="neu-hover-row app-list-row app-list-row--rich cursor-pointer rounded-[18px] mb-1.5 px-3 w-full text-left neu-row-hover"
	style="background:{notif.read ? 'transparent' : cfgTone(7)};border:{notif.read ? '1px solid transparent' : `1px solid ${cfgTone(18)}`};"
	onclick={() => onRead(notif.id)}
>
	<!-- Avatar + icon badge -->
	<div class="relative flex-shrink-0">
		{#if notif.user.avatar}
			<div class="rounded-full overflow-hidden" style="width:44px;height:44px;">
				<img src={notif.user.avatar} alt={notif.user.name} class="w-full h-full object-cover" />
			</div>
		{:else}
			<div class="rounded-full flex items-center justify-center" style="width:44px;height:44px;background:{cfgTone(18)};">
				<cfg.icon style="width:18px;height:18px;color:{cfg.color};" strokeWidth={1.75} />
			</div>
		{/if}
		{#if notif.user.avatar}
			<div class="absolute -bottom-0.5 -right-0.5 flex items-center justify-center rounded-full" style="width:18px;height:18px;background:{cfg.color};border:2px solid var(--neu-bg);">
				<cfg.icon style="width:9px;height:9px;color:white;" strokeWidth={2.5} />
			</div>
		{/if}
	</div>

	<!-- Text -->
	<div class="flex-1 min-w-0">
		<p class="truncate text-[var(--neu-text)] text-[13px]">
			<span style="font-weight:{notif.read ? 400 : 500};">{notif.user.name}</span>
			{' '}{notif.text}
		</p>
		<p class="text-[var(--neu-text-muted)] text-[11px] mt-0.5">{notif.time}</p>
	</div>

	<!-- Post thumb or unread dot -->
	<div class="flex items-center gap-2 flex-shrink-0">
		{#if notif.postImage}
			<div class="rounded-[10px] overflow-hidden" style="width:38px;height:38px;">
				<img src={notif.postImage} alt="" class="w-full h-full object-cover" />
			</div>
		{/if}
		{#if !notif.read}
			<div class="rounded-full" style="width:8px;height:8px;background:{cfg.color};"></div>
		{/if}
	</div>
</button>
