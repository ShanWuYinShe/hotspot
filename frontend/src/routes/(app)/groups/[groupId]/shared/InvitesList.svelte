<script lang="ts">
	import Avatar from '@/components/shared/Avatar.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import type { GroupSentInviteItem } from '@/api/types';

	// InvitesList —— 已发邀请（默认仅 pending，revoked 含过期自动撤销）。
	let {
		invites,
		inviteStatusText,
		formatDateTime,
		onRevoke
	}: {
		invites: GroupSentInviteItem[];
		inviteStatusText: (status: string) => string;
		formatDateTime: (iso: string) => string;
		onRevoke: (id: string) => void;
	} = $props();
</script>

<ListContainer bordered class="mt-4">
	{#each invites as inv (inv.invite_id)}
		<ListItem class="flex list-row-standard items-center gap-3 px-3.5 py-3" ariaLabel={`${inv.invitee.nickname} 的入群邀请`}>
			<Avatar src={inv.invitee.avatar_url ?? undefined} name={inv.invitee.nickname} size="lg" />
			<span class="min-w-0 flex-1">
				<span class="block truncate text-sm text-text-primary">{inv.invitee.nickname}</span>
				<span class="text-xs text-text-secondary">{inviteStatusText(inv.status)} · {formatDateTime(inv.created_at)}</span>
			</span>
			{#if inv.status === 'pending'}
				<button type="button" onclick={() => onRevoke(inv.invite_id)} class="hover-btn hit-slop rounded-full border border-border-default px-3.5 py-2 text-sm text-text-primary focus-ring hover:border-border-strong">撤销</button>
			{/if}
		</ListItem>
	{:else}
		<EmptyState bare title="暂无待处理的邀请" />
	{/each}
</ListContainer>
