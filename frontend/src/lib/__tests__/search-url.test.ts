import { describe, expect, it } from 'vitest';
import { buildSearchUrl, parseSearchUrl } from '../search-url';

describe('parseSearchUrl', () => {
	it('解析 query/tab/tag', () => {
		expect(parseSearchUrl('?query=%E5%92%96%E5%95%A1&tab=posts&tag=%E7%BE%8E%E9%A3%9F')).toEqual({
			query: '咖啡',
			tab: 'posts',
			tag: '美食'
		});
	});

	it('tab 别名 users/moments 归一化，非法值回退 all', () => {
		expect(parseSearchUrl('?query=a&tab=users').tab).toBe('people');
		expect(parseSearchUrl('?query=a&tab=moments').tab).toBe('posts');
		expect(parseSearchUrl('?query=a&tab=nope').tab).toBe('all');
	});

	it('空 query 不保留 tag、返回发现态', () => {
		expect(parseSearchUrl('?tag=x')).toEqual({ query: '', tab: 'all', tag: null });
	});

	it('无参/空串安全', () => {
		expect(parseSearchUrl('')).toEqual({ query: '', tab: 'all', tag: null });
	});
});

describe('buildSearchUrl', () => {
	it('有 query 时保留 tab/tag', () => {
		expect(buildSearchUrl({ query: '咖啡', tab: 'posts', tag: '美食' })).toBe(
			'/search?query=%E5%92%96%E5%95%A1&tag=%E7%BE%8E%E9%A3%9F&tab=posts'
		);
	});
	it('all 不写 tab；空 query 回发现态', () => {
		expect(buildSearchUrl({ query: 'a', tab: 'all', tag: null })).toBe('/search?query=a');
		expect(buildSearchUrl({ query: '', tab: 'posts', tag: null })).toBe('/search');
	});
});
