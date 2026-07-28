import React, { useEffect, useState } from "react";
import {
  ArrowLeft,
  Bell,
  Lock,
  ChevronRight,
  LogOut,
  Globe,
  Info,
  Volume2,
  MapPin,
  User,
} from "lucide-react";
import { motion } from "motion/react";
import { useNavigate } from "react-router";
import { useHoverConfig } from "../hooks/useHoverConfig";
import { ActionIconButton } from "../components/shared/ActionIconButton";
import { getAccount, getSettings } from "../../api";
import { useAuth } from "../contexts/AuthContext";

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
    <div className="rounded-[22px] overflow-hidden neu-surface-flat">
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
          background: "var(--neu-bg)",
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

/* ══════════════════════════════════════════════════════════════════
   Settings Page
═════════════════════════════════════════════════════════════════ */
export function Settings() {
  const navigate = useNavigate();
  const hover = useHoverConfig();
  const { logout: authLogout } = useAuth();
  const [notif, setNotif] = useState(true);
  const [sounds, setSounds] = useState(true);
  const [privateProfile, setPrivateProfile] = useState(false);
  const [locationSharing, setLocationSharing] = useState(true);
  const [accountName, setAccountName] = useState("");
  const [accountHandle, setAccountHandle] = useState("");
  const [accountEmail, setAccountEmail] = useState("");
  const [accountAvatar, setAccountAvatar] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    async function fetchData() {
      try {
        setLoading(true);
        const [accountRes, settingsRes] = await Promise.all([
          getAccount().catch(() => ({ code: -1, data: null })),
          getSettings().catch(() => ({ code: -1, data: null })),
        ]);
        if (cancelled) return;

        const accountData = (accountRes.code === 0 ? accountRes.data : null) as any;
        const settingsData = (settingsRes.code === 0 ? settingsRes.data : null) as any;

        if (accountData) {
          setAccountName(accountData.name ?? accountData.nickname ?? "");
          setAccountHandle(accountData.handle ?? accountData.username ?? "");
          setAccountEmail(accountData.email ?? "");
          setAccountAvatar(accountData.avatar_url ?? accountData.avatar ?? "");
        }

        if (settingsData) {
          if (settingsData.notifications !== undefined) setNotif(settingsData.notifications);
          if (settingsData.sounds !== undefined) setSounds(settingsData.sounds);
          if (settingsData.private_profile !== undefined) setPrivateProfile(settingsData.private_profile);
          if (settingsData.location_sharing !== undefined) setLocationSharing(settingsData.location_sharing);
        }
      } catch {
        // empty data is normal
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    fetchData();
    return () => { cancelled = true; };
  }, []);

  const handleLogout = async () => {
    // AuthContext.logout clears client auth state and keeps the user on the
    // current route (anonymous browsing). No more redirect to a dead /login.
    await authLogout();
    navigate("/");
  };

  return (
    <>
      {/* Sticky header */}
      <div
        className="sticky top-0 z-10 flex items-center gap-3 px-5 py-3"
        style={{ backgroundColor: "var(--neu-bg)", borderBottom: "1px solid var(--neu-border)" }}
      >
        <ActionIconButton icon={ArrowLeft} label="返回" onClick={() => navigate(-1)} />
        <h1 className="text-[18px] tracking-tight" style={{ color: "var(--neu-text)" }}>
          设置
        </h1>
      </div>

      {/* Page content */}
      <motion.div
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.28, ease: [0.22, 1, 0.36, 1] }}
        className="content-mode-reading flex flex-col px-5 pt-4 pb-10"
        style={{ maxWidth: "var(--content-reading-max)", marginLeft: "auto", marginRight: "auto", width: "100%" }}
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
              {accountAvatar ? (
                <img
                  src={accountAvatar}
                  alt="Avatar"
                  className="w-full h-full object-cover"
                />
              ) : (
                <span className="flex w-full h-full items-center justify-center" style={{ fontSize: 18, color: "var(--neu-text-muted)" }}>
                  {(accountName || "?").slice(0, 1)}
                </span>
              )}
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-[15px] tracking-tight" style={{ color: "var(--neu-text)" }}>{accountName || "用户"}</p>
              <p className="text-[11px] mt-0.5" style={{ color: "var(--neu-text-muted)" }}>{accountHandle}{accountEmail ? ` · ${accountEmail}` : ""}</p>
            </div>
            <ChevronRight style={{ width: 15, height: 15, color: "var(--neu-text-muted)", flexShrink: 0 }} strokeWidth={1.5} />
          </motion.button>
        </div>


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
            onClick={handleLogout}
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