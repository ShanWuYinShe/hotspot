<script lang="ts">
	import { Ban } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import type { AdminBannedMemberItem } from '@/api/types';

	// BannedList —— 封禁名单（owner/admin）。
	// R13：行语言/动作档位统一；空态升级为带图标与说明的 EmptyState。
	let {
		banned,
		formatDateTime,
		onUnban
	}: {
		banned: AdminBannedMemberItem[];
		formatDateTime: (iso: string) => string;
		onUnban: (userId: string) => void;
	} = $props();

	const CHIP_QUIET =
		'hover-btn hit-slop inline-flex h-8 shrink-0 items-center justify-center rounded-full bg-surface-subtle/60 px-3.5 text-sm text-text-primary transition-colors hover:bg-surface-subtle focus-ring';
</script>

{#if banned.length === 0}
	<EmptyState bare title="暂无封禁成员" description="被移出并封禁的成员会出现在这里。" class="mt-6">
		{#snippet icon()}
			<Ban class="size-7" strokeWidth={1.5} />
		{/snippet}
	</EmptyState>
{:else}
<!-- 子 Tab 栏→内容统一 24 -->
<ListContainer class="mt-6 divide-y divide-border-subtle/30">
	{#each banned as item (item.user.user_id)}
		<ListItem class="flex list-row-standard items-center gap-3 px-3.5 py-3" ariaLabel={`${item.user.nickname} 的封禁记录`}>
			<Avatar src={item.user.avatar_url ?? undefined} name={item.user.nickname} size="lg" />
			<span class="min-w-0 flex-1">
				<span class="block truncate text-md font-medium text-text-primary">{item.user.nickname}</span>
				<span class="block truncate text-xs text-text-secondary">{formatDateTime(item.banned_at)}{item.reason ? ` · ${item.reason}` : ''}</span>
			</span>
			<button type="button" onclick={() => onUnban(item.user.user_id)} class={CHIP_QUIET}>解封</button>
		</ListItem>
	{/each}
</ListContainer>
{/if}
