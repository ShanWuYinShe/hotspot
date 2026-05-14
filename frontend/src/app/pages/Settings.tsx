import React, { useState } from "react";
import {
  ArrowLeft,
  Moon,
  Sun,
  Bell,
  Lock,
  ChevronRight,
  LogOut,
  Globe,
  Info,
  Volume2,
  MapPin,
  User,
  Check,
} from "lucide-react";
import { motion } from "motion/react";
import { useNavigate } from "react-router";
import { useTheme } from "../contexts/ThemeContext";
import { useHoverConfig } from "../hooks/useHoverConfig";

const USER_AVATAR =
  "https://images.unsplash.com/photo-1614339804578-8e82dd7f3214?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400";

/* ── Small helpers ─────────────────────────────────────────────── */

function SectionLabel({ children }: { children: React.ReactNode }) {
  return (
    <p
      className="px-1 mt-3 mb-1.5"
      style={{
        fontSize: 10,
        letterSpacing: "0.14em",
        textTransform: "uppercase",
        color: "var(--neu-text-muted)",
      }}
    >
      {children}
    </p>
  );
}

function Card({ children }: { children: React.ReactNode }) {
  return (
    <div
      className="rounded-[22px] overflow-hidden"
      style={{
        background: "var(--neu-bg)",
        boxShadow:
          "7px 7px 16px var(--neu-shadow-dark), -7px -7px 16px var(--neu-shadow-light)",
        border: "1px solid var(--neu-border)",
      }}
    >
      {children}
    </div>
  );
}

function Divider() {
  return <div className="mx-4" style={{ height: 1, background: "var(--neu-border)" }} />;
}

function IconBadge({ Icon, color = "var(--neu-text-muted)" }: { Icon: React.ElementType; color?: string }) {
  return (
    <div
      className="flex items-center justify-center rounded-full flex-shrink-0 neu-icon-badge"
      style={{ width: 36, height: 36 }}
    >
      <Icon style={{ width: 15, height: 15, color }} strokeWidth={1.5} />
    </div>
  );
}

/* ── Toggle switch — iOS-style, works in every theme & mode ────
   OFF: neutral gray track + white thumb (always visible)
   ON:  accent-primary track + white thumb
   The thumb is always pure white with a real drop-shadow so it
   never disappears against any background colour.
──────────────────────────────────────────────────────────────── */
function NeuToggle({ on, onToggle }: { on: boolean; onToggle: () => void }) {
  return (
    <motion.button
      whileTap={{ scale: 0.94 }}
      onClick={onToggle}
      role="switch"
      aria-checked={on}
      className="relative outline-none flex-shrink-0"
      style={{
        width: 50,
        height: 28,
        borderRadius: 14,
        /* Track colour — semantic, not themed */
        background: on
          ? "var(--neu-accent-primary)"
          : "rgba(120, 120, 128, 0.22)",
        /* Smooth colour transition */
        transition: "background 0.25s cubic-bezier(0.4,0,0.2,1)",
        /* Faint inner border so track has edge on white bg */
        boxShadow: on
          ? "inset 0 1px 2px rgba(0,0,0,0.12)"
          : "inset 0 1px 2px rgba(0,0,0,0.08), 0 0 0 1px rgba(120,120,128,0.15)",
      }}
    >
      {/* Thumb — always white, always has a visible shadow */}
      <motion.div
        animate={{ x: on ? 23 : 2 }}
        transition={{ type: "spring", stiffness: 420, damping: 30 }}
        style={{
          position: "absolute",
          top: 3,
          width: 22,
          height: 22,
          borderRadius: 11,
          background: "#FFFFFF",
          boxShadow: "0 1px 4px rgba(0,0,0,0.26), 0 0.5px 1px rgba(0,0,0,0.14)",
        }}
      />
    </motion.button>
  );
}

function ToggleRow({
  Icon, iconColor, label, sub, on, onToggle,
}: {
  Icon: React.ElementType; iconColor?: string; label: string; sub: string; on: boolean; onToggle: () => void;
}) {
  return (
    <div className="flex items-center gap-3 px-4 py-3.5">
      <IconBadge Icon={Icon} color={iconColor} />
      <div className="flex-1 min-w-0">
        <p className="text-[14px]" style={{ color: "var(--neu-text)" }}>{label}</p>
        <p className="text-[11px]" style={{ color: "var(--neu-text-muted)" }}>{sub}</p>
      </div>
      <NeuToggle on={on} onToggle={onToggle} />
    </div>
  );
}

function NavRow({ Icon, label, value }: { Icon: React.ElementType; label: string; value: string }) {
  const hover = useHoverConfig();
  // For forest flat theme: x-nudge only. Others: x-nudge + tap spring.
  return (
    <motion.button
      whileTap={hover.tap as any}
      whileHover={{ x: 3, transition: (hover.tap as any).transition }}
      className="flex items-center gap-3 px-4 py-3.5 w-full outline-none text-left"
      style={{ transition: "background 0.16s ease" }}
    >
      <IconBadge Icon={Icon} />
      <div className="flex-1 min-w-0">
        <p className="text-[14px]" style={{ color: "var(--neu-text)" }}>{label}</p>
      </div>
      <span className="text-[12px] flex-shrink-0" style={{ color: "var(--neu-text-muted)" }}>{value}</span>
      <ChevronRight style={{ width: 15, height: 15, color: "var(--neu-text-muted)", flexShrink: 0 }} strokeWidth={1.5} />
    </motion.button>
  );
}

