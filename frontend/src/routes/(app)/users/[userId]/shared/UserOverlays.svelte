<script lang="ts">
	import { Check, Flag, PencilLine, UserX, X } from '@lucide/svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import Button from '@/components/ui/button.svelte';
	import ConfirmDialog from '@/components/overlays/ConfirmDialog.svelte';
	import ReportDialog from '@/components/overlays/ReportDialog.svelte';
	import Input from '@/components/forms/Input.svelte';
	import * as Dialog from '@/components/ui/dialog';

	let {
		userName,
		userId,
		moreOpen = $bindable(false),
		remarkDraft = $bindable(''),
		followed,
		savingRemark,
		blocked,
		confirmBlockOpen = $bindable(false),
		confirmUnblockOpen = $bindable(false),
		reportOpen = $bindable(false),
		isMobile,
		onSaveRemark,
		onToggleBlock
	}: {
		userName: string;
		userId: string;
		moreOpen?: boolean;
		remarkDraft?: string;
		followed: boolean;
		savingRemark: boolean;
		blocked: boolean;
		confirmBlockOpen?: boolean;
		confirmUnblockOpen?: boolean;
		reportOpen?: boolean;
		isMobile: boolean;
		onSaveRemark: () => void;
		onToggleBlock: () => void;
	} = $props();
</script>

<Dialog.Root open={moreOpen} onOpenChange={(next) => (next ? undefined : (moreOpen = false))}>
	<Dialog.Content side="bottom" floating class="p-5" aria-label={`${userName} 的更多操作`}>
		<div class="flex items-center justify-between">
			<div>
				<p class="eyebrow">关系与安全</p>
				<p class="mt-1 text-md text-text-primary">{userName}</p>
			</div>
			<ActionIconButton icon={X} label="关闭" onClick={() => (moreOpen = false)} />
		</div>
		<Input
			class="mt-5"
			label="备注名称"
			value={remarkDraft}
			onChange={(v) => (remarkDraft = v.slice(0, 24))}
			placeholder="只对你可见"
			maxlength={24}
		/>
		<Button
			type="button"
			variant="secondary"
			disabled={!followed || savingRemark}
			class="mt-2 w-full"
			onclick={onSaveRemark}
		>
			<PencilLine size={14} />
			{savingRemark ? '保存中…' : '保存备注'}
		</Button>
		{#if !followed}
			<p class="mt-1 text-xs text-text-secondary">关注对方后可设备注</p>
		{/if}
		<Button
			type="button"
			variant="ghost"
			class="mt-2 w-full text-status-danger hover:text-text-status-danger-on-soft hover:bg-danger-bg"
			onclick={() => {
				moreOpen = false;
				if (blocked) confirmUnblockOpen = true;
				else confirmBlockOpen = true;
			}}
		>
			<UserX size={14} />
			{blocked ? '解除屏蔽' : '屏蔽用户'}
		</Button>
		<Button
			type="button"
			variant="ghost"
			class="mt-1 w-full text-text-secondary"
			onclick={() => {
				moreOpen = false;
				reportOpen = true;
			}}
		>
			<Flag size={14} />
			举报用户
		</Button>
	</Dialog.Content>
</Dialog.Root>

<ReportDialog
	open={reportOpen}
	targetType="user"
	targetId={userId}
	targetLabel={userName}
	onClose={() => (reportOpen = false)}
/>

<ConfirmDialog
	open={confirmBlockOpen}
	tone="danger"
	icon={UserX}
	title="屏蔽该用户？"
	description="屏蔽后将无法收到对方消息与动态，且自动取消关注。"
	confirmLabel="屏蔽"
	layout={isMobile ? 'stacked' : 'row'}
	onCancel={() => (confirmBlockOpen = false)}
	onConfirm={() => {
		confirmBlockOpen = false;
		onToggleBlock();
	}}
/>

<ConfirmDialog
	open={confirmUnblockOpen}
	tone="neutral"
	icon={Check}
	title="解除屏蔽？"
	description="解除后可以重新关注对方并接收其动态。"
	confirmLabel="解除屏蔽"
	layout={isMobile ? 'stacked' : 'row'}
	onCancel={() => (confirmUnblockOpen = false)}
	onConfirm={() => {
		confirmUnblockOpen = false;
		onToggleBlock();
	}}
/>
