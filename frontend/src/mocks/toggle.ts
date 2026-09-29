type Listener = () => void;

let worker: any = null;
let enabled = false;
const listeners = new Set<Listener>();
const STORAGE_KEY = "hotspot:mock-enabled";

function readPersisted(fallback: boolean): boolean {
  try {
    const v = localStorage.getItem(STORAGE_KEY);
    if (v !== null) return v === "true";
  } catch {}
  return fallback;
}

function writePersisted(v: boolean) {
  try { localStorage.setItem(STORAGE_KEY, String(v)); } catch {}
}

/**
 * worker.start() 最长等待：service worker 注册在异常环境（隐私模式/无痕
 * 禁用、headless 嵌套沙盒等）下可能既不成功也不失败，hooks.client init 会
 * 被它按住导致首屏白屏。超时后直接走既有降级（直连真实后端），与 catch
 * 分支行为一致；正常注册远快于此时限，不受影响。
 */
export const MOCK_START_TIMEOUT_MS = 8000;

/**
 * 应用启动时调用一次（hooks.client init，早于任何 API 请求）。
 * 读取 localStorage（可覆盖默认），启用时等待 service worker 真正就绪。
 * 返回是否启用了 mock。
 */
export async function initMock(w: any, envDefault: boolean): Promise<boolean> {
  worker = w;
  enabled = readPersisted(envDefault);
  if (enabled) {
    // Only warn about genuinely missing API handlers. SPA navigation requests
    // (e.g. GET /me/groups, GET /me/relationships — the page routes, not API
    // calls) and Vite HMR / font / static asset requests would otherwise each
    // log a noisy "[MSW] intercepted a request without a matching request
    // handler" warning on every navigation, drowning out real gaps.
    let timer: ReturnType<typeof setTimeout> | undefined;
    try {
      await Promise.race([
        worker.start({
          onUnhandledRequest(req: { url: string }) {
            if (req.url.includes("/api/v1/")) return "warn";
            return "bypass";
          },
        }),
        new Promise<never>((_, reject) => {
          timer = setTimeout(() => reject(new Error("mock worker start timeout")), MOCK_START_TIMEOUT_MS);
        }),
      ]);
    } catch (err) {
      // 超时后先尝试停掉 worker：迟到的注册成功会在首屏请求发出后中途接管，
      // 造成 mock/真实双数据源混用；停掉后会话全程走真实后端，来源一致。
      try {
        await worker.stop?.();
      } catch {
        // 停止失败忽略：调用方 catch 同样回落直连。
      }
      throw err;
    } finally {
      clearTimeout(timer);
    }
  }
  notify();
  return enabled;
}

export function isMockEnabled(): boolean {
  return enabled;
}

export function subscribe(listener: Listener): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

function notify() {
  listeners.forEach((l) => l());
}

// Persist + reload so every page re-fetches from the correct data source.
export async function toggleMock() {
  if (!worker) return;
  enabled = !enabled;
  writePersisted(enabled);
  window.location.reload();
}
