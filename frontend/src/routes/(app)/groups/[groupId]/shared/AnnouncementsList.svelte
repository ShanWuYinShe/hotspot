<script lang="ts">
	import { Megaphone, PencilLine, Pin, Trash2 } from '@lucide/svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import type { GroupAnnouncement } from '@/lib/group-data.svelte';

	// AnnouncementsList —— 群公告列表 + （管理者）发公告入口与行内编辑/删除。
	let {
		announcements,
		canManage,
		onCreate,
		onEdit,
		onDelete
	}: {
		announcements: GroupAnnouncement[];
		canManage: boolean;
		onCreate: () => void;
		onEdit: (item: GroupAnnouncement) => void;
		onDelete: (id: string) => void;
	} = $props();
</script>

<div class="mt-6 flex items-center justify-between">
	<p class="text-sm text-text-secondary">群公告</p>
	{#if canManage}
		<button type="button" onclick={onCreate} class="hover-btn hit-slop flex items-center gap-1 rounded-full bg-surface-subtle/60 px-3.5 py-2 text-sm text-text-primary focus-ring hover:bg-surface-subtle">
			<Megaphone size={12} strokeWidth={1.75} />
			发公告
		</button>
	{/if}
</div>
<!-- 空旷内容区 → List bordered（有边框列表契约） -->
<ListContainer class="mt-4 divide-y divide-border-subtle/30">
	{#each announcements as item (item.id)}
		<ListItem class="flex flex-col items-start px-4 py-4" ariaLabel={item.title}>
			<div class="flex items-center gap-1.5 pr-16">
				{#if item.isPinned}
					<Pin size={12} class="shrink-0 text-accent-default" strokeWidth={1.75} />
				{/if}
				<p class="text-sm font-medium text-text-primary">{item.title}</p>
			</div>
			<p class="mt-1 text-sm leading-5 text-text-secondary">{item.body}</p>
			<p class="mt-2 text-xs text-text-secondary">{item.time}</p>
			{#if canManage}
				<div class="absolute right-3 top-3 flex gap-1">
					<button type="button" onclick={() => onEdit(item)} class="hit-slop hover-btn scale-btn flex h-8 w-8 items-center justify-center rounded-full text-text-secondary outline-none hover:bg-interactive-hover focus-visible:ring-2 focus-visible:ring-accent-default" aria-label="编辑公告">
						<PencilLine size={12} strokeWidth={1.75} />
					</button>
					<button type="button" onclick={() => onDelete(item.id)} class="hit-slop hover-btn scale-btn flex h-8 w-8 items-center justify-center rounded-full text-status-danger outline-none hover:bg-danger-bg focus-visible:ring-2 focus-visible:ring-status-danger" aria-label="删除公告">
						<Trash2 size={12} strokeWidth={1.75} />
					</button>
				</div>
			{/if}
		</ListItem>
	{:else}
		<EmptyState bare title="暂无公告" />
	{/each}
</ListContainer>
