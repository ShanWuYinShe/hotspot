/**
 * 主题族状态管理
 * 管理多主题 family，明暗模式仍由 mode-watcher 负责。
 */

import { browser } from '$app/environment';
import type { TranslationKey } from '$lib/i18n';

export const THEME_FAMILIES = ['vibrant', 'forest', 'neu'] as const;

export type ThemeFamily = (typeof THEME_FAMILIES)[number];

export const THEME_FAMILY_OPTIONS: {
	family: ThemeFamily;
	labelKey: TranslationKey;
	descriptionKey: TranslationKey;
}[] = [
	{
		family: 'vibrant',
		labelKey: 'sidebar.vibrant',
		descriptionKey: 'sidebar.vibrantDesc'
	},
	{
		family: 'forest',
		labelKey: 'sidebar.forest',
		descriptionKey: 'sidebar.forestDesc'
	},
	{
		family: 'neu',
		labelKey: 'sidebar.neu',
		descriptionKey: 'sidebar.neuDesc'
	}
];

const THEME_FAMILY_STORAGE_KEY = 'hotspot_theme_family';
const DEFAULT_THEME_FAMILY: ThemeFamily = 'forest';

/** Map old theme names to current names */
const THEME_MIGRATION: Record<string, ThemeFamily> = {
	social_blue: 'vibrant',
	default_clean: 'forest',
	figma_neu: 'neu'
};

function isThemeFamily(value: string | null | undefined): value is ThemeFamily {
	return THEME_FAMILIES.some((themeFamily) => themeFamily === value);
}

function normalizeThemeFamily(value: string | null | undefined): ThemeFamily {
	if (isThemeFamily(value)) return value;
	if (value && value in THEME_MIGRATION) return THEME_MIGRATION[value as keyof typeof THEME_MIGRATION]!;
	return DEFAULT_THEME_FAMILY;
}

class ThemeStore {
	#themeFamily = $state<ThemeFamily>(DEFAULT_THEME_FAMILY);

	themeFamily = $derived(this.#themeFamily);

	init() {
		if (!browser) return;
		const stored = normalizeThemeFamily(localStorage.getItem(THEME_FAMILY_STORAGE_KEY));
		this.#themeFamily = stored;
		if (stored !== normalizeThemeFamily(null)) {
			localStorage.setItem(THEME_FAMILY_STORAGE_KEY, stored);
		}
		this.#apply(stored);
	}

	setThemeFamily(themeFamily: ThemeFamily) {
		if (this.#themeFamily === themeFamily) return;
		this.#themeFamily = themeFamily;
		if (!browser) return;
		localStorage.setItem(THEME_FAMILY_STORAGE_KEY, themeFamily);
		this.#apply(themeFamily);
	}

	#apply(themeFamily: ThemeFamily) {
		if (!browser) return;
		document.documentElement.setAttribute('data-theme-family', themeFamily);
	}
}

export const themeStore = new ThemeStore();
