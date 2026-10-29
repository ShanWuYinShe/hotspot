<script lang="ts">
	import { Megaphone, PencilLine, Pin, Trash2 } from '@lucide/svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import Badge from '@/components/data-display/Badge.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import SectionHead from '@/components/home/SectionHead.svelte';
	import Button from '@/components/ui/button.svelte';
	import type { GroupAnnouncement } from '@/lib/group-data.svelte';

	// AnnouncementsList —— 群公告列表 + （管理者）发公告入口与行内编辑/删除。
	// R13：标题行走 SectionHead（标题 + 计数 + 右侧发公告），置顶改为「置顶」徽章，
	// 正文 12px/1.7、时间 11px tabular；空态升级为带图标与说明的 EmptyState。
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

<div class="mt-6">
	<div class="flex items-end justify-between gap-3">
		<SectionHead title="群公告" count={announcements.length} class="mb-0 min-w-0 flex-1" />
		{#if canManage}
			<Button
				type="button"
				variant="soft"
				size="sm"
				class="mb-1 shrink-0"
				onclick={onCreate}
			>
				<Megaphone size={13} strokeWidth={1.75} />
				发公告
			</Button>
		{/if}
	</div>
	{#if announcements.length === 0}
		<EmptyState
			bare
			title="暂无公告"
			description={canManage ? '发布第一条群公告，让成员第一时间看到最新安排。' : '群主发布公告后会出现在这里。'}
			class="mt-3"
		>
			{#snippet icon()}
				<Megaphone class="size-7" strokeWidth={1.5} />
			{/snippet}
		</EmptyState>
	{:else}
		<!-- 空旷内容区 → List bordered（有边框列表契约） -->
		<ListContainer class="mt-3 divide-y divide-border-subtle/30">
			{#each announcements as item (item.id)}
				<ListItem class="relative flex flex-col items-start px-4 py-4" ariaLabel={item.title}>
					<div class="flex min-w-0 items-center gap-2 pr-16">
						{#if item.isPinned}
							<Badge tone="accent" class="shrink-0 gap-1">
								<Pin size={10} strokeWidth={2} />
								置顶
							</Badge>
						{/if}
						<p class="min-w-0 truncate text-md font-semibold text-text-primary">{item.title}</p>
					</div>
					<p class="mt-1.5 text-sm leading-card text-text-primary">{item.body}</p>
					<p class="mt-2 text-xs tabular-nums text-text-secondary">{item.time}</p>
					{#if canManage}
						<div class="absolute right-3 top-3 flex gap-1">
							<button
								type="button"
								onclick={() => onEdit(item)}
								class="hit-slop hover-btn scale-btn flex size-8 items-center justify-center rounded-full text-text-secondary focus-ring hover:bg-interactive-hover"
								aria-label="编辑公告"
							>
								<PencilLine size={13} strokeWidth={1.75} />
							</button>
							<button
								type="button"
								onclick={() => onDelete(item.id)}
								class="hit-slop hover-btn scale-btn flex size-8 items-center justify-center rounded-full text-status-danger focus-ring hover:bg-danger-bg"
								aria-label="删除公告"
							>
								<Trash2 size={13} strokeWidth={1.75} />
							</button>
						</div>
					{/if}
				</ListItem>
			{/each}
		</ListContainer>
	{/if}
</div>
