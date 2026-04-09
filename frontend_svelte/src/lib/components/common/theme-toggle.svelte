<script lang="ts">
	import { mode, setMode } from 'mode-watcher';
	import { Button } from '$lib/components/ui/button';
	import { THEME_FAMILY_OPTIONS, themeStore, type ThemeFamily } from '$lib/stores/theme.svelte';
	import { t } from '$lib/i18n';
	import Sun from 'lucide-svelte/icons/sun';
	import Moon from 'lucide-svelte/icons/moon';

	const currentMode = $derived(mode.current);
	const currentFamily = $derived(themeStore.themeFamily);

	function toggleThemeMode() {
		setMode(currentMode === 'dark' ? 'light' : 'dark');
	}

	const themeFamilyCycle: ThemeFamily[] = THEME_FAMILY_OPTIONS.map(({ family }) => family);

	function getThemeFamilyLabel(themeFamily: ThemeFamily) {
		const option = THEME_FAMILY_OPTIONS.find((item) => item.family === themeFamily);
		return option ? t(option.labelKey) : t('sidebar.defaultClean');
	}

	function toggleThemeFamily() {
		const currentIndex = themeFamilyCycle.indexOf(currentFamily);
		const nextFamily = themeFamilyCycle[(currentIndex + 1) % themeFamilyCycle.length] ?? 'default_clean';
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
		{getThemeFamilyLabel(currentFamily)}
	</Button>
</div>
