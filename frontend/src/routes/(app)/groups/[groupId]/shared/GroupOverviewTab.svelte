<script lang="ts">
	import { ChevronRight, Megaphone, Pin } from '@lucide/svelte';
	import { formatCount } from '@/lib/utils';
	import MetricRow from '@/components/data-display/MetricRow.svelte';
	import Badge from '@/components/data-display/Badge.svelte';
	import SectionHead from '@/components/home/SectionHead.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';

	interface AnnouncementSummary {
		id: string;
		title: string;
		body: string;
		isPinned: boolean;
	}

	interface MemberSummary {
		userId?: string;
		avatar?: string;
		name?: string;
	}

	let {
		detail,
		members,
		announcements,
		onViewAllMembers,
		onViewAnnouncements
	}: {
		detail: {
			description?: string;
			members?: number;
			online?: number;
			distance?: string;
		};
		members: MemberSummary[];
		announcements: AnnouncementSummary[];
		onViewAllMembers: () => void;
		onViewAnnouncements: () => void;
	} = $props();
</script>

<div class="mt-6 flex flex-col gap-6">
	<section class="flex flex-col gap-6" aria-label="群组概览">
		<p class="max-w-prose text-md leading-copy text-text-primary whitespace-pre-wrap break-words">
			{detail.description || '这个群还没有填写简介。'}
		</p>

		{#snippet onlineValue()}
			<span class="inline-flex items-center gap-1.5 tabular-nums">
				{detail.online ?? 0}
				<span class="size-1.5 rounded-full bg-status-online" aria-hidden="true"></span>
			</span>
		{/snippet}

		<MetricRow
			variant="strip"
			items={[
				{ label: '群成员', value: formatCount(detail.members ?? 0) },
				{ label: '在线', value: onlineValue, tone: 'positive' },
				{ label: '距你', value: detail.distance || '—' },
				{ label: '公告', value: announcements.length }
			]}
		/>

		{#if members.length > 0}
			<div>
				<SectionHead
					title="活跃成员"
					count={formatCount(detail.members ?? 0)}
					action="查看全部"
					onAction={onViewAllMembers}
				/>
				<div class="flex items-center gap-3">
					<div class="flex shrink-0 -space-x-2">
						{#each members.slice(0, 6) as m, i (m.userId ?? i)}
							<Avatar src={m.avatar} name={m.name} size="md" decorative class="rounded-full border-2 border-surface-page shadow-xs" />
						{/each}
					</div>
					<p class="min-w-0 truncate text-xs text-text-secondary tabular-nums">{formatCount(detail.members ?? 0)} 位成员在群中</p>
				</div>
			</div>
		{/if}

		{#if announcements.length > 0}
			{@const latest = announcements[0]}
			<div>
				<SectionHead
					title="群公告"
					count={announcements.length}
				/>
				<button
					type="button"
					class="hover-btn flex w-full items-start gap-3 rounded-lg bg-surface-subtle/50 p-3.5 text-left transition-colors focus-ring hover:bg-surface-subtle"
					onclick={onViewAnnouncements}
				>
					<span class="flex size-9 shrink-0 items-center justify-center rounded-full bg-accent-soft text-accent-default">
						<Megaphone size={16} strokeWidth={1.75} />
					</span>
					<span class="min-w-0 flex-1">
						<span class="flex min-w-0 items-center gap-2">
							{#if latest.isPinned}
								<Badge tone="accent" class="shrink-0 gap-1">
									<Pin size={10} strokeWidth={2} />
									置顶
								</Badge>
							{/if}
							<span class="truncate text-md font-semibold text-text-primary">{latest.title}</span>
						</span>
						<span class="mt-1 block truncate text-sm text-text-secondary">{latest.body}</span>
					</span>
					<ChevronRight size={14} class="mt-1 shrink-0 text-text-secondary" strokeWidth={1.5} />
				</button>
			</div>
		{/if}
	</section>
</div>
