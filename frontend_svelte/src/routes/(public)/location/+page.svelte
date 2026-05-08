<script lang="ts">
	import LocationPage from '$lib/components/location/location-page.svelte';
</script>

<svelte:head>
	<title>位置选择 - HotSpot</title>
</svelte:head>

<div class="fixed inset-0 z-50" style="background: var(--neu-bg);">
	<LocationPage />
</div>
