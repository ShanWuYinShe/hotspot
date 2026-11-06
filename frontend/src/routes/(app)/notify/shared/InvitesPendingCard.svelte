<script lang="ts">
	import type { GroupInviteItem } from '@/api/types';
	import Avatar from '@/components/shared/Avatar.svelte';
	import Button from '@/components/ui/button.svelte';
	import SectionHead from '@/components/home/SectionHead.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import { formatMomentTime } from '@/lib/utils';

	let {
		invites,
		invitePending,
		onAccept,
		onReject
	}: {
		invites: GroupInviteItem[];
		invitePending: Set<string>;
		onAccept: (inviteId: string) => void;
		onReject: (inviteId: string) => void;
	} = $props();
</script>

{#if invites.length > 0}
	<SectionHead variant="eyebrow" title="入群邀请" count={invites.length} />
	<ListContainer bordered class="mb-6">
		{#each invites as inv (inv.invite_id)}
			<div role="listitem" class="flex items-center gap-x-3.5 px-3 py-3.5">
				<Avatar src={inv.group.avatar_url} name={inv.group.name} size="lg" shape="rounded" />
				<span class="min-w-0 flex-1">
					<span class="block truncate text-sm font-medium text-text-primary">{inv.group.name}</span>
					<span class="block truncate text-xs text-text-secondary">{inv.inviter.nickname} 邀请你 · {formatMomentTime(inv.created_at)}</span>
				</span>
				<div class="flex shrink-0 flex-col items-stretch gap-1.5 sm:flex-row sm:items-center">
					<Button
						type="button"
						size="sm"
						variant="primary"
						shape="pill"
						class="hit-slop-44 px-3.5"
						disabled={invitePending.has(inv.invite_id)}
						onclick={() => onAccept(inv.invite_id)}
					>接受</Button>
					<Button
						type="button"
						size="sm"
						variant="soft"
						shape="pill"
						class="hit-slop-44 px-3.5"
						disabled={invitePending.has(inv.invite_id)}
						onclick={() => onReject(inv.invite_id)}
					>拒绝</Button>
				</div>
			</div>
		{/each}
	</ListContainer>
{/if}
