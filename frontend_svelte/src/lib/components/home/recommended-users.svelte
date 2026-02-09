<script lang="ts">
	import { dev } from '$app/environment';
	import { usersService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
	import type { UserItem } from '$lib/types/api';
	import UserCard from '$lib/components/common/user-card.svelte';
	import ListSkeleton from '$lib/components/common/list-skeleton.svelte';
	import Users from 'lucide-svelte/icons/users';

	interface Props {
		refreshTrigger?: number;
	}

	let { refreshTrigger = 0 }: Props = $props();

	let users = $state.raw<UserItem[]>([]);
	let loading = $state(false);

	async function loadUsers() {
		loading = true;
		try {
			const response = await usersService.search({
				search: {
					reference_point: {
						latitude: mapAnchorStore.latitude,
						longitude: mapAnchorStore.longitude
					},
					pagination: {
						limit: 6
					}
				},
				radius: mapAnchorStore.radius * 1000
			});
			if (isSuccess(response) && response.data) {
				users = response.data.items ?? [];
			}
		} catch (error) {
			if (dev) console.error('Failed to load recommended users:', error);
			// 不显示 toast，避免影响首页体验
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		refreshTrigger;
		mapAnchorStore.anchor.updatedAt; // Dependency
		loadUsers();
	});
</script>

{#if loading}
	<section class="space-y-4">
		<div class="flex items-center justify-between">
			<div class="flex items-center gap-2 px-1">
				<Users class="h-5 w-5 text-primary" />
				<h2 class="text-lg font-semibold tracking-tight">推荐用户</h2>
			</div>
		</div>
		<div class="columns-1 gap-3 sm:columns-2 lg:columns-3 xl:columns-4">
			{#each Array(6) as _}
				<div class="mb-3 break-inside-avoid">
					<ListSkeleton type="user" count={1} layout="list" />
				</div>
			{/each}
		</div>
	</section>
{:else if users.length > 0}
	<section class="space-y-4">
		<div class="flex items-center justify-between">
			<div class="flex items-center gap-2 px-1">
				<Users class="h-5 w-5 text-primary" />
				<h2 class="text-lg font-semibold tracking-tight">推荐用户</h2>
			</div>
			<a href="/users/nearby" class="text-sm font-medium text-muted-foreground hover:underline">
				查看更多
			</a>
		</div>
		<div class="columns-1 gap-3 sm:columns-2 lg:columns-3 xl:columns-4">
			{#each users as user (user.user.user_id)}
				<div class="mb-3 break-inside-avoid">
					<UserCard {user} showDistance showBio showTags variant="square" />
				</div>
			{/each}
		</div>
	</section>
{/if}