/* ── Theme Card ────────────────────────────────────────────────── */
function ThemeCard({
  id, name, desc, previewBg, previewAccent, isSelected, onSelect,
}: {
  id: AppTheme; name: string; desc: string;
  previewBg: string; previewAccent: string;
  isSelected: boolean; onSelect: () => void;
}) {
  const hover = useHoverConfig();
  return (
    <motion.button
      whileTap={hover.tap as any}
      whileHover={!isSelected ? hover.card as any : { scale: 1.02, transition: { type: "tween", duration: 0.2, ease: [0.22,1,0.36,1] } }}
      onClick={onSelect}
      className="flex-1 flex flex-col items-center gap-3 py-4 rounded-[20px] outline-none"
      style={{
        background: previewBg,
        boxShadow: isSelected
          ? `0 0 0 2.5px ${previewAccent}, 5px 5px 12px rgba(0,0,0,0.10)`
          : "5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light)",
        border: isSelected
          ? `1.5px solid ${previewAccent}`
          : "1.5px solid var(--neu-border)",
        transition: "box-shadow 0.2s ease, border-color 0.2s ease",
      }}
    >
      {/* Mini phone frame preview */}
      <div
        className="relative flex flex-col overflow-hidden"
        style={{
          width: 60,
          height: 80,
          borderRadius: 12,
          background: previewBg,
          border: `1px solid ${previewAccent}22`,
          boxShadow: "0 2px 8px rgba(0,0,0,0.12)",
        }}
      >
        {/* Mini nav bar */}
        <div
          className="absolute bottom-0 left-0 right-0 flex justify-around items-center"
          style={{ height: 20, background: previewBg, borderTop: `1px solid ${previewAccent}30` }}
        >
          {[0, 1, 2].map((i) => (
            <div
              key={i}
              style={{
                width: i === 1 ? 14 : 10,
                height: i === 1 ? 14 : 8,
                borderRadius: i === 1 ? 7 : 4,
                background: i === 1 ? previewAccent : `${previewAccent}55`,
              }}
            />
          ))}
        </div>

        {/* Mini content lines */}
        <div className="flex flex-col gap-1.5 p-2 pt-3">
          {[80, 60, 70].map((w, i) => (
            <div
              key={i}
              style={{
                width: `${w}%`,
                height: 5,
                borderRadius: 3,
                background: i === 0 ? `${previewAccent}90` : `${previewAccent}35`,
              }}
            />
          ))}
        </div>
      </div>

      <div className="flex flex-col items-center gap-0.5">
        <span className="text-[12px]" style={{ color: previewAccent }}>{name}</span>
        <span className="text-[10px] text-center leading-snug" style={{ color: `${previewAccent}99` }}>{desc}</span>
      </div>

      {/* Selected checkmark */}
      {isSelected && (
        <div
          className="flex items-center justify-center rounded-full"
          style={{ width: 20, height: 20, background: previewAccent }}
        >
          <Check style={{ width: 12, height: 12, color: "white" }} strokeWidth={2.5} />
        </div>
      )}
    </motion.button>
  );
}

