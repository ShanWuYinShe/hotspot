'use client'
import React, { createContext, useContext, useState, useEffect } from "react";

export type Platform = "mobile" | "tablet" | "desktop";
export type Orientation = "portrait" | "landscape";

interface PlatformContextValue {
  platform: Platform;
  orientation: Orientation;
}

const PlatformContext = createContext<PlatformContextValue>({
  platform: "mobile",
  orientation: "portrait",
});

function detect(): PlatformContextValue {
  if (typeof window === "undefined") {
    return { platform: "mobile", orientation: "portrait" };
  }
  const w = window.innerWidth;
  const h = window.innerHeight;
  const minDim = Math.min(w, h);
  const orientation: Orientation = w > h ? "landscape" : "portrait";

  let platform: Platform;
  if (w >= 1280) platform = "desktop";
  else if (minDim >= 600) platform = "tablet";
  else platform = "mobile";

  return { platform, orientation };
}

export function PlatformProvider({ children }: { children: React.ReactNode }) {
  const [value, setValue] = useState<PlatformContextValue>(detect);

  useEffect(() => {
    let raf = 0;
    const handler = () => {
      cancelAnimationFrame(raf);
      raf = requestAnimationFrame(() => setValue(detect()));
    };
    window.addEventListener("resize", handler);
    window.addEventListener("orientationchange", handler);
    return () => {
      window.removeEventListener("resize", handler);
      window.removeEventListener("orientationchange", handler);
      cancelAnimationFrame(raf);
    };
  }, []);

  return (
    <PlatformContext.Provider value={value}>{children}</PlatformContext.Provider>
  );
}

export function usePlatform() {
  return useContext(PlatformContext);
}
