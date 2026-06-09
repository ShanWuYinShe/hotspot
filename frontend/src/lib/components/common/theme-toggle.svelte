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
    class="flex items-center justify-center rounded-full outline-none transition-all duration-150 hover:scale-105 active:scale-95"
    style="width: 36px; height: 36px; background: var(--surface-bg); box-shadow: var(--shadow-md); border: 1px solid var(--border-theme);"
    onclick={toggleThemeMode}
    aria-label={t('sidebar.toggleTheme')}
  >
    {#if currentMode === 'dark'}
      <Sun size={16} strokeWidth={1.5} color="var(--text-primary)" />
    {:else}
      <Moon size={16} strokeWidth={1.5} color="var(--text-primary)" />
    {/if}
  </button>
  <button
    class="rounded-full px-3 py-1.5 outline-none transition-all duration-150 hover:scale-105 active:scale-95"
    style="font-size: 11px; font-weight: 500; background: var(--surface-bg); color: var(--text-primary); box-shadow: var(--shadow-md); border: 1px solid var(--border-theme);"
    onclick={toggleThemeFamily}
  >
    {getThemeFamilyLabel(currentFamily)}
  </button>
</div>
