import { describe, it, expect, beforeEach } from 'vitest';
import { headerIsland } from './header-island.svelte';

describe('headerIsland store', () => {
	beforeEach(() => {
		headerIsland.unregister();
	});

	it('initializes with inactive state', () => {
		expect(headerIsland.active).toBe(false);
		expect(headerIsland.scrolled).toBe(false);
		expect(headerIsland.category).toBe('');
		expect(headerIsland.entityName).toBe('');
	});

	it('registers header island configuration', () => {
		headerIsland.register({
			category: '社群',
			entityName: '测试群组',
			author: { name: '测试用户' }
		});

		expect(headerIsland.active).toBe(true);
		expect(headerIsland.category).toBe('社群');
		expect(headerIsland.entityName).toBe('测试群组');
		expect(headerIsland.author?.name).toBe('测试用户');
	});

	it('updates scroll state', () => {
		headerIsland.register({ entityName: '详情页' });
		expect(headerIsland.scrolled).toBe(false);

		headerIsland.setScrolled(true);
		expect(headerIsland.scrolled).toBe(true);

		headerIsland.setScrolled(false);
		expect(headerIsland.scrolled).toBe(false);
	});

	it('unregisters cleanly', () => {
		headerIsland.register({ entityName: '详情页' });
		headerIsland.setScrolled(true);
		headerIsland.unregister();

		expect(headerIsland.active).toBe(false);
		expect(headerIsland.scrolled).toBe(false);
		expect(headerIsland.entityName).toBe('');
	});
});
