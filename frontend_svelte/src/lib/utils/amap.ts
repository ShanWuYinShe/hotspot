const AMAP_KEY = import.meta.env.VITE_AMAP_KEY;
const AMAP_SERVICE_HOST = import.meta.env.VITE_AMAP_SERVICE_HOST || '/_AMapService';

let loadPromise: Promise<void> | null = null;
let securityConfigured = false;

function configureSecurity() {
	if (securityConfigured || typeof window === 'undefined') return;
	window._AMapSecurityConfig = { serviceHost: AMAP_SERVICE_HOST };
	securityConfigured = true;
}

export function loadAMap(): Promise<void> {
	if (typeof window === 'undefined') {
		return Promise.reject(new Error('AMap only available in browser'));
	}
	if (window.AMap) return Promise.resolve();
	if (loadPromise) return loadPromise;

	if (!AMAP_KEY) {
		return Promise.reject(new Error('AMap key is not configured'));
	}

	configureSecurity();

	loadPromise = new Promise((resolve, reject) => {
		const existing = document.getElementById('amap-sdk-script') as HTMLScriptElement | null;
		if (existing) {
			existing.addEventListener('load', () => resolve());
			existing.addEventListener('error', () => reject(new Error('AMap SDK load failed')));
			return;
		}

		const script = document.createElement('script');
		script.id = 'amap-sdk-script';
		script.src = `https://webapi.amap.com/maps?v=2.0&key=${AMAP_KEY}`;
		script.async = true;
		script.onload = () => resolve();
		script.onerror = () => reject(new Error('AMap SDK load failed'));
		document.head.appendChild(script);
	});

	return loadPromise;
}

export function loadAMapPlugin(plugins: string[]): Promise<void> {
	return new Promise((resolve, reject) => {
		loadAMap()
			.then(() => {
				if (!window.AMap) {
					reject(new Error('AMap SDK not available'));
					return;
				}
				window.AMap.plugin(plugins, () => resolve());
			})
			.catch(reject);
	});
}
