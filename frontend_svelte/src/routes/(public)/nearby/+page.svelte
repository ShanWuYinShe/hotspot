<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';

	onMount(() => {
		const params = new URLSearchParams($page.url.searchParams);
		const legacyTab = params.get('tab');
		if (legacyTab && ['moments', 'users', 'groups'].includes(legacyTab)) {
			params.set('nearby', legacyTab);
		}
		params.set('tab', 'nearby');
		goto(`/explore?${params.toString()}`, { replaceState: true });
	});
</script>

<svelte:head>
	<title>附近 - HotSpot</title>
</svelte:head>

<div class="flex min-h-[60vh] items-center justify-center text-sm text-muted-foreground">
	正在跳转到探索...
</div>
