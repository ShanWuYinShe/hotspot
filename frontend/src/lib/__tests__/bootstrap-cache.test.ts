// 冷启动 bootstrap 单次请求契约：并发/重复读取只发一次请求，refresh 才重拉。
// 用于防止首屏再出现「bootstrap + system/client_config」双份公共配置请求。
import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('@/api', () => ({ bootstrap: vi.fn() }));

import { bootstrap } from '@/api';
import { loadBootstrap, refreshBootstrap, resetBootstrapCache } from '../bootstrap-cache';

const mocked = vi.mocked(bootstrap);

function ok(data: unknown) {
	return { code: 0, data } as never;
}

beforeEach(() => {
	vi.clearAllMocks();
	resetBootstrapCache();
	mocked.mockResolvedValue(ok({ bootstrap_version: 'v1' }));
});

describe('bootstrap-cache', () => {
	it('并发调用共享同一次请求（in-flight 去重）', async () => {
		const [a, b] = await Promise.all([loadBootstrap(), loadBootstrap()]);
		expect(mocked).toHaveBeenCalledTimes(1);
		expect(a?.bootstrap_version).toBe('v1');
		expect(b).toBe(a);
	});

	it('命中缓存后不再发请求', async () => {
		await loadBootstrap();
		await loadBootstrap();
		expect(mocked).toHaveBeenCalledTimes(1);
	});

	it('refreshBootstrap 强制重拉并刷新缓存', async () => {
		await loadBootstrap();
		mocked.mockResolvedValue(ok({ bootstrap_version: 'v2' }));
		const fresh = await refreshBootstrap();
		expect(mocked).toHaveBeenCalledTimes(2);
		expect(fresh?.bootstrap_version).toBe('v2');
		// refresh 后普通读取命中新缓存
		expect((await loadBootstrap())?.bootstrap_version).toBe('v2');
		expect(mocked).toHaveBeenCalledTimes(2);
	});

	it('请求失败返回 null 且不抛错，失败后可重试', async () => {
		mocked.mockRejectedValueOnce(new Error('fetch failed'));
		await expect(loadBootstrap()).resolves.toBeNull();
		const retry = await loadBootstrap();
		expect(mocked).toHaveBeenCalledTimes(2);
		expect(retry?.bootstrap_version).toBe('v1');
	});
});
