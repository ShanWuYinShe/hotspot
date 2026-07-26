import { createRoot } from "react-dom/client";
import App from "./app/App.tsx";
import "./styles/index.css";
import { setApiBaseUrl } from "./api";

const apiUrl = import.meta.env.VITE_API_URL ?? window.location.origin;
setApiBaseUrl(apiUrl);

async function startApp() {
  // Enable MSW mock API in dev mode. Set VITE_API_MOCK=false to use real backend.
  if (import.meta.env.DEV && import.meta.env.VITE_API_MOCK !== "false") {
    const { worker } = await import("./mocks/browser");
    await worker.start({ onUnhandledRequest: "bypass" });
  }

  createRoot(document.getElementById("root")!).render(<App />);
}

startApp();
