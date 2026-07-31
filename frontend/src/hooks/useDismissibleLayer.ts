'use client'
import { useEffect } from "react";
import type React from "react";

type DismissibleLayerOptions = {
  open: boolean;
  onDismiss: () => void;
  rootRef?: React.RefObject<HTMLElement | null>;
  closeOnOutsidePress?: boolean;
};

/** Shared Escape / outside-press lifecycle for menus, dialogs and transient panels. */
export function useDismissibleLayer({
  open,
  onDismiss,
  rootRef,
  closeOnOutsidePress = false,
}: DismissibleLayerOptions) {
  useEffect(() => {
    if (!open) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") onDismiss();
    };
    const onPointerDown = (event: PointerEvent) => {
      if (closeOnOutsidePress && rootRef?.current && !rootRef.current.contains(event.target as Node)) onDismiss();
    };
    window.addEventListener("keydown", onKeyDown);
    if (closeOnOutsidePress) window.addEventListener("pointerdown", onPointerDown);
    return () => {
      window.removeEventListener("keydown", onKeyDown);
      if (closeOnOutsidePress) window.removeEventListener("pointerdown", onPointerDown);
    };
  }, [closeOnOutsidePress, onDismiss, open, rootRef]);
}
