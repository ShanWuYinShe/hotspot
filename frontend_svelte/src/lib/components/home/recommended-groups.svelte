<script lang="ts">
	import { dev } from '$app/environment';
	import { groupsService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
	import type { GroupItem } from '$lib/types/api';
	import GroupCard from '$lib/components/common/group-card.svelte';
	import ListSkeleton from '$lib/components/common/list-skeleton.svelte';
	import Users from 'lucide-svelte/icons/users';

	interface Props {
		refreshTrigger?: number;
	}

	let { refreshTrigger = 0 }: Props = $props();

	let groups = $state.raw<GroupItem[]>([]);
	let loading = $state(false);
	let loadVersion = 0;

	async function loadGroups() {
		const version = ++loadVersion;
		loading = true;
		try {
			const response = await groupsService.search({
				radius: mapAnchorStore.radius * 1000,
				reference_point: {
					latitude: mapAnchorStore.latitude,
					longitude: mapAnchorStore.longitude
				},
				pagination: { limit: 6 }
			});
			if (version !== loadVersion) return; // 丢弃过期响应
			if (isSuccess(response) && response.data) {
				groups = response.data.items ?? [];
			}
		} catch (error) {
			if (version !== loadVersion) return;
			if (dev) console.error('Failed to load recommended groups:', error);
		} finally {
			if (version === loadVersion) loading = false;
		}
	}

	$effect(() => {
		refreshTrigger;
		mapAnchorStore.anchor.updatedAt; // Dependency
		loadGroups();
	});
</script>

{#if loading}
	<section class="space-y-4">
		<div class="flex items-center justify-between">
			<div class="flex items-center gap-2 px-1">
				<Users class="h-5 w-5 text-primary" />
				<h2 class="text-lg font-semibold tracking-tight">推荐群组</h2>
			</div>
		</div>
		<div class="columns-1 gap-3 sm:columns-2 lg:columns-3 xl:columns-4">
			{#each Array(6) as _, i (i)}
				<div class="mb-3 break-inside-avoid">
					<ListSkeleton type="group" count={1} layout="list" />
				</div>
			{/each}
		</div>
	</section>
{:else if groups.length > 0}
	<section class="space-y-4">
		<div class="flex items-center justify-between">
			<div class="flex items-center gap-2 px-1">
				<Users class="h-5 w-5 text-primary" />
				<h2 class="text-lg font-semibold tracking-tight">推荐群组</h2>
			</div>
			<a href="/groups/nearby" class="text-sm font-medium text-muted-foreground hover:underline">
				查看更多
			</a>
		</div>
		<div class="columns-1 gap-3 sm:columns-2 lg:columns-3 xl:columns-4">
			{#each groups as group (group.group.group_id)}
				<div class="mb-3 break-inside-avoid">
					<GroupCard {group} showDistance showMembers variant="square" />
				</div>
			{/each}
		</div>
	</section>
{/if}
