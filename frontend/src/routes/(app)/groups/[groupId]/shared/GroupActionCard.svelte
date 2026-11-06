<script lang="ts">
	import { Bell, LogOut, MessageCircle, Settings, X } from '@lucide/svelte';
	import Button from '@/components/ui/button.svelte';
	import { popOn } from '@/lib/motion';

	let {
		joined,
		myRole,
		joinPendingApproval,
		joinPop,
		onOpenChat,
		onOpenSettings,
		onLeave,
		onCancelApplication,
		onRequestJoin
	}: {
		joined: boolean;
		myRole: string | null;
		joinPendingApproval: boolean;
		joinPop: number;
		onOpenChat: () => void;
		onOpenSettings: () => void;
		onLeave: () => void;
		onCancelApplication: () => void;
		onRequestJoin: () => void;
	} = $props();
</script>

<section aria-label="群组操作" class="flex h-fit flex-col gap-3.5 rounded-lg bg-surface-island p-4 shadow-xs">
	{#if joined}
		<Button type="button" variant="primary" size="lg" class="w-full" onclick={onOpenChat}>
			<MessageCircle size={15} />
			打开群聊
		</Button>
		{#if myRole === 'owner' || myRole === 'admin'}
			<Button type="button" variant="soft" size="lg" class="w-full" onclick={onOpenSettings}>
				<Settings size={15} />
				群管理
			</Button>
		{/if}
		<div class="border-t border-border-subtle/50 pt-3">
			<Button
				type="button"
				variant="danger-soft"
				size="md"
				class="hit-slop w-full"
				onclick={onLeave}
			>
				<LogOut size={14} strokeWidth={1.75} />
				已加入 · 点击退出
			</Button>
		</div>
	{:else if joinPendingApproval}
		<Button
			type="button"
			variant="soft"
			size="lg"
			class="w-full"
			onclick={onCancelApplication}
		>
			<X size={15} strokeWidth={1.75} />
			审核中 · 点击取消申请
		</Button>
		<p class="px-2 text-center text-xs leading-5 text-text-secondary">群主通过申请后即可进入群聊</p>
	{:else}
		<Button
			type="button"
			variant="primary"
			size="lg"
			class="w-full"
			onclick={onRequestJoin}
		>
			<span class="inline-flex" use:popOn={joinPop}>加入群组</span>
		</Button>
		<p class="px-2 text-center text-xs leading-5 text-text-secondary flex items-center justify-center gap-1.5">
			<Bell size={13} class="text-accent-default/80 shrink-0" />
			<span>加入后可管理群组通知并进入群聊</span>
		</p>
	{/if}
</section>
