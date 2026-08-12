<script lang="ts">
	// 成员操作底部 sheet（设管理员/取消、封禁、移出）。危险操作二次确认。
	import * as Dialog from '@/components/ui/dialog';
	import { X, ShieldCheck, ShieldOff, Ban, UserMinus } from '@lucide/svelte';

	export type MemberAction = 'set_admin' | 'unset_admin' | 'ban' | 'remove';

	let {
		open,
		member,
		onClose,
		onSelect
	}: {
		open: boolean;
		member: { userId: string; name: string; role: string } | null;
		onClose: () => void;
		onSelect: (action: MemberAction) => void;
	} = $props();

	let confirmAction = $state<MemberAction | null>(null);

	// 打开时重置确认态
	$effect(() => {
		if (open) confirmAction = null;
	});

	const isAdmin = $derived(member?.role === 'admin');

	const choose = (action: MemberAction) => {
		// 危险操作（封禁/移出）走二次确认；设角色直接执行
		if (action === 'ban' || action === 'remove') {
			confirmAction = action;
		} else {
			onSelect(action);
		}
	};
	const confirm = () => {
		if (confirmAction) onSelect(confirmAction);
	};
</script>

<Dialog.Root open={open} onOpenChange={(next) => (next ? undefined : onClose())}>
	<Dialog.Content
		side="bottom"
		class="inset-x-4 bottom-4 mx-auto max-w-md rounded-xl bg-surface-primary p-4"
		aria-label="成员管理"
		style="box-shadow:var(--cst-elevation-modal);"
	>
		<div class="flex items-center justify-between px-1 pb-3">
			<p class="text-md font-medium text-text-primary">{member?.name ?? '成员'}</p>
			<button
				type="button"
				onclick={onClose}
				class="flex h-7 w-7 items-center justify-center rounded-full text-text-secondary outline-none hover:bg-interactive-hover focus-visible:ring-2 focus-visible:ring-accent-default"
				aria-label="关闭"
			>
				<X size={15} />
			</button>
		</div>

		{#if confirmAction}
			<!-- 危险操作二次确认 -->
			<div class="px-1 py-2">
				<p class="text-sm text-text-primary">确认{confirmAction === 'ban' ? '封禁' : '移出'}「{member?.name}」？</p>
				<p class="mt-1 text-xs text-text-secondary">
					{confirmAction === 'ban' ? '封禁后该成员将被移出群组并加入封禁名单' : '移出后该成员需要重新申请加入'}
				</p>
			</div>
			<div class="mt-2 flex items-center justify-end gap-3">
				<button type="button" onclick={() => (confirmAction = null)} class="hover-btn rounded-full px-4 py-2 text-sm text-text-secondary outline-none focus-visible:ring-2 focus-visible:ring-accent-default">取消</button>
				<button type="button" onclick={confirm} class="hover-btn rounded-full px-5 py-2 text-sm font-medium text-text-on-accent outline-none focus-visible:ring-2 focus-visible:ring-status-danger enabled:bg-status-danger">
					确认{confirmAction === 'ban' ? '封禁' : '移出'}
				</button>
			</div>
		{:else}
			<!-- 操作列表 -->
			<div class="flex flex-col gap-1">
				<button type="button" onclick={() => choose(isAdmin ? 'unset_admin' : 'set_admin')} class="hover-btn flex items-center gap-2.5 rounded-md px-3 py-3 text-sm text-text-primary outline-none focus-visible:ring-2 focus-visible:ring-accent-default">
					{#if isAdmin}
						<ShieldOff size={15} class="text-text-secondary" strokeWidth={1.75} />
						取消管理员
					{:else}
						<ShieldCheck size={15} class="text-accent-default" strokeWidth={1.75} />
						设为管理员
					{/if}
				</button>
				<button type="button" onclick={() => choose('ban')} class="hover-btn flex items-center gap-2.5 rounded-md px-3 py-3 text-sm text-status-danger outline-none focus-visible:ring-2 focus-visible:ring-status-danger">
					<Ban size={15} strokeWidth={1.75} />
					封禁该成员
				</button>
				<button type="button" onclick={() => choose('remove')} class="hover-btn flex items-center gap-2.5 rounded-md px-3 py-3 text-sm text-status-danger outline-none focus-visible:ring-2 focus-visible:ring-status-danger">
					<UserMinus size={15} strokeWidth={1.75} />
					移出群组
				</button>
			</div>
		{/if}
	</Dialog.Content>
</Dialog.Root>
