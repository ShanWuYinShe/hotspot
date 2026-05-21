import { useEffect, useRef, useCallback } from "react";

export function useNetworkStatus(onChange?: (online: boolean) => void) {
  const onlineRef = useRef(navigator.onLine);
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;

  const update = useCallback((online: boolean) => {
    if (onlineRef.current !== online) {
      onlineRef.current = online;
      onChangeRef.current?.(online);
    }
  }, []);

  useEffect(() => {
    const handleOnline = () => update(true);
    const handleOffline = () => update(false);

    window.addEventListener("online", handleOnline);
    window.addEventListener("offline", handleOffline);

    return () => {
      window.removeEventListener("online", handleOnline);
      window.removeEventListener("offline", handleOffline);
    };
  }, [update]);
}
