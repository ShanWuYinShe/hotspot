import { createRoot } from "react-dom/client";
import App from "./app/App.tsx";
import { initApiClient } from "./lib/api/client";
import "./styles/index.css";

initApiClient({
  baseUrl: '',
  onRefreshSuccess: () => {
    console.debug('[API] Token refreshed');
  },
  onRefreshFail: () => {
    console.warn('[API] Token refresh failed');
    window.dispatchEvent(new CustomEvent('auth:session-expired'));
  },
  onAccountBanned: (message: string) => {
    window.dispatchEvent(new CustomEvent('auth:account-banned', { detail: { message } }));
  }
});

createRoot(document.getElementById("root")!).render(<App />);

if ("serviceWorker" in navigator && import.meta.env.PROD) {
  window.addEventListener("load", () => {
    navigator.serviceWorker.register("/sw.js");
  });
}
