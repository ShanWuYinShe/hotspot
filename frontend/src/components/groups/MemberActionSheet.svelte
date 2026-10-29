<script lang="ts">
	// 成员操作底部 sheet：设管理员/取消、禁言/解除（时长）、封禁、移出。危险操作二次确认。
	import * as Dialog from '@/components/ui/dialog';
	import Button from '@/components/ui/button.svelte';
	import { X, ShieldCheck, ShieldOff, Ban, UserMinus, VolumeOff, Volume2, Clock } from '@lucide/svelte';

	export type MemberAction = 'set_admin' | 'unset_admin' | 'mute' | 'unmute' | 'ban' | 'remove';

	let {
		open,
		member,
		onClose,
		onSelect
	}: {
		open: boolean;
		member: { userId: string; name: string; role: string; mutedUntil?: string } | null;
		onClose: () => void;
		onSelect: (action: MemberAction, durationMinutes?: number) => void;
	} = $props();

	type View = 'actions' | 'mute_options' | 'confirm';
	let view = $state<View>('actions');
	let confirmAction = $state<MemberAction | null>(null);

	const MUTE_PRESETS = [
		{ label: '10 分钟', minutes: 10 },
		{ label: '1 小时', minutes: 60 },
		{ label: '12 小时', minutes: 720 },
		{ label: '1 天', minutes: 1440 }
	];

	// 打开时重置视图
	$effect(() => {
		if (open) { view = 'actions'; confirmAction = null; }
	});

	const isAdmin = $derived(member?.role === 'admin');
	const isMuted = $derived(Boolean(member?.mutedUntil));

	const choose = (action: MemberAction) => {
		if (action === 'mute') {
			view = 'mute_options';
		} else if (action === 'ban' || action === 'remove') {
			confirmAction = action;
			view = 'confirm';
		} else {
			onSelect(action);
		}
	};
</script>

<Dialog.Root open={open} onOpenChange={(next) => (next ? undefined : onClose())}>
	<Dialog.Content side="bottom" floating class="p-4" aria-label="成员管理">
		<div class="flex items-center justify-between px-1 pb-3">
			<div class="min-w-0">
				<p class="text-md font-semibold text-text-primary">{member?.name ?? '成员'}</p>
				<p class="mt-0.5 text-xs text-text-secondary">成员管理</p>
			</div>
			<button
				type="button"
				onclick={onClose}
				class="hit-slop flex size-8 shrink-0 items-center justify-center rounded-full text-text-secondary focus-ring hover:bg-interactive-hover"
				aria-label="关闭"
			>
				<X size={15} strokeWidth={1.75} />
			</button>
		</div>

		{#if view === 'actions'}
			<div class="flex flex-col gap-1">
				<button type="button" onclick={() => choose(isAdmin ? 'unset_admin' : 'set_admin')} class="hover-btn flex items-center gap-2.5 rounded-md px-3 py-3 text-sm text-text-primary outline-none focus-visible:ring-2 focus-visible:ring-accent-default">
					{#if isAdmin}
						<ShieldOff size={15} class="text-text-secondary" strokeWidth={1.75} />取消管理员
					{:else}
						<ShieldCheck size={15} class="text-accent-default" strokeWidth={1.75} />设为管理员
					{/if}
				</button>
				<button type="button" onclick={() => choose(isMuted ? 'unmute' : 'mute')} class="hover-btn flex items-center gap-2.5 rounded-md px-3 py-3 text-sm text-text-primary outline-none focus-visible:ring-2 focus-visible:ring-accent-default">
					{#if isMuted}
						<Volume2 size={15} class="text-text-secondary" strokeWidth={1.75} />解除禁言
					{:else}
						<VolumeOff size={15} class="text-accent-default" strokeWidth={1.75} />禁言
					{/if}
				</button>
				<button type="button" onclick={() => choose('ban')} class="hover-btn flex items-center gap-2.5 rounded-md px-3 py-3 text-sm text-status-danger outline-none focus-visible:ring-2 focus-visible:ring-status-danger">
					<Ban size={15} strokeWidth={1.75} />封禁该成员
				</button>
				<button type="button" onclick={() => choose('remove')} class="hover-btn flex items-center gap-2.5 rounded-md px-3 py-3 text-sm text-status-danger outline-none focus-visible:ring-2 focus-visible:ring-status-danger">
					<UserMinus size={15} strokeWidth={1.75} />移出群组
				</button>
			</div>
		{:else if view === 'mute_options'}
			<div class="flex flex-col gap-1">
				<div class="flex items-center gap-2 px-3 pb-2 text-xs text-text-secondary">
					<Clock size={13} strokeWidth={1.75} />选择禁言时长
				</div>
				{#each MUTE_PRESETS as preset}
					<button type="button" onclick={() => onSelect('mute', preset.minutes)} class="hover-btn flex items-center justify-between rounded-md px-3 py-3 text-sm text-text-primary outline-none focus-visible:ring-2 focus-visible:ring-accent-default">
						{preset.label}<span class="text-xs text-text-secondary">→</span>
					</button>
				{/each}
				<button type="button" onclick={() => (view = 'actions')} class="hover-btn mt-1 rounded-md px-3 py-2.5 text-sm text-text-secondary outline-none focus-visible:ring-2 focus-visible:ring-accent-default">返回</button>
			</div>
		{:else}
			<!-- 危险操作二次确认 -->
			<div class="px-1 py-2">
				<p class="text-sm text-text-primary">确认{confirmAction === 'ban' ? '封禁' : '移出'}「{member?.name}」？</p>
				<p class="mt-1 text-xs text-text-secondary">{confirmAction === 'ban' ? '封禁后该成员将被移出群组并加入封禁名单' : '移出后该成员需要重新申请加入'}</p>
			</div>
			<div class="mt-2 flex items-center justify-end gap-3">
				<Button type="button" variant="soft" size="md" onclick={() => { view = 'actions'; confirmAction = null; }}>取消</Button>
				<Button type="button" variant="danger" size="md" onclick={() => { if (confirmAction) onSelect(confirmAction); }}>
					确认{confirmAction === 'ban' ? '封禁' : '移出'}
				</Button>
			</div>
		{/if}
	</Dialog.Content>
</Dialog.Root>
