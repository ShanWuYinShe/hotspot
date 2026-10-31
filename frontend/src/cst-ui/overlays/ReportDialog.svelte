<script lang="ts">
	import { Flag, Check } from '@lucide/svelte';
	import * as Dialog from '@/components/ui/dialog';
	import Button from '@/components/ui/button.svelte';
	import { createReport } from '@/api';
	import type { ReportReason, ReportTargetType } from '@/api/types';
	import { showToast } from '@/stores/toast.svelte';
	import { requireAuth } from '@/stores/auth.svelte';
	import { platform } from '@/stores/platform.svelte';
	import { cn } from '@/lib/utils';

	// ReportDialog — 举报理由选择 + 提交的全站唯一举报弹窗（bits-ui Dialog 底座）。
	// 交互收敛自 users/[userId] 页既有实现（createReport + 原因选择按钮组），
	// 供动态/评论（moments 详情）、群组（groups 详情）、消息（ChatView/GroupChatView）复用。
	// spec: api_spec_report.md → reports/create（重复举报幂等回放 is_duplicate=true，均按成功提示）。
	//
	// R13：弹层形态按设备分派——移动端底部 sheet（拇指可达），桌面/平板居中模态
	// （与 ConfirmDialog 同一套"图标头 + 标题 + 描述 + 动作行"语言）；理由改为单选行，
	// 提交/取消为规范按钮（危险动作用 danger 实底，不再用描边文字钮）。

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
	const REPORT_REASONS: { id: ReportReason; label: string; hint: string }[] = [
		{ id: 'spam', label: '垃圾信息', hint: '广告、刷屏、诱导分享' },
		{ id: 'harassment', label: '骚扰攻击', hint: '辱骂、人身攻击、恶意骚扰' },
		{ id: 'impersonation', label: '冒充他人', hint: '冒用身份或伪造信息' },
		{ id: 'other', label: '其他', hint: '不适合社区的其他内容' }
	];

	let reason = $state<ReportReason>('spam');
	let pending = $state(false);

	// 每次打开重置为默认理由，避免上一次目标的选择残留
	$effect(() => {
		if (open) reason = 'spam';
	});

	const isMobile = $derived(platform.platform === 'mobile');

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
		side={isMobile ? 'bottom' : 'center'}
		class={cn('mx-auto w-full max-w-md', isMobile ? 'px-5 pt-5 pb-6' : 'px-6 pt-6 pb-5')}
		aria-label={`举报${targetLabel}`}
	>
		<!-- 图标头 + 标题 + 一句说明（与 ConfirmDialog 同语言） -->
		<div class="flex items-start gap-3.5">
			<div
				class="flex size-11 shrink-0 items-center justify-center rounded-full bg-danger-bg text-text-status-danger-on-soft ring-1 ring-status-danger-border"
			>
				<Flag class="size-5" strokeWidth={1.75} />
			</div>
			<div class="min-w-0 flex-1">
				<Dialog.Title class="mt-0.5 text-lg font-semibold tracking-title text-text-primary">
					举报{targetLabel}
				</Dialog.Title>
				<Dialog.Description class="mt-1 text-md leading-relaxed text-text-secondary">
					请选择最符合的理由，我们会尽快核实处理。
				</Dialog.Description>
			</div>
		</div>

		<!-- 理由单选行：左侧单选点 + 主文案 + 次要说明，选中 accent 淡底（on-soft 文字档） -->
		<div class="mt-5 flex flex-col gap-1.5" role="radiogroup" aria-label="举报理由">
			{#each REPORT_REASONS as r (r.id)}
				{@const selected = reason === r.id}
				<button
					type="button"
					role="radio"
					aria-checked={selected}
					onclick={() => (reason = r.id)}
					class="group flex w-full items-center gap-3 rounded-md px-3 py-2.5 text-left transition-colors duration-rapid focus-ring {selected
						? 'bg-accent-soft'
						: 'hover:bg-interactive-hover'}"
				>
					<span
						class="flex size-4.5 shrink-0 items-center justify-center rounded-full border-2 transition-colors {selected
							? 'border-accent-default bg-accent-default text-text-on-accent'
							: 'border-border-strong'}"
						aria-hidden="true"
					>
						{#if selected}<Check class="size-2.5" strokeWidth={3} />{/if}
					</span>
					<span class="min-w-0 flex-1">
						<span class="block text-sm font-medium {selected ? 'text-text-primary' : 'text-text-primary'}">{r.label}</span>
						<span class="mt-0.5 block truncate text-xs {selected ? 'text-text-accent-on-soft' : 'text-text-secondary'}">{r.hint}</span>
					</span>
				</button>
			{/each}
		</div>

		<!-- 动作行：取消（次级）+ 提交举报（危险实底） -->
		<div class="mt-5 flex items-center justify-end gap-2.5">
			<Button type="button" variant="secondary" size="md" onclick={onClose}>取消</Button>
			<Button type="button" variant="danger" size="md" loading={pending} onclick={() => void submit()}>
				提交举报
			</Button>
		</div>
	</Dialog.Content>
</Dialog.Root>
