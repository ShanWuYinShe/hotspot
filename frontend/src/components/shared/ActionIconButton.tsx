'use client'
import React from "react";
import { motion } from "motion/react";
import { useHoverConfig } from "@/hooks/useHoverConfig";
import { useTheme } from "@/contexts/ThemeContext";

type ActionIconButtonProps = {
  icon: React.ElementType;
  label: string;
  onClick?: () => void;
  variant?: "surface" | "overlay" | "ghost";
  tone?: "muted" | "accent" | "inverse";
  size?: 32 | 34 | 36 | 42;
  disabled?: boolean;
  active?: boolean;
  type?: "button" | "submit" | "reset";
  className?: string;
};

/** Shared icon-only action control for every page header and compact toolbar. */
export function ActionIconButton({
  icon: Icon, label, onClick, variant = "surface", tone = "muted", size = 36, disabled = false, active = false, type = "button", className = "",
}: ActionIconButtonProps) {
  const [hovered, setHovered] = React.useState(false);
  const hover = useHoverConfig();
  const { theme } = useTheme();
  const highlighted = !disabled && (active || (variant === "ghost" && hovered));
  const iconColor = tone === "inverse" ? "white" : tone === "accent" || highlighted ? "var(--neu-accent-primary)" : "var(--neu-text-muted)";
  const style: React.CSSProperties = variant === "overlay"
    ? { width: size, height: size, background: "rgba(0,0,0,.35)", backdropFilter: "blur(10px)", border: "1px solid rgba(255,255,255,.25)" }
    : variant === "ghost"
      ? {
          width: size,
          height: size,
          background: highlighted ? (active ? (hovered ? "var(--neu-active-hover)" : "var(--neu-accent-soft)") : hover.bgHover) : "transparent",
          border: active ? `1px solid ${hovered ? "var(--neu-accent-primary)" : "var(--neu-accent-soft-border)"}` : "1px solid transparent",
          transition: "background 0.18s ease, border-color 0.18s ease",
        }
      : {
          width: size,
          height: size,
          background: active ? (hovered ? "var(--neu-active-hover)" : "var(--neu-accent-soft)") : "var(--neu-bg)",
          border: active ? `1px solid ${hovered ? "var(--neu-accent-primary)" : "var(--neu-accent-soft-border)"}` : "1px solid var(--neu-border)",
          boxShadow: active || theme === "forest" ? "none" : "4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light)",
        };

  return <motion.button whileTap={disabled ? undefined : hover.tap as any} whileHover={disabled ? undefined : hover.btn as any} onHoverStart={() => setHovered(true)} onHoverEnd={() => setHovered(false)} onClick={onClick} disabled={disabled} type={type} aria-label={label} title={label} className={`flex items-center justify-center rounded-full outline-none flex-shrink-0 transition-[box-shadow,background-color,color] focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--neu-bg)] disabled:opacity-45 ${className}`} style={style}><Icon size={Math.round(size * .42)} color={iconColor} strokeWidth={highlighted ? 2 : 1.65} /></motion.button>;
}
