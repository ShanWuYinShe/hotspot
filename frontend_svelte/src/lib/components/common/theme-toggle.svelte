<script lang="ts">
	import { mode, setMode } from 'mode-watcher';
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
		return option ? t(option.labelKey) : t('sidebar.forest');
	}

	function toggleThemeFamily() {
		const currentIndex = themeFamilyCycle.indexOf(currentFamily);
		const nextFamily = themeFamilyCycle[(currentIndex + 1) % themeFamilyCycle.length] ?? 'forest';
		themeStore.setThemeFamily(nextFamily);
	}
</script>

<div class="flex items-center gap-2">
	<button
		class="flex items-center justify-center rounded-full outline-none"
		style="width: 36px; height: 36px; background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border);"
		onclick={toggleThemeMode}
		aria-label={t('sidebar.toggleTheme')}
	>
		{#if currentMode === 'dark'}
			<Sun size={16} strokeWidth={1.5} color="var(--neu-text)" />
		{:else}
			<Moon size={16} strokeWidth={1.5} color="var(--neu-text)" />
		{/if}
	</button>
	<button
		class="rounded-full px-3 py-1.5 outline-none"
		style="font-size: 11px; font-weight: 500; background: var(--neu-bg); color: var(--neu-text); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border);"
		onclick={toggleThemeFamily}
	>
		{getThemeFamilyLabel(currentFamily)}
	</button>
</div>
