import React from "react";

/** Reusable data-display primitives. Use these before inventing a one-off stat block. */
export type MetricItem = {
  label: string;
  value: React.ReactNode;
  tone?: "default" | "accent" | "positive" | "warning";
};

const tones = {
  default: "text-[var(--neu-text)]",
  accent: "text-[var(--neu-accent-primary)]",
  positive: "text-[var(--semantic-online)]",
  warning: "text-[var(--semantic-warning)]",
};

/** Dense, horizontally comparable metrics: counts, ratios, and relationship states. */
export function MetricStrip({ items, className = "" }: { items: MetricItem[]; className?: string }) {
  return (
    <dl className={`grid rounded-[14px] ring-1 ring-[var(--neu-border)] ${className}`} style={{ gridTemplateColumns: `repeat(${items.length}, minmax(0, 1fr))`, gap: 1, background: "var(--neu-border)" }}>
      <div style={{ overflow: "hidden", borderRadius: 13 }}>
        {items.map((item) => (
          <div key={item.label} className="flex min-w-0 flex-col items-center bg-[var(--neu-bg)] px-2 py-2.5 text-center">
            <dd className={`truncate text-[19px] font-semibold leading-none ${tones[item.tone ?? "default"]}`}>{item.value}</dd>
            <dt className="mt-1 truncate text-[11px] text-[var(--neu-text-muted)]">{item.label}</dt>
          </div>
        ))}
      </div>
    </dl>
  );
}

/** A compact label/value pair for settings, metadata, and side rails. */
export function DataKeyValue({ label, value, detail }: { label: string; value: React.ReactNode; detail?: React.ReactNode }) {
  return (
    <div className="flex min-w-0 items-center justify-between gap-4">
      <div className="min-w-0"><p className="text-[12px] text-[var(--neu-text-muted)]">{label}</p>{detail && <p className="mt-0.5 truncate text-[11px] text-[var(--neu-text-muted)]">{detail}</p>}</div>
      <p className="shrink-0 text-[13px] font-medium text-[var(--neu-text)]">{value}</p>
    </div>
  );
}

/** Semantic state tag for online, selected, pending, and similar compact labels. */
export function DataStatus({ children, tone = "accent" }: { children: React.ReactNode; tone?: "accent" | "positive" | "warning" }) {
  const style = tone === "positive"
    ? "border-[color-mix(in_srgb,var(--semantic-online)_25%,transparent)] bg-[color-mix(in_srgb,var(--semantic-online)_12%,transparent)] text-[var(--semantic-online)]"
    : tone === "warning"
      ? "border-[color-mix(in_srgb,var(--semantic-warning)_25%,transparent)] bg-[color-mix(in_srgb,var(--semantic-warning)_12%,transparent)] text-[var(--semantic-warning)]"
      : "border-[var(--neu-accent-soft-border)] bg-[var(--neu-accent-soft)] text-[var(--neu-accent-primary)]";
  return <span className={`inline-flex items-center rounded-full border px-2 py-0.5 text-[10px] font-medium ${style}`}>{children}</span>;
}
