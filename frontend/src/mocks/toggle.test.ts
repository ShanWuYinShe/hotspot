// initMock 启动超时：worker.start() 悬挂时不能按住首屏（hooks.client init 在等它）
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { initMock, MOCK_START_TIMEOUT_MS } from './toggle';

const KEY = 'hotspot:mock-enabled';

function fakeWorker(impl: { start?: (...args: never[]) => unknown; stop?: () => unknown }) {
  return { start: vi.fn().mockResolvedValue(undefined), stop: vi.fn().mockResolvedValue(undefined), ...impl };
}

beforeEach(() => {
  vi.useFakeTimers();
  localStorage.clear();
});

afterEach(() => {
  vi.useRealTimers();
  localStorage.clear();
});

describe('initMock', () => {
  it('正常注册不受超时影响', async () => {
    localStorage.setItem(KEY, 'true');
    const w = fakeWorker({});
    await expect(initMock(w, true)).resolves.toBe(true);
    expect(w.start).toHaveBeenCalledTimes(1);
    expect(w.stop).not.toHaveBeenCalled();
  });

  it('start 悬挂时超时抛错并尝试停掉 worker（防迟到接管）', async () => {
    localStorage.setItem(KEY, 'true');
    const w = fakeWorker({ start: () => new Promise(() => {}) });
    const p = initMock(w, true);
    // 先挂载拒绝断言再推时钟：否则定时器触发的拒绝在 expect 订阅前处于无 handler 状态。
    const assertion = expect(p).rejects.toThrow('mock worker start timeout');
    await vi.advanceTimersByTimeAsync(MOCK_START_TIMEOUT_MS);
    await assertion;
    expect(w.stop).toHaveBeenCalledTimes(1);
  });

  it('开关关闭时不启动 worker', async () => {
    localStorage.setItem(KEY, 'false');
    const w = fakeWorker({});
    await expect(initMock(w, true)).resolves.toBe(false);
    expect(w.start).not.toHaveBeenCalled();
  });
});