/* ══════════════════════════════════════════════════════════════════
   Settings Page
═════════════════════════════════════════════════════════════════ */
export function Settings() {
  const navigate = useNavigate();
  const { isDark, toggleDark, theme, setTheme } = useTheme();
  const hover = useHoverConfig();
  const [notif, setNotif] = useState(true);
  const [sounds, setSounds] = useState(true);
  const [privateProfile, setPrivateProfile] = useState(false);
  const [locationSharing, setLocationSharing] = useState(true);

  return (
    <>
      {/* Sticky header */}
      <div
        className="sticky top-0 z-10 flex items-center gap-3 px-5 py-3"
        style={{ backgroundColor: "var(--neu-bg)", borderBottom: "1px solid var(--neu-border)" }}
      >
        <motion.button
          whileTap={{ scale: 0.92 }}
          onClick={() => navigate(-1)}
          className="flex items-center justify-center rounded-full outline-none flex-shrink-0 neu-surface-flat"
          style={{ width: 36, height: 36 }}
        >
          <ArrowLeft style={{ width: 15, height: 15, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
        </motion.button>
        <h1 className="text-[18px] tracking-tight" style={{ color: "var(--neu-text)" }}>
          设置
        </h1>
      </div>

      {/* Page content */}
      <motion.div
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.28, ease: [0.22, 1, 0.36, 1] }}
        className="flex flex-col px-5 pt-4 pb-10"
        style={{ maxWidth: 560, marginLeft: "auto", marginRight: "auto", width: "100%" }}
      >
        {/* User card — hero with ambient glow */}
        <div className="neu-ambient-glow mb-3">
          <motion.button
            whileTap={hover.tap as any}
            className="flex items-center gap-4 px-4 py-3.5 rounded-[20px] w-full outline-none text-left neu-surface-hero"
          >
            <div className="rounded-full overflow-hidden flex-shrink-0"
              style={{
                width: 48, height: 48,
                border: "2px solid var(--neu-bg)",
                boxShadow: "0 4px 12px var(--neu-glow)",
              }}>
              <img
                src="https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=200"
                alt="Avatar"
                className="w-full h-full object-cover"
              />
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-[15px] tracking-tight" style={{ color: "var(--neu-text)" }}>Elara Vance</p>
              <p className="text-[11px] mt-0.5" style={{ color: "var(--neu-text-muted)" }}>@elara.v · elara@example.com</p>
            </div>
            <ChevronRight style={{ width: 15, height: 15, color: "var(--neu-text-muted)", flexShrink: 0 }} strokeWidth={1.5} />
          </motion.button>
        </div>

        {/* ── Theme picker ──────────────────────────────────
            Three themes:
            星际 (vibrant) — 3D floating glass, electric
            暖砂 (neu)     — 3D neumorphic clay, warm
            清野 (forest)  — flat, organic paper
        ─────────────────────────────────────────────────── */}
        <SectionLabel>主题风格</SectionLabel>
        <div className="flex gap-2.5">
          <ThemeCard
            id="vibrant"
            name="星际"
            desc="电光立体"
            previewBg={isDark ? "#130F26" : "#FAFAFE"}
            previewAccent={isDark ? "#818CF8" : "#6366F1"}
            isSelected={theme === "vibrant"}
            onSelect={() => setTheme("vibrant")}
          />
          <ThemeCard
            id="neu"
            name="暖砂"
            desc="质感立体"
            previewBg={isDark ? "#1A1C23" : "#EAE5DE"}
            previewAccent={isDark ? "#E8A87C" : "#D4845A"}
            isSelected={theme === "neu"}
            onSelect={() => setTheme("neu")}
          />
          <ThemeCard
            id="forest"
            name="清野"
            desc="自然扁平"
            previewBg={isDark ? "#161A16" : "#F9F9F5"}
            previewAccent={isDark ? "#52B878" : "#3D7A55"}
            isSelected={theme === "forest"}
            onSelect={() => setTheme("forest")}
          />
        </div>

        {/* ── Appearance ─────────────────────────────────── */}
        <SectionLabel>外观</SectionLabel>
        <Card>
          <ToggleRow
            Icon={isDark ? Moon : Sun}
            iconColor="var(--neu-accent-primary)"
            label="深色模式"
            sub={isDark ? "当前为深色主题" : "当前为浅色主题"}
            on={isDark}
            onToggle={toggleDark}
          />
        </Card>

        {/* ── Notifications ──────────────────────────────── */}
        <SectionLabel>通知</SectionLabel>
        <Card>
          <ToggleRow
            Icon={Bell}
            label="推送通知"
            sub="消息、提及与新动态提醒"
            on={notif}
            onToggle={() => setNotif((v) => !v)}
          />
          <Divider />
          <ToggleRow
            Icon={Volume2}
            label="应用声音"
            sub="操作反馈音效"
            on={sounds}
            onToggle={() => setSounds((v) => !v)}
          />
        </Card>

        {/* ── Privacy ────────────────────────────────────── */}
        <SectionLabel>隐私与安全</SectionLabel>
        <Card>
          <ToggleRow
            Icon={Lock}
            label="私密账号"
            sub="仅关注者可查看你的内容"
            on={privateProfile}
            onToggle={() => setPrivateProfile((v) => !v)}
          />
          <Divider />
          <ToggleRow
            Icon={MapPin}
            label="位置共享"
            sub="允许附近用户发现你"
            on={locationSharing}
            onToggle={() => setLocationSharing((v) => !v)}
          />
        </Card>

        {/* ── General ────────────────────────────────────── */}
        <SectionLabel>通用</SectionLabel>
        <Card>
          <NavRow Icon={Globe} label="语言" value="中文" />
          <Divider />
          <NavRow Icon={User} label="账号信息" value="" />
          <Divider />
          <NavRow Icon={Info} label="关于" value="v 1.0.0" />
        </Card>

        {/* ── Log Out — flat, less visually heavy ─────────── */}
        <div className="mt-5">
          <motion.button
            whileTap={hover.tap as any}
            className="flex items-center justify-center gap-2 px-4 py-3 rounded-[14px] w-full outline-none neu-surface-flat"
            style={{ color: "var(--neu-accent-primary)" }}
          >
            <LogOut style={{ width: 14, height: 14 }} strokeWidth={1.75} />
            <span className="text-[14px]">退出登录</span>
          </motion.button>
        </div>
      </motion.div>
    </>
  );
}