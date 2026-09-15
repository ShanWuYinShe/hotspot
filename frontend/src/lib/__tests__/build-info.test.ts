import { describe, expect, it, vi } from 'vitest';
import { createBuildInfo } from '../build-info.svelte';

const JSON_RESPONSE = (payload: unknown, ok = true, status = ok ? 200 : 404) =>
	({
		ok,
		status,
		json: async () => payload
	}) as Response;

describe('createBuildInfo', () => {
	it('成功读取 /version.json 后暴露版本指纹', async () => {
		const payload = {
			project: 'hotspot-frontend',
			version: 'v1.1.0',
			commit: 'a'.repeat(40),
			build_time: '2026-09-16T00:00:00Z'
		};
		const store = createBuildInfo(async () => payload);
		await store.load();
		expect(store.info).toEqual(payload);
	});

	it('请求失败兜底 dev（不抛出、不阻塞页面）', async () => {
		const store = createBuildInfo(async () => {
			throw new Error('network down');
		});
		await store.load();
		expect(store.info).toEqual({ version: 'dev', commit: '', build_time: '' });
	});

	it('非 200（如本地 dev 未生成 version.json）同样兜底 dev', async () => {
		vi.stubGlobal(
			'fetch',
			vi.fn(async () => JSON_RESPONSE({ message: 'not found' }, false, 404))
		);
		const store = createBuildInfo();
		await store.load();
		expect(store.info?.version).toBe('dev');
		vi.unstubAllGlobals();
	});

	it('响应缺 version 字段视为无效，兜底 dev', async () => {
		const store = createBuildInfo(async () => ({ unexpected: true } as never));
		await store.load();
		expect(store.info?.version).toBe('dev');
	});

	it('幂等：并发调用共享同一次请求', async () => {
		const fetchJson = vi.fn(async () => ({ version: 'v1.1.0', commit: '', build_time: '' }));
		const store = createBuildInfo(fetchJson);
		await Promise.all([store.load(), store.load(), store.load()]);
		expect(fetchJson).toHaveBeenCalledTimes(1);
		expect(store.info?.version).toBe('v1.1.0');
	});
});
