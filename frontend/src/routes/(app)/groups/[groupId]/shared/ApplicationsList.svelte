<script lang="ts">
	import { UserPlus } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import Button from '@/components/ui/button.svelte';
	import type { GroupApplicationItem } from '@/api/types';

	// ApplicationsList —— 入群申请审批（owner/admin 工作台语义，默认仅 pending）。
	// 行语言与成员列表对齐，同意/拒绝动作统一使用 Button 原语（primary 实底 / soft 软填胶囊）。
	let {
		applications,
		formatDateTime,
		pending,
		onApprove,
		onReject
	}: {
		applications: GroupApplicationItem[];
		formatDateTime: (iso: string) => string;
		/** 处理中的申请 id 集合（防重复提交）：按钮禁用 + 透明度反馈 */
		pending: Set<string>;
		onApprove: (id: string) => void;
		onReject: (id: string) => void;
	} = $props();
</script>

{#if applications.length === 0}
	<!-- 子 Tab 栏→内容统一 24（空态与列表同值，切换子 Tab 不跳） -->
	<EmptyState bare title="暂无待处理的申请" description="新的入群申请会出现在这里。" class="mt-6">
		{#snippet icon()}
			<UserPlus class="size-7" strokeWidth={1.5} />
		{/snippet}
	</EmptyState>
{:else}
<ListContainer bordered class="mt-6">
	{#each applications as app (app.application_id)}
		<ListItem class="flex list-row-standard items-center gap-3 px-3.5 py-3" ariaLabel={`${app.applicant.nickname} 的入群申请`}>
			<Avatar src={app.applicant.avatar_url ?? undefined} name={app.applicant.nickname} size="lg" />
			<span class="min-w-0 flex-1">
				<span class="block truncate text-md font-medium text-text-primary">{app.applicant.nickname}</span>
				<span class="block truncate text-xs text-text-secondary">{formatDateTime(app.created_at)}{app.message ? ` · ${app.message}` : ''}</span>
			</span>
			<Button
				type="button"
				variant="primary"
				size="sm"
				shape="pill"
				disabled={pending.has(app.application_id)}
				onclick={() => onApprove(app.application_id)}
			>
				同意
			</Button>
			<Button
				type="button"
				variant="soft"
				size="sm"
				shape="pill"
				disabled={pending.has(app.application_id)}
				onclick={() => onReject(app.application_id)}
			>
				拒绝
			</Button>
		</ListItem>
	{/each}
</ListContainer>
{/if}
