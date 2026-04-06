<script lang="ts">
	import { mode, setMode } from 'mode-watcher';
	import { Button } from '$lib/components/ui/button';
	import { themeStore, type ThemeFamily } from '$lib/stores/theme.svelte';
	import { t } from '$lib/i18n';
	import Sun from 'lucide-svelte/icons/sun';
	import Moon from 'lucide-svelte/icons/moon';

	const currentMode = $derived(mode.current);
	const currentFamily = $derived(themeStore.themeFamily);

	function toggleThemeMode() {
		setMode(currentMode === 'dark' ? 'light' : 'dark');
	}

	function toggleThemeFamily() {
		const nextFamily: ThemeFamily = currentFamily === 'default_clean' ? 'figma_neu' : 'default_clean';
		themeStore.setThemeFamily(nextFamily);
	}
</script>

<div class="flex items-center gap-2">
	<Button variant="ghost" size="icon" onclick={toggleThemeMode} aria-label={t('sidebar.toggleTheme')}>
		{#if currentMode === 'dark'}
			<Sun class="h-5 w-5" />
		{:else}
			<Moon class="h-5 w-5" />
		{/if}
		<span class="sr-only">{t('sidebar.toggleTheme')}</span>
	</Button>
	<Button variant="outline" size="sm" class="rounded-[1rem] px-3" onclick={toggleThemeFamily}>
		{currentFamily === 'default_clean' ? t('sidebar.defaultClean') : t('sidebar.figmaNeu')}
	</Button>
</div>
