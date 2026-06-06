import type { LucideIcon } from "lucide-react";
import { motion } from "motion/react";

interface EmptyStateProps {
  icon?: LucideIcon;
  title: string;
  hint?: string;
  actionLabel?: string;
  onAction?: () => void;
  className?: string;
}

/**
 * Branded empty / placeholder state. Uses the neumorphic icon badge so empty
 * screens read as intentional rather than "nothing loaded". Keep copy short:
 * `title` is the headline, `hint` an optional one-line nudge.
 */
export function EmptyState({
  icon: Icon,
  title,
  hint,
  actionLabel,
  onAction,
  className,
}: EmptyStateProps) {
  return (
    <div
      className={`flex flex-col items-center justify-center text-center gap-3 py-10 px-8${
        className ? ` ${className}` : ""
      }`}
    >
      {Icon ? (
        <div
          className="neu-icon-badge flex items-center justify-center rounded-full"
          style={{ width: 56, height: 56 }}
        >
          <Icon size={24} style={{ color: "var(--neu-text-muted)" }} />
        </div>
      ) : null}
      <p style={{ fontSize: 14, color: "var(--neu-text-muted)" }}>{title}</p>
      {hint ? (
        <p style={{ fontSize: 12, color: "var(--neu-text-muted)", opacity: 0.7 }}>
          {hint}
        </p>
      ) : null}
      {actionLabel && onAction ? (
        <motion.button
          whileTap={{ scale: 0.94 }}
          onClick={onAction}
          className="mt-1 px-4 py-2 rounded-full outline-none neu-pill-hover"
          style={{
            background: "var(--neu-accent-primary)",
            color: "var(--text-on-accent, white)",
            fontSize: 13,
          }}
        >
          {actionLabel}
        </motion.button>
      ) : null}
    </div>
  );
}
