import { describe, it, expect, vi } from 'vitest';
import { createCursorPagination } from '../cursor-pager.svelte';

interface Item {
	id: string;
	title: string;
}

describe('createCursorPagination', () => {
	it('loads the first page correctly', async () => {
		const mockFetch = vi.fn().mockResolvedValue({
			items: [
				{ id: '1', title: 'First' },
				{ id: '2', title: 'Second' }
			],
			nextCursor: 'cursor_2'
		});

		const pager = createCursorPagination<Item>({
			fetchPage: mockFetch,
			getKey: (i) => i.id
		});

		expect(pager.items).toEqual([]);
		expect(pager.loaded).toBe(false);
		expect(pager.loading).toBe(false);

		const p = pager.load();
		expect(pager.loading).toBe(true);
		await p;

		expect(pager.loading).toBe(false);
		expect(pager.loaded).toBe(true);
		expect(pager.items).toHaveLength(2);
		expect(pager.hasMore).toBe(true);
		expect(pager.nextCursor).toBe('cursor_2');
		expect(mockFetch).toHaveBeenCalledWith(null);
	});

	it('loads more items with deduplication', async () => {
		const mockFetch = vi
			.fn()
			.mockResolvedValueOnce({
				items: [{ id: '1', title: 'First' }],
				nextCursor: 'cursor_1'
			})
			.mockResolvedValueOnce({
				items: [
					{ id: '1', title: 'First Duplicate' },
					{ id: '2', title: 'Second' }
				],
				nextCursor: null
			});

		const pager = createCursorPagination<Item>({
			fetchPage: mockFetch,
			getKey: (i) => i.id
		});

		await pager.load();
		expect(pager.items).toHaveLength(1);
		expect(pager.hasMore).toBe(true);

		await pager.loadMore();
		expect(pager.items).toHaveLength(2);
		expect(pager.items.map((i) => i.id)).toEqual(['1', '2']);
		expect(pager.hasMore).toBe(false);
		expect(mockFetch).toHaveBeenCalledWith('cursor_1');
	});

	it('handles errors cleanly', async () => {
		const mockFetch = vi.fn().mockRejectedValue(new Error('Network disconnected'));

		const pager = createCursorPagination<Item>({
			fetchPage: mockFetch,
			getKey: (i) => i.id
		});

		await pager.load();
		expect(pager.loading).toBe(false);
		expect(pager.loaded).toBe(false);
		expect(pager.error).toBe('Network disconnected');
		expect(pager.items).toEqual([]);
	});

	it('cancels out-of-order responses on reset', async () => {
		let resolveFirst: ((val: any) => void) | null = null;
		const mockFetch = vi.fn().mockImplementation(() => {
			return new Promise((resolve) => {
				resolveFirst = resolve;
			});
		});

		const pager = createCursorPagination<Item>({
			fetchPage: mockFetch,
			getKey: (i) => i.id
		});

		void pager.load();
		expect(pager.loading).toBe(true);

		// Reset before first request resolves
		pager.reset();
		expect(pager.loading).toBe(false);
		expect(pager.items).toEqual([]);

		// First request resolves now
		resolveFirst!({
			items: [{ id: '1', title: 'Late arrival' }],
			nextCursor: 'c1'
		});

		// Give microtasks time to run
		await new Promise((r) => setTimeout(r, 0));

		// Items should still be empty because of requestId cancellation
		expect(pager.items).toEqual([]);
		expect(pager.loaded).toBe(false);
	});

	it('supports prepend, remove, and update mutations', async () => {
		const mockFetch = vi.fn().mockResolvedValue({
			items: [
				{ id: '1', title: 'One' },
				{ id: '2', title: 'Two' }
			],
			nextCursor: null
		});

		const pager = createCursorPagination<Item>({
			fetchPage: mockFetch,
			getKey: (i) => i.id
		});

		await pager.load();

		// Prepend
		pager.prepend({ id: '0', title: 'Zero' });
		expect(pager.items[0]).toEqual({ id: '0', title: 'Zero' });

		// Prepend duplicate key replaces & moves to front
		pager.prepend({ id: '2', title: 'Two Updated' });
		expect(pager.items.map((i) => i.id)).toEqual(['2', '0', '1']);

		// Update
		pager.update('0', (it) => ({ ...it, title: 'Zero Modified' }));
		expect(pager.items.find((i) => i.id === '0')?.title).toBe('Zero Modified');

		// Remove
		pager.remove('1');
		expect(pager.items.map((i) => i.id)).toEqual(['2', '0']);
	});
});
