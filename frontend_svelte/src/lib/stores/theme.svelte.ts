/**
 * 主题族状态管理
 * 管理多主题 family，明暗模式仍由 mode-watcher 负责。
 */

import { browser } from '$app/environment';

export type ThemeFamily = 'default_clean' | 'figma_neu';

const THEME_FAMILY_STORAGE_KEY = 'hotspot_theme_family';
const DEFAULT_THEME_FAMILY: ThemeFamily = 'default_clean';

function normalizeThemeFamily(value: string | null | undefined): ThemeFamily {
	return value === 'figma_neu' ? 'figma_neu' : DEFAULT_THEME_FAMILY;
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
