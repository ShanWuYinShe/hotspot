import { createRoot } from "react-dom/client";
import App from "./app/App.tsx";
import "./styles/index.css";
import { setApiBaseUrl } from "./api";

const apiUrl = import.meta.env.VITE_API_URL ?? window.location.origin;
setApiBaseUrl(apiUrl);

async function startApp() {
  // Always initialize MSW in dev mode. Reads persisted preference from localStorage,
  // falls back to VITE_API_MOCK env. Toggling via AppearanceMenu saves to localStorage
  // and reloads the page so all data is re-fetched from the correct source.
  if (import.meta.env.DEV) {
    const { worker } = await import("./mocks/browser");
    const { initMock } = await import("./mocks/toggle");
    initMock(worker, import.meta.env.VITE_API_MOCK !== "false");
  }

  createRoot(document.getElementById("root")!).render(<App />);
}

startApp();
