'use client'
import React, { useState, useEffect } from "react";
import { Wifi } from "lucide-react";

/** Simulated iOS-style status bar — real clock, decorative signal/battery. */
export function StatusBar() {
  const [time, setTime] = useState(new Date());

  useEffect(() => {
    const timer = setInterval(() => setTime(new Date()), 1000);
    return () => clearInterval(timer);
  }, []);

  const h = time.getHours().toString().padStart(2, "0");
  const m = time.getMinutes().toString().padStart(2, "0");

  return (
    <div
      className="flex-shrink-0 flex items-center justify-between px-5 select-none"
      style={{
        height: 44,
        backgroundColor: "var(--neu-bg)",
        /* Real device: paddingTop: "env(safe-area-inset-top, 0px)" */
      }}
    >
      {/* ── Time ── */}
      <span
        style={{
          fontSize: 15,
          fontWeight: 600,
          color: "var(--neu-text)",
          letterSpacing: "-0.3px",
          fontVariantNumeric: "tabular-nums",
        }}
      >
        {h}:{m}
      </span>

      {/* ── System indicators ── */}
      <div className="flex items-center gap-[5px]">
        <CellSignal bars={3} total={4} />
        <Wifi
          style={{ width: 14, height: 14, color: "var(--neu-text)" }}
          strokeWidth={2.5}
        />
        <BatteryIcon level={0.72} />
      </div>
    </div>
  );
}

/* ── Cell signal bars ─────────────────────────── */
function CellSignal({ bars, total }: { bars: number; total: number }) {
  const heights = [4, 7, 10, 13];
  return (
    <div className="flex items-end gap-[2px]">
      {heights.slice(0, total).map((h, i) => (
        <div
          key={i}
          style={{
            width: 3,
            height: h,
            borderRadius: 1,
            backgroundColor: "var(--neu-text)",
            opacity: i < bars ? 1 : 0.28,
          }}
        />
      ))}
    </div>
  );
}

/* ── Battery icon ─────────────────────────────── */
function BatteryIcon({ level }: { level: number }) {
  const isLow = level < 0.2;
  return (
    <div style={{ display: "flex", alignItems: "center" }}>
      {/* Outer shell */}
      <div
        style={{
          width: 23,
          height: 11,
          borderRadius: 3,
          border: "1.5px solid var(--neu-text)",
          position: "relative",
          overflow: "hidden",
          boxSizing: "border-box",
        }}
      >
        {/* Fill */}
        <div
          style={{
            position: "absolute",
            inset: 1,
            right: "auto",
            width: `calc(${level * 100}% - 2px)`,
            borderRadius: 1.5,
            backgroundColor: isLow ? "var(--destructive)" : "var(--neu-text)",
          }}
        />
      </div>
      {/* Terminal nub */}
      <div
        style={{
          width: 2,
          height: 5,
          borderRadius: "0 1px 1px 0",
          backgroundColor: "var(--neu-text)",
          opacity: 0.45,
          marginLeft: 1,
        }}
      />
    </div>
  );
}