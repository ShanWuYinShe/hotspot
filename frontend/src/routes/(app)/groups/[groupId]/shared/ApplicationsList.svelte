<script lang="ts">
	import Avatar from '@/components/shared/Avatar.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import type { GroupApplicationItem } from '@/api/types';

	// ApplicationsList —— 入群申请审批（owner/admin 工作台语义，默认仅 pending）。
	let {
		applications,
		formatDateTime,
		onApprove,
		onReject
	}: {
		applications: GroupApplicationItem[];
		formatDateTime: (iso: string) => string;
		onApprove: (id: string) => void;
		onReject: (id: string) => void;
	} = $props();
</script>

<ListContainer bordered class="mt-4">
	{#each applications as app (app.application_id)}
		<ListItem class="flex list-row-standard items-center gap-3 px-3.5 py-3" ariaLabel={`${app.applicant.nickname} 的入群申请`}>
			<Avatar src={app.applicant.avatar_url ?? undefined} name={app.applicant.nickname} size="lg" />
			<span class="min-w-0 flex-1">
				<span class="block truncate text-sm text-text-primary">{app.applicant.nickname}</span>
				<span class="text-xs text-text-secondary">{formatDateTime(app.created_at)}{app.message ? ` · ${app.message}` : ''}</span>
			</span>
			<button type="button" onclick={() => onApprove(app.application_id)} class="hover-btn hit-slop rounded-full bg-accent-default px-3.5 py-1.5 text-sm font-medium text-text-on-accent shadow-raised-sm focus-ring hover:shadow-raised-md">同意</button>
			<button type="button" onclick={() => onReject(app.application_id)} class="hover-btn rounded-full border border-border-default px-3.5 py-1.5 text-sm text-text-primary focus-ring">拒绝</button>
		</ListItem>
	{:else}
		<EmptyState bare title="暂无待处理的申请" />
	{/each}
</ListContainer>
