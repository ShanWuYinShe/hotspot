<script lang="ts">
	import { Flag, X, Check } from '@lucide/svelte';
	import * as Dialog from '@/components/ui/dialog';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import { createReport } from '@/api';
	import type { ReportReason, ReportTargetType } from '@/api/types';
	import { showToast } from '@/stores/toast.svelte';
	import { requireAuth } from '@/stores/auth.svelte';

	// ReportDialog — 举报理由选择 + 提交的全站唯一举报弹窗（bits-ui Dialog 底座）。
	// 交互收敛自 users/[userId] 页既有实现（createReport + 原因选择按钮组），
	// 供动态/评论（moments 详情）、群组（groups 详情）、消息（ChatView/GroupChatView）复用。
	// spec: api_spec_report.md → reports/create（重复举报幂等回放 is_duplicate=true，均按成功提示）。

	let {
		open,
		targetType,
		targetId,
		targetLabel = '该内容',
		onClose
	}: {
		open: boolean;
		targetType: ReportTargetType;
		targetId: string;
		/** 弹窗标题里的目标名词（如「动态」「评论」「群组」「消息」）。 */
		targetLabel?: string;
		onClose: () => void;
	} = $props();

	// 举报理由（ReportReason 枚举取常用四档；harassment 骚扰 / impersonation 冒充他人）
	const REPORT_REASONS: { id: ReportReason; label: string }[] = [
		{ id: 'spam', label: '垃圾信息' },
		{ id: 'harassment', label: '骚扰攻击' },
		{ id: 'impersonation', label: '冒充他人' },
		{ id: 'other', label: '其他' }
	];

	let reason = $state<ReportReason>('spam');
	let pending = $state(false);

	// 每次打开重置为默认理由，避免上一次目标的选择残留
	$effect(() => {
		if (open) reason = 'spam';
	});

	const submit = async () => {
		if (pending) return;
		if (!requireAuth({ action: '举报', targetId, run: () => void submit() })) return;
		pending = true;
		try {
			const res = await createReport({ target_type: targetType, target_id: targetId, reason });
			if (res.code === 0) {
				showToast('举报已记录，感谢你的反馈', 'success');
				onClose();
			} else {
				showToast(res.message || '举报失败，请重试', 'danger');
			}
		} catch {
			showToast('举报失败，请重试', 'danger');
		} finally {
			pending = false;
		}
	};
</script>

<Dialog.Root {open} onOpenChange={(next) => (next ? undefined : onClose())}>
	<Dialog.Content
		side="bottom"
		class="inset-x-4 bottom-4 mx-auto max-w-md rounded-xl p-5"
		aria-label={`举报${targetLabel}`}
		style="box-shadow:var(--cst-elevation-modal);"
	>
		<div class="flex items-center justify-between">
			<div>
				<p class="eyebrow">举报</p>
				<p class="mt-1 text-md text-text-primary">举报{targetLabel}</p>
			</div>
			<ActionIconButton icon={X} label="关闭" onClick={onClose} />
		</div>
		<div class="mt-4 flex flex-col gap-1.5" aria-label="举报理由">
			{#each REPORT_REASONS as r (r.id)}
				{@const selected = reason === r.id}
				<!-- 选中/未选中 hover 差异化（四态契约）：未选中 hover 淡底预览、选中 hover 加深 -->
				<button
					type="button"
					aria-pressed={selected}
					onclick={() => (reason = r.id)}
					class="hover-btn flex items-center justify-between rounded-sm border px-3 py-2 text-left text-sm transition-colors duration-rapid ease-standard focus-ring
						{selected
						? 'border-accent-border bg-accent-soft text-accent-default hover:bg-interactive-active'
						: 'border-border-default text-text-secondary hover:bg-interactive-hover'}"
				>
					{r.label}
					{#if selected}<Check class="size-3.5" strokeWidth={2.5} />{/if}
				</button>
			{/each}
			<button
				type="button"
				disabled={pending}
				onclick={() => void submit()}
				class="hover-btn mt-1 flex w-full items-center justify-center gap-2 rounded-full border border-border-default py-2.5 text-sm font-medium text-status-danger focus-ring hover:bg-danger-bg disabled:pointer-events-none disabled:opacity-faint"
			>
				<Flag class="size-3.5" strokeWidth={1.75} />
				{pending ? '提交中…' : '提交举报'}
			</button>
		</div>
	</Dialog.Content>
</Dialog.Root>
