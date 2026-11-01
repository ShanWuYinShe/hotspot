import type { Snippet } from 'svelte';

export interface AuthorInfo {
	id?: string;
	name: string;
	avatar?: string | null;
	location?: string | null;
	time?: string | null;
}

export interface HeaderIslandConfig {
	category?: string;
	entityName?: string;
	author?: AuthorInfo;
	onBack?: () => void;
	backLabel?: string;
	onAuthorClick?: () => void;
	trailing?: Snippet;
}

class HeaderIslandStore {
	active = $state(false);
	scrolled = $state(false);
	category = $state<string>('');
	entityName = $state<string>('');
	author = $state<AuthorInfo | undefined>(undefined);
	onBack = $state<(() => void) | undefined>(undefined);
	backLabel = $state<string>('返回');
	onAuthorClick = $state<(() => void) | undefined>(undefined);
	trailing = $state<Snippet | undefined>(undefined);

	register(config: HeaderIslandConfig) {
		this.category = config.category ?? '';
		this.entityName = config.entityName ?? '';
		this.author = config.author;
		this.onBack = config.onBack;
		this.backLabel = config.backLabel ?? '返回';
		this.onAuthorClick = config.onAuthorClick;
		this.trailing = config.trailing;
		this.active = true;
	}

	update(config: Partial<HeaderIslandConfig>) {
		if (config.category !== undefined) this.category = config.category;
		if (config.entityName !== undefined) this.entityName = config.entityName;
		if (config.author !== undefined) this.author = config.author;
		if (config.onBack !== undefined) this.onBack = config.onBack;
		if (config.backLabel !== undefined) this.backLabel = config.backLabel;
		if (config.onAuthorClick !== undefined) this.onAuthorClick = config.onAuthorClick;
		if (config.trailing !== undefined) this.trailing = config.trailing;
	}

	setScrolled(isScrolled: boolean) {
		this.scrolled = isScrolled;
	}

	unregister() {
		this.active = false;
		this.scrolled = false;
		this.category = '';
		this.entityName = '';
		this.author = undefined;
		this.onBack = undefined;
		this.backLabel = '返回';
		this.onAuthorClick = undefined;
		this.trailing = undefined;
	}
}

export const headerIsland = new HeaderIslandStore();
