'use client'
import React, { useState } from "react";

type AvatarProps = {
  /** Image URL. When empty/undefined or the image fails to load, the name initial is shown instead. */
  src?: string;
  /** Display name — supplies the alt text and the fallback initial. */
  name?: string;
  /** Square dimension in px. */
  size?: number;
  /** Circle (people) or rounded rectangle (groups). */
  shape?: "circle" | "rounded";
  /** Draw a thin themed ring around the avatar. The ring shares one element with
   *  the border-radius so the outline stays continuous (the broken-border bug
   *  came from layering a bordered online-dot onto a separate rounded <img>). */
  ring?: boolean;
  /** Optional badge (e.g. online dot) rendered absolutely positioned inside the
   *  wrapper, so it sits on the same base ring as the outline. */
  badge?: React.ReactNode;
  className?: string;
  style?: React.CSSProperties;
};

/**
 * Avatar — single source of truth for showing a user/group avatar with a
 * graceful fallback. Replaces the scattered `avatar_url ? <img/> : <span>首字母</span>`
 * pattern (previously hand-written in 4+ places with inconsistent field names).
 *
 * The wrapper owns the border-radius + outline + overflow:hidden in one element,
 * which is what keeps the border continuous — the prior layout stacked a
 * separate bordered online-dot against a rounded <img> and the seam read as
 * "broken/dashed". Here the badge is a child of that same wrapper.
 */
export function Avatar({
  src, name, size = 40, shape = "circle", ring = false, badge, className, style,
}: AvatarProps) {
  const [hasError, setHasError] = useState(false);
  const showImage = Boolean(src) && !hasError;
  const initial = (name || "?").slice(0, 1);

  const borderRadius = shape === "circle" ? "9999px" : `${Math.round(size * 0.28)}px`;

  return (
    <div
      className={`relative flex-shrink-0 ${className ?? ""}`}
      style={{
        width: size,
        height: size,
        borderRadius,
        overflow: "hidden",
        background: "var(--neu-bg)",
        border: ring ? "1px solid color-mix(in srgb, var(--neu-accent) 10%, var(--neu-bg))" : undefined,
        ...style,
      }}
    >
      {showImage ? (
        <img
          src={src}
          alt={name ?? ""}
          className="h-full w-full object-cover"
          onError={() => setHasError(true)}
        />
      ) : (
        <span
          className="flex h-full w-full items-center justify-center"
          style={{
            fontSize: Math.round(size * 0.42),
            color: "var(--neu-text-muted)",
            background: "color-mix(in srgb, var(--neu-accent) 8%, var(--neu-bg))",
          }}
        >
          {initial}
        </span>
      )}
      {badge}
    </div>
  );
}
