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

/** Called once at startup from main.tsx. Reads localStorage, falls back to env. */
export function initMock(w: any, envDefault: boolean) {
  worker = w;
  enabled = readPersisted(envDefault);
  if (enabled) {
    // "warn" surfaces unhandled requests so missing handlers are visible during
    // dev instead of silently falling through to the real network.
    worker.start({ onUnhandledRequest: "warn" });
  }
  notify();
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
