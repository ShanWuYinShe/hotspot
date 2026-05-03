import { browser } from '$app/environment';

export type Platform = 'mobile' | 'tablet' | 'desktop';
export type Orientation = 'portrait' | 'landscape';

class PlatformStore {
	platform = $state<Platform>('mobile');
	orientation = $state<Orientation>('portrait');
	isMobile = $state(true);
	isTablet = $state(false);
	isDesktop = $state(false);

	#initialized = false;

	constructor() {
		if (!browser) return;
		this.update();
		window.addEventListener('resize', () => {
			requestAnimationFrame(() => this.update());
		});
		window.addEventListener('orientationchange', () => {
			requestAnimationFrame(() => this.update());
		});
	}

	private update() {
		const w = window.innerWidth;
		let p: Platform;
		if (w >= 1280) p = 'desktop';
		else if (w >= 600) p = 'tablet';
		else p = 'mobile';

		this.platform = p;
		this.orientation = w > window.innerHeight ? 'landscape' : 'portrait';
		this.isMobile = p === 'mobile';
		this.isTablet = p === 'tablet';
		this.isDesktop = p === 'desktop';
	}

	init() {
		if (this.#initialized) return;
		this.#initialized = true;
		this.update();
	}
}

export const platformStore = new PlatformStore();
