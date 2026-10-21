<script lang="ts">
	import Avatar from '@/components/shared/Avatar.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import type { AdminBannedMemberItem } from '@/api/types';

	// BannedList —— 封禁名单（owner/admin）。
	let {
		banned,
		formatDateTime,
		onUnban
	}: {
		banned: AdminBannedMemberItem[];
		formatDateTime: (iso: string) => string;
		onUnban: (userId: string) => void;
	} = $props();
</script>

<!-- 子 Tab 栏→内容统一 24 -->
<ListContainer bordered class="mt-6">
	{#each banned as item (item.user.user_id)}
		<ListItem class="flex list-row-standard items-center gap-3 px-3.5 py-3" ariaLabel={`${item.user.nickname} 的封禁记录`}>
			<Avatar src={item.user.avatar_url ?? undefined} name={item.user.nickname} size="lg" />
			<span class="min-w-0 flex-1">
				<span class="block truncate text-sm text-text-primary">{item.user.nickname}</span>
				<span class="text-xs text-text-secondary">{formatDateTime(item.banned_at)}{item.reason ? ` · ${item.reason}` : ''}</span>
			</span>
			<button type="button" onclick={() => onUnban(item.user.user_id)} class="hover-btn hit-slop rounded-full border border-border-default px-3.5 py-2 text-sm text-text-primary focus-ring hover:border-border-strong">解封</button>
		</ListItem>
	{:else}
		<EmptyState bare title="暂无封禁成员" />
	{/each}
</ListContainer>
