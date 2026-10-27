<script lang="ts">
	import { Hourglass } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import SectionHead from '@/components/home/SectionHead.svelte';
	import Button from '@/components/ui/button.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import ConfirmDialog from '@/components/overlays/ConfirmDialog.svelte';
	import { showToast } from '@/stores/toast.svelte';
	import { listMyApplications, cancelGroupApplication, netError } from '@/api';
	import type { GroupApplicationItem, ApplicationStatus } from '@/api/types';
	import { formatDateTime } from '@/lib/utils';

	// 我的入群申请进度（spec: api_spec_group.md → groups/applications/fetch_submitted +
	// applications/cancel）：展示我提交的入群申请状态流转（pending/accepted/rejected/revoked），
	// pending 申请可撤回。被 SectionGroups（/me/groups 分区）内嵌，自身不拉群组数据。

	// 状态胶囊统一「淡底 + on-soft 文字」档（R12 口径）：不再用 danger 实底大色块，
	// 与举报页/关系页的状态语言一致。
	const STATUS_META: Record<ApplicationStatus, { label: string; class: string }> = {
		pending: { label: '审核中', class: 'bg-status-warning-bg text-text-status-warning-on-soft' },
		accepted: { label: '已通过', class: 'bg-status-online-bg text-text-status-online-on-soft' },
		rejected: { label: '未通过', class: 'bg-danger-bg text-text-status-danger-on-soft' },
		revoked: { label: '已撤回', class: 'bg-surface-subtle text-text-secondary' }
	};

	let items = $state<GroupApplicationItem[]>([]);
	let loading = $state(true);
	// 10100（未登录）是正常业务态：整个区块静默隐藏（外层已有登录引导），不报错
	let hidden = $state(false);
	let reloadKey = $state(0);

	$effect(() => {
		void reloadKey;
		let cancelled = false;
		async function load() {
			loading = true;
			try {
				const res = await listMyApplications({ status: 'all' }).catch(() => netError(null));
				if (cancelled) return;
				if (res.code === 10100) {
					hidden = true;
					return;
				}
				if (res.code === 0 && res.data) {
					items = res.data.items;
				}
			} finally {
				if (!cancelled) loading = false;
			}
		}
		void load();
		return () => {
			cancelled = true;
		};
	});

	// 撤回 pending 申请（防连点 + 确认弹窗）
	let cancelTarget = $state<GroupApplicationItem | null>(null);
	let cancelPending = $state(false);

	const confirmCancel = async () => {
		if (!cancelTarget || cancelPending) return;
		cancelPending = true;
		try {
			// spec: applications/cancel { group_id, application_id }
			const res = await cancelGroupApplication({ group_id: cancelTarget.group.group_id, application_id: cancelTarget.application_id });
			if (res.code === 0) {
				items = items.map((a) =>
					a.application_id === cancelTarget?.application_id ? { ...a, status: 'revoked' as const } : a
				);
				showToast('申请已撤回', 'success');
			} else {
				showToast(res.message || '撤回失败，请重试', 'danger');
			}
		} catch {
			showToast('撤回失败，请重试', 'danger');
		} finally {
			cancelPending = false;
			cancelTarget = null;
		}
	};
</script>

{#if !hidden && !loading}
	{#if items.length > 0}
		<section class="mt-6" aria-label="我的入群申请">
			<SectionHead title="我的入群申请" count={items.length} icon={Hourglass} />
			<ListContainer bordered>
				{#each items as a (a.application_id)}
					{@const meta = STATUS_META[a.status]}
					<ListItem class="flex items-center gap-3 px-3 py-3">
						<Avatar src={a.group.avatar_url ?? ''} name={a.group.name} size="md" shape="rounded" class="shrink-0" />
						<div class="min-w-0 flex-1">
							<p class="truncate text-sm font-medium text-text-primary">{a.group.name}</p>
							<p class="mt-0.5 text-xs text-text-secondary">申请于 {formatDateTime(a.created_at)}</p>
						</div>
						<span class="ml-auto inline-flex h-5 shrink-0 items-center justify-center rounded-full px-2 text-xxs font-medium leading-none {meta.class}">
							{meta.label}
						</span>
						{#if a.status === 'pending'}
							<Button
								type="button"
								variant="soft"
								size="sm"
								class="hit-slop shrink-0 gap-1 rounded-full px-3 py-1.5 text-xs"
								aria-label={`撤回加入「${a.group.name}」的申请`}
								onclick={() => (cancelTarget = a)}
							>
								撤回
							</Button>
						{/if}
					</ListItem>
				{/each}
			</ListContainer>
		</section>
	{/if}
{:else if !hidden}
	<InlineLoading class="py-6" label="加载入群申请…" />
{/if}

<ConfirmDialog
	open={cancelTarget !== null}
	title={`撤回加入「${cancelTarget?.group.name ?? ''}」的申请？`}
	description="撤回后需重新提交申请才能再次加入该群。"
	icon={Hourglass}
	confirmVariant="primary"
	confirmLabel="撤回申请"
	loading={cancelPending}
	onCancel={() => (cancelTarget = null)}
	onConfirm={() => void confirmCancel()}
/>