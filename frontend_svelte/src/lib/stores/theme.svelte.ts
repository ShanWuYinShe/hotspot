/**
 * 主题族状态管理
 * 管理多主题 family，明暗模式仍由 mode-watcher 负责。
 */

import { browser } from '$app/environment';
import type { TranslationKey } from '$lib/i18n';

export const THEME_FAMILIES = ['social_blue', 'default_clean', 'figma_neu'] as const;

export type ThemeFamily = (typeof THEME_FAMILIES)[number];

export const THEME_FAMILY_OPTIONS: {
	family: ThemeFamily;
	labelKey: TranslationKey;
	descriptionKey: TranslationKey;
}[] = [
	{
		family: 'social_blue',
		labelKey: 'sidebar.socialBlue',
		descriptionKey: 'sidebar.socialBlueDesc'
	},
	{
		family: 'default_clean',
		labelKey: 'sidebar.defaultClean',
		descriptionKey: 'sidebar.defaultCleanDesc'
	},
	{
		family: 'figma_neu',
		labelKey: 'sidebar.figmaNeu',
		descriptionKey: 'sidebar.figmaNeuDesc'
	}
];

const THEME_FAMILY_STORAGE_KEY = 'hotspot_theme_family';
const DEFAULT_THEME_FAMILY: ThemeFamily = 'default_clean';

function isThemeFamily(value: string | null | undefined): value is ThemeFamily {
	return THEME_FAMILIES.some((themeFamily) => themeFamily === value);
}

function normalizeThemeFamily(value: string | null | undefined): ThemeFamily {
	return isThemeFamily(value) ? value : DEFAULT_THEME_FAMILY;
}

class ThemeStore {
	#themeFamily = $state<ThemeFamily>(DEFAULT_THEME_FAMILY);

	themeFamily = $derived(this.#themeFamily);

	init() {
		if (!browser) return;
		const stored = normalizeThemeFamily(localStorage.getItem(THEME_FAMILY_STORAGE_KEY));
		this.#themeFamily = stored;
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
