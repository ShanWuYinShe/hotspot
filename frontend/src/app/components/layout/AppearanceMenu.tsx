import React, { useRef, useState } from "react";
import { Check, Moon, Palette, Settings, Sun } from "lucide-react";
import { ActionIconButton } from "../shared/ActionIconButton";
import { AnimatePresence, motion } from "motion/react";
import { useTheme } from "../../contexts/ThemeContext";
import { useNavigate } from "react-router";
import { useHoverConfig } from "../../hooks/useHoverConfig";
import { useDismissibleLayer } from "../../hooks/useDismissibleLayer";
import { ROUTES } from "../../lib/routes";

type AppearanceMenuProps = { className?: string };

const THEMES = [
  { id: "vibrant" as const, label: "星际", note: "冷色玻璃", swatch: "#6366F1" },
  { id: "forest" as const, label: "清野", note: "自然纸面", swatch: "#3D7A55" },
];

/** The only theme/mode control. Selection stays open; only explicit completion, outside press, or Escape dismisses it. */
export function AppearanceMenu({ className = "" }: AppearanceMenuProps) {
  const { isDark, toggleDark, theme, setTheme } = useTheme();
  const hover = useHoverConfig();
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);
  const currentTheme = THEMES.find((item) => item.id === theme)?.label ?? "星际";
  const modeLabel = isDark ? "深色" : "浅色";

  useDismissibleLayer({ open, onDismiss: () => setOpen(false), rootRef, closeOnOutsidePress: true });

  return (
    <div ref={rootRef} className={`relative ${className}`}>
      <ActionIconButton
        icon={Palette}
        label="外观设置"
        onClick={() => setOpen((value) => !value)}
        variant="ghost"
        active={open}
        size={36}
      />

      <AnimatePresence>
        {open && (
          <motion.div
            initial={{ opacity: 0, y: -6, scale: 0.98 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: -6, scale: 0.98 }}
            transition={{ duration: 0.16, ease: [0.22, 1, 0.36, 1] }}
            role="dialog"
            aria-label="外观设置"
            className="absolute right-0 top-[42px] z-50 w-[244px] rounded-[18px] border border-[var(--neu-border)] bg-[var(--neu-bg)] p-2.5"
            style={{ boxShadow: "0 16px 40px var(--neu-shadow-dark)" }}
          >
            <div className="px-2 pb-2">
              <p className="neu-eyebrow">外观</p>
              <p className="mt-1 text-[11px] text-[var(--neu-text-muted)]">
                当前：{currentTheme} · {modeLabel}
              </p>
            </div>

            <section aria-labelledby="theme-choice-label" className="rounded-[18px] border border-[var(--neu-border)] bg-[color-mix(in_srgb,var(--neu-bg)_94%,var(--page-bg)_6%)] p-2">
              <div className="flex items-center justify-between px-2 pb-2 pt-1">
                <p id="theme-choice-label" className="neu-eyebrow">主题</p>
                <span className="text-[10px] text-[var(--neu-text-muted)]">选择一种基调</span>
              </div>
              <div className="grid grid-cols-2 gap-1" role="group" aria-label="选择主题">
                {THEMES.map((item) => {
                  const selected = theme === item.id;
                  return (
                    <button
                      key={item.id}
                      type="button"
                      onClick={() => setTheme(item.id)}
                      aria-pressed={selected}
                      className={`flex min-w-0 items-center gap-2 rounded-[12px] px-2 py-2 text-left outline-none transition-colors focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)] ${
                        selected
                          ? "bg-[var(--neu-accent-soft)] text-[var(--neu-accent-primary)] hover:bg-[var(--neu-active-hover)]"
                          : "neu-surface-ghost text-[var(--neu-text-muted)]"
                      }`}
                    >
                      <span
                        className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full"
                        style={{ background: item.swatch }}
                      >
                        {selected && <Check size={13} color="white" strokeWidth={2.5} />}
                      </span>
                      <span className="min-w-0">
                        <span className="block truncate text-[12px] font-medium">{item.label}</span>
                        <span className="block truncate text-[10px] text-[var(--neu-text-muted)]">{item.note}</span>
                      </span>
                    </button>
                  );
                })}
              </div>
            </section>

            <motion.button
              type="button"
              whileTap={{ scale: 0.985 }}
              onClick={toggleDark}
              aria-pressed={isDark}
              className="mt-2 flex w-full items-center justify-between rounded-[12px] border border-[var(--neu-border)] px-3 py-2.5 text-left outline-none transition-colors hover:bg-[var(--neu-accent-soft)] focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)]"
              style={{ color: "var(--neu-text)" }}
            >
              <span className="flex items-center gap-2">
                <span className="flex h-[26px] w-[26px] items-center justify-center rounded-full bg-[var(--neu-accent-soft)]">
                  {isDark ? <Sun size={13} color="var(--neu-accent-primary)" /> : <Moon size={13} color="var(--neu-accent-primary)" />}
                </span>
                <span>
                  <span className="block text-[12px] font-medium">{isDark ? "深色模式" : "浅色模式"}</span>
                  <span className="block text-[10px] text-[var(--neu-text-muted)]">点击切换</span>
                </span>
              </span>
              <span className="relative h-5 w-[34px] overflow-hidden rounded-full" style={{ background: isDark ? "var(--neu-accent-primary)" : "var(--neu-border-pressed)" }}>
                <motion.span
                  animate={{ left: isDark ? 17 : 3 }}
                  transition={{ type: "spring", stiffness: 420, damping: 30 }}
                  className="absolute top-[3px] h-[14px] w-[14px] rounded-full bg-white shadow-[0_1px_3px_rgba(0,0,0,.18)]"
                />
              </span>
            </motion.button>

            <div className="mt-2 flex items-center gap-1.5 border-t border-[var(--neu-border)] pt-2">
              <button
                type="button"
                onClick={() => { setOpen(false); navigate(ROUTES.settings); }}
                className="flex min-w-0 flex-1 items-center gap-2 rounded-[10px] px-2.5 py-2 text-left text-[11px] text-[var(--neu-text-muted)] outline-none transition-colors hover:bg-[var(--neu-accent-soft)] hover:text-[var(--neu-text)] focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)]"
              >
                <Settings size={14} strokeWidth={1.6} />
                账号与应用设置
              </button>
              <motion.button
                type="button"
                whileTap={hover.tap as any}
                onClick={() => setOpen(false)}
                className="rounded-full bg-[var(--neu-accent-primary)] px-3 py-1.5 text-[11px] font-medium text-white outline-none shadow-[0_3px_10px_var(--neu-glow)] focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--neu-bg)]"
              >
                完成
              </motion.button>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
