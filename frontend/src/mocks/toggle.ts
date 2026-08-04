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
    await worker.start({
      onUnhandledRequest(req: { url: string }) {
        if (req.url.includes("/api/v1/")) return "warn";
        return "bypass";
      },
    });
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
