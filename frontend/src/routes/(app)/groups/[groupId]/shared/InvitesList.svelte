<script lang="ts">
	import { Send } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import Button from '@/components/ui/button.svelte';
	import type { GroupSentInviteItem } from '@/api/types';

	// InvitesList —— 已发邀请（默认仅 pending，revoked 含过期自动撤销）。
	// 行语言/动作档位与成员、申请列表统一，撤销动作统一使用 Button 原语（soft 软填胶囊）。
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

{#if invites.length === 0}
	<EmptyState bare title="暂无待处理的邀请" description="你发出的邀请会出现在这里，可随时撤销。" class="mt-6">
		{#snippet icon()}
			<Send class="size-7" strokeWidth={1.5} />
		{/snippet}
	</EmptyState>
{:else}
<!-- 子 Tab 栏→内容统一 24 -->
<ListContainer class="mt-6 divide-y divide-border-subtle/30">
	{#each invites as inv (inv.invite_id)}
		<ListItem class="flex list-row-standard items-center gap-3 px-3.5 py-3" ariaLabel={`${inv.invitee.nickname} 的入群邀请`}>
			<Avatar src={inv.invitee.avatar_url ?? undefined} name={inv.invitee.nickname} size="lg" />
			<span class="min-w-0 flex-1">
				<span class="block truncate text-md font-medium text-text-primary">{inv.invitee.nickname}</span>
				<span class="block truncate text-xs text-text-secondary">{inviteStatusText(inv.status)} · {formatDateTime(inv.created_at)}</span>
			</span>
			{#if inv.status === 'pending'}
				<Button type="button" variant="soft" size="sm" shape="pill" onclick={() => onRevoke(inv.invite_id)}>
					撤销
				</Button>
			{/if}
		</ListItem>
	{/each}
</ListContainer>
{/if}
