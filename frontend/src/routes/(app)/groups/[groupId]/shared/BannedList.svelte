<script lang="ts">
	import { Ban } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import Button from '@/components/ui/button.svelte';
	import type { AdminBannedMemberItem } from '@/api/types';

	// BannedList —— 封禁名单（owner/admin）。
	// 行语言/动作档位统一使用 Button 原语（soft 软填胶囊）。
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

{#if banned.length === 0}
	<EmptyState bare title="暂无封禁成员" description="被移出并封禁的成员会出现在这里。" class="mt-6">
		{#snippet icon()}
			<Ban class="size-7" strokeWidth={1.5} />
		{/snippet}
	</EmptyState>
{:else}
<!-- 子 Tab 栏→内容统一 24；开放页面区域 → bordered 内容软岛（行不另加边框/分隔线） -->
<ListContainer bordered class="mt-6">
	{#each banned as item (item.user.user_id)}
		<ListItem class="flex list-row-standard items-center gap-3 px-3.5 py-3" ariaLabel={`${item.user.nickname} 的封禁记录`}>
			<Avatar src={item.user.avatar_url ?? undefined} name={item.user.nickname} size="lg" />
			<span class="min-w-0 flex-1">
				<span class="block truncate text-md font-medium text-text-primary">{item.user.nickname}</span>
				<span class="block truncate text-xs text-text-secondary">{formatDateTime(item.banned_at)}{item.reason ? ` · ${item.reason}` : ''}</span>
			</span>
			<Button type="button" variant="soft" size="sm" shape="pill" onclick={() => onUnban(item.user.user_id)}>
				解封
			</Button>
		</ListItem>
	{/each}
</ListContainer>
{/if}
