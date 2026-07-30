import { useEffect, useRef, useState } from "react";
import { Check, Moon, Palette, Sun, Database } from "lucide-react";
import { ActionIconButton } from "../shared/ActionIconButton";
import { AnimatePresence, motion } from "motion/react";
import { useTheme } from "../../contexts/ThemeContext";
import { useDismissibleLayer } from "../../hooks/useDismissibleLayer";

type AppearanceMenuProps = { className?: string };

const THEMES = [
  { id: "vibrant" as const, label: "星际", note: "冷色玻璃", swatch: "#6366F1" },
  { id: "forest" as const, label: "清野", note: "自然纸面", swatch: "#3D7A55" },
];

/* ── Dev API source toggle ─────────────────────────────────── */
function DevApiSection() {
  const [enabled, setEnabled] = useState(true);

  useEffect(() => {
    import("../../../mocks/toggle").then(({ isMockEnabled, subscribe, toggleMock: _ }) => {
      setEnabled(isMockEnabled());
      return subscribe(() => setEnabled(isMockEnabled()));
    });
  }, []);

  const handleToggle = async () => {
    const { toggleMock } = await import("../../../mocks/toggle");
    await toggleMock();
  };

  return (
    <motion.button
      type="button"
      whileTap={{ scale: 0.985 }}
      onClick={handleToggle}
      className="mt-2 flex w-full items-center justify-between rounded-[12px] ring-1 ring-[var(--neu-border)] px-3 py-2.5 text-left outline-none transition-colors hover:bg-[var(--neu-accent-soft)] focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)]"
      style={{ color: "var(--neu-text)" }}
    >
      <span className="flex items-center gap-2">
        <span className="flex h-[26px] w-[26px] items-center justify-center rounded-full bg-[var(--neu-accent-soft)]">
          <Database size={13} color="var(--neu-accent-primary)" />
        </span>
        <span>
          <span className="block text-[12px] font-medium">API 数据来源</span>
          <span className="block text-[10px] text-[var(--neu-text-muted)]">{enabled ? "模拟数据" : "真实后端"}</span>
        </span>
      </span>
      <span
        className="relative h-5 w-[34px] overflow-hidden rounded-full"
        style={{ background: enabled ? "var(--neu-accent-primary)" : "var(--neu-border-pressed)" }}
      >
        <motion.span
          animate={{ left: enabled ? 17 : 3 }}
          transition={{ type: "spring", stiffness: 420, damping: 30 }}
          className="absolute top-[3px] h-[14px] w-[14px] rounded-full bg-white shadow-[0_1px_3px_rgba(0,0,0,.18)]"
        />
      </span>
    </motion.button>
  );
}

/* ── Dev auth scenario + login-state controls (mock only) ──── */
const AUTH_SCENARIOS = [
  { id: "ok", label: "正常登录" },
  { id: "invalid-creds", label: "账号/密码错误 (10100)" },
  { id: "banned", label: "账号封禁 (10201)" },
  { id: "param-error", label: "参数错误 (10000)" },
  { id: "rate-limit", label: "限流 (10500)" },
] as const;

function DevAuthSection() {
  const [scenario, setScenario] = useState<string>("ok");

  useEffect(() => {
    import("../../../mocks/data").then(({ getMockAuthScenario }) => {
      setScenario(getMockAuthScenario());
    });
  }, []);

  const apply = async (next: string) => {
    const { setMockAuthScenario } = await import("../../../mocks/data");
    setMockAuthScenario(next as any);
    setScenario(next);
  };

  const resetLoggedOut = async () => {
    const { setMockLoggedIn } = await import("../../../mocks/data");
    // Clear the mock session cookies + state, then reload so AuthContext
    // re-checks status and renders the anonymous UI.
    document.cookie = "access_token=; Path=/; Max-Age=0";
    document.cookie = "refresh_token=; Path=/; Max-Age=0";
    setMockLoggedIn(false);
    window.location.reload();
  };

  return (
    <div className="mt-2 rounded-[12px] ring-1 ring-[var(--neu-border)] px-3 py-2.5" style={{ color: "var(--neu-text)" }}>
      <span className="block text-[12px] font-medium">模拟登录态</span>
      <span className="mb-2 block text-[10px] text-[var(--neu-text-muted)]">影响 login/register/guest/reset 的返回</span>
      <div className="flex flex-col" style={{ gap: 4 }}>
        {AUTH_SCENARIOS.map((s) => (
          <button
            key={s.id}
            type="button"
            onClick={() => apply(s.id)}
            className="flex items-center justify-between rounded-[8px] px-2 py-1.5 text-left text-[11px] outline-none transition-colors"
            style={{
              background: scenario === s.id ? "var(--neu-accent-soft)" : "transparent",
              color: scenario === s.id ? "var(--neu-accent-primary)" : "var(--neu-text)",
            }}
          >
            <span>{s.label}</span>
            {scenario === s.id && <Check size={12} />}
          </button>
        ))}
      </div>
      <button
        type="button"
        onClick={resetLoggedOut}
        className="mt-2 w-full rounded-[8px] ring-1 ring-[var(--neu-border)] px-2 py-1.5 text-[11px] outline-none transition-colors hover:bg-[var(--neu-accent-soft)]"
        style={{ color: "var(--neu-text-muted)" }}
      >
        重置为未登录（匿名浏览）
      </button>
    </div>
  );
}

/** The only theme/mode control. Selection stays open; only explicit completion, outside press, or Escape dismisses it. */
export function AppearanceMenu({ className = "" }: AppearanceMenuProps) {
  const { isDark, toggleDark, theme, setTheme } = useTheme();
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
            className="absolute right-0 top-[42px] z-50 w-[244px] rounded-[18px] ring-1 ring-[var(--neu-border)] bg-[var(--neu-bg)] p-2.5"
            style={{ boxShadow: "0 16px 40px var(--neu-shadow-dark)" }}
          >
            <div className="px-2 pb-2">
              <p className="neu-eyebrow">外观</p>
              <p className="mt-1 text-[11px] text-[var(--neu-text-muted)]">
                当前：{currentTheme} · {modeLabel}
              </p>
            </div>

            <section aria-labelledby="theme-choice-label" className="rounded-[18px] ring-1 ring-[var(--neu-border)] bg-[color-mix(in_srgb,var(--neu-bg)_94%,var(--page-bg)_6%)] p-2">
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
              className="mt-2 flex w-full items-center justify-between rounded-[12px] ring-1 ring-[var(--neu-border)] px-3 py-2.5 text-left outline-none transition-colors hover:bg-[var(--neu-accent-soft)] focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)]"
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

            {import.meta.env.DEV && <DevApiSection />}
            {import.meta.env.DEV && <DevAuthSection />}
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
