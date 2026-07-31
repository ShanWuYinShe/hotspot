'use client'
import React, { createContext, useContext, useMemo, useState } from "react";

type ComposerContextValue = {
  composerOpen: boolean;
  openComposer: () => void;
  closeComposer: () => void;
  toggleComposer: () => void;
};

const ComposerContext = createContext<ComposerContextValue | null>(null);

/** One publishing entry point for every platform shell and in-page call to action. */
export function ComposerProvider({ children }: { children: React.ReactNode }) {
  const [composerOpen, setComposerOpen] = useState(false);
  const value = useMemo(() => ({
    composerOpen,
    openComposer: () => setComposerOpen(true),
    closeComposer: () => setComposerOpen(false),
    toggleComposer: () => setComposerOpen((open) => !open),
  }), [composerOpen]);
  return <ComposerContext.Provider value={value}>{children}</ComposerContext.Provider>;
}

export function useComposer() {
  const context = useContext(ComposerContext);
  if (!context) {
    if (typeof window === 'undefined') {
      return { composerOpen: false, openComposer: () => {}, closeComposer: () => {}, toggleComposer: () => {} };
    }
    throw new Error("useComposer must be used within ComposerProvider");
  }
  return context;
}
