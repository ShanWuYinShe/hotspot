import { useState, useCallback } from "react";
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
import { useAuthStore } from "@/lib/stores/auth";
import { useSettingsData } from "@/lib/hooks/use-settings-data";
import { useAuthActions } from "@/lib/hooks/use-auth-actions";
import { useDocumentTitle } from "@/lib/hooks/use-document-title";
import { goBack } from "@/lib/utils/navigation";

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
  return (
    <div
      className="mx-4"
      style={{ height: 1, background: "var(--neu-border)" }}
    />
  );
}

function IconBadge({
  Icon,
  color = "var(--neu-text-muted)",
}: {
  Icon: React.ElementType;
  color?: string;
}) {
  return (
    <div
      className="flex items-center justify-center rounded-full flex-shrink-0 neu-icon-badge"
      style={{ width: 36, height: 36 }}
    >
      <Icon style={{ width: 15, height: 15, color }} strokeWidth={1.5} />
    </div>
  );
}

function NeuToggle({ on }: { on: boolean }) {
  return (
    <div
      aria-hidden="true"
      className="relative outline-none flex-shrink-0"
      style={{
        width: 50,
        height: 28,
        borderRadius: 14,
        background: on
          ? "var(--neu-accent-primary)"
          : "rgba(120, 120, 128, 0.22)",
        transition: "background 0.25s cubic-bezier(0.4,0,0.2,1)",
        boxShadow: on
          ? "inset 0 1px 2px rgba(0,0,0,0.12)"
          : "inset 0 1px 2px rgba(0,0,0,0.08), 0 0 0 1px rgba(120,120,128,0.15)",
      }}
    >
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
    </div>
  );
}

function ToggleRow({
  Icon,
  iconColor,
  label,
  sub,
  on,
  onToggle,
}: {
  Icon: React.ElementType;
  iconColor?: string;
  label: string;
  sub: string;
  on: boolean;
  onToggle: () => void;
}) {
  const hover = useHoverConfig();
  return (
    <motion.button
      whileTap={hover.tap}
      whileHover={{ x: 2, transition: (hover.tap as any).transition }}
      onClick={onToggle}
      role="switch"
      aria-checked={on}
      className="flex items-center gap-3 px-4 py-3.5 w-full outline-none text-left neu-focus-ring"
    >
      <IconBadge Icon={Icon} color={iconColor} />
      <div className="flex-1 min-w-0">
        <p className="text-[14px]" style={{ color: "var(--neu-text)" }}>
          {label}
        </p>
        <p className="text-[11px]" style={{ color: "var(--neu-text-muted)" }}>
          {sub}
        </p>
      </div>
      <NeuToggle on={on} />
    </motion.button>
  );
}

function NavRow({
  Icon,
  label,
  value,
}: {
  Icon: React.ElementType;
  label: string;
  value: string;
}) {
  const hover = useHoverConfig();
  return (
    <motion.button
      whileTap={hover.tap}
      whileHover={{ x: 3, transition: hover.tap.transition }}
      className="flex items-center gap-3 px-4 py-3.5 w-full outline-none text-left neu-focus-ring"
      style={{ transition: "background 0.16s ease" }}
    >
      <IconBadge Icon={Icon} />
      <div className="flex-1 min-w-0">
        <p className="text-[14px]" style={{ color: "var(--neu-text)" }}>
          {label}
        </p>
      </div>
      <span
        className="text-[12px] flex-shrink-0"
        style={{ color: "var(--neu-text-muted)" }}
      >
        {value}
      </span>
      <ChevronRight
        style={{
          width: 15,
          height: 15,
          color: "var(--neu-text-muted)",
          flexShrink: 0,
        }}
        strokeWidth={1.5}
      />
    </motion.button>
  );
}

/* ── Theme Card ────────────────────────────────────────────────── */
function ThemeCard({
  name,
  desc,
  previewBg,
  previewAccent,
  isSelected,
  onSelect,
}: {
  id: string;
  name: string;
  desc: string;
  previewBg: string;
  previewAccent: string;
  isSelected: boolean;
  onSelect: () => void;
}) {
  const hover = useHoverConfig();
  return (
    <motion.button
      whileTap={hover.tap}
      whileHover={
        !isSelected
          ? hover.card
          : {
              scale: 1.02,
              transition: {
                type: "tween",
                duration: 0.2,
                ease: [0.22, 1, 0.36, 1],
              },
            }
      }
      onClick={onSelect}
      className="flex-1 flex flex-col items-center gap-3 py-4 rounded-[20px] outline-none neu-focus-ring"
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
        <div
          className="absolute bottom-0 left-0 right-0 flex justify-around items-center"
          style={{
            height: 20,
            background: previewBg,
            borderTop: `1px solid ${previewAccent}30`,
          }}
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
        <div className="flex flex-col gap-1.5 p-2 pt-3">
          {[80, 60, 70].map((w, i) => (
            <div
              key={i}
              style={{
                width: `${w}%`,
                height: 5,
                borderRadius: 3,
                background:
                  i === 0 ? `${previewAccent}90` : `${previewAccent}35`,
              }}
            />
          ))}
        </div>
      </div>
      <div className="flex flex-col items-center gap-0.5">
        <span className="text-[12px]" style={{ color: previewAccent }}>
          {name}
        </span>
        <span
          className="text-[10px] text-center leading-snug"
          style={{ color: `${previewAccent}99` }}
        >
          {desc}
        </span>
      </div>
      {isSelected && (
        <div
          className="flex items-center justify-center rounded-full"
          style={{ width: 20, height: 20, background: previewAccent }}
        >
          <Check
            style={{ width: 12, height: 12, color: "white" }}
            strokeWidth={2.5}
          />
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
  const currentAccount = useAuthStore((s) => s.currentAccount);
  const profile = useAuthStore((s) => s.profile);
  const storeLogout = useAuthStore((s) => s.logout);
  const { settings, error, updateSetting, refresh } = useSettingsData();
  const { logout: apiLogout } = useAuthActions();
  useDocumentTitle("设置");

  const [notif, setNotif] = useState(true);
  const [sounds, setSounds] = useState(true);

  const handleLogout = useCallback(async () => {
    await apiLogout();
    storeLogout();
    navigate("/login");
  }, [apiLogout, storeLogout, navigate]);

  const avatarUrl = currentAccount?.avatar_url;
  const nickname = currentAccount?.nickname || profile?.nickname || "用户";
  const userId = profile?.user_id?.slice(0, 8) || "";
  const email = profile?.email;

  return (
    <>
      {/* Sticky header */}
      <div
        className="sticky top-0 z-10 flex items-center gap-3 px-5 py-3"
        style={{
          backgroundColor: "var(--neu-bg)",
          borderBottom: "1px solid var(--neu-border)",
        }}
      >
        <motion.button
          whileTap={{ scale: 0.88 }}
          onClick={() => goBack(navigate)}
          aria-label="返回"
          className="flex items-center justify-center rounded-full outline-none flex-shrink-0 neu-surface-flat neu-focus-ring"
          style={{ width: 44, height: 44 }}
        >
          <ArrowLeft
            style={{ width: 15, height: 15, color: "var(--neu-text-muted)" }}
            strokeWidth={1.5}
          />
        </motion.button>
        <h1
          className="text-[18px] tracking-tight"
          style={{ color: "var(--neu-text)" }}
        >
          设置
        </h1>
      </div>

      {/* Page content */}
      <motion.div
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.28, ease: [0.22, 1, 0.36, 1] }}
        className="flex flex-col px-5 pt-4 pb-10"
        style={{
          maxWidth: 560,
          marginLeft: "auto",
          marginRight: "auto",
          width: "100%",
        }}
      >
        {error && (
          <div
            className="flex items-center justify-between px-4 py-2.5 rounded-[14px] mb-3"
            style={{ background: "var(--neu-accent-soft)" }}
          >
            <span style={{ fontSize: 13, color: "var(--neu-accent-primary)" }}>
              {error}
            </span>
            <motion.button
              whileTap={{ scale: 0.94 }}
              onClick={refresh}
              className="px-3 py-1 rounded-full outline-none neu-focus-ring"
              style={{
                background: "var(--neu-accent-primary)",
                color: "var(--text-on-accent, white)",
                fontSize: 12,
              }}
            >
              重试
            </motion.button>
          </div>
        )}
        {/* User card */}
        <div className="neu-ambient-glow mb-3">
          <motion.button
            whileTap={hover.tap}
            onClick={() => navigate("/profile")}
            className="flex items-center gap-4 px-4 py-3.5 rounded-[20px] w-full outline-none text-left neu-surface-hero neu-focus-ring"
          >
            <div
              className="rounded-full overflow-hidden flex-shrink-0"
              style={{
                width: 48,
                height: 48,
                border: "2px solid var(--neu-bg)",
                boxShadow: "0 4px 12px var(--neu-glow)",
              }}
            >
              {avatarUrl ? (
                <img
                  src={avatarUrl}
                  alt="头像"
                  className="w-full h-full object-cover"
                />
              ) : (
                <div
                  className="w-full h-full flex items-center justify-center"
                  style={{ background: "var(--neu-accent-primary)" }}
                >
                  <span
                    style={{
                      fontSize: 20,
                      color: "var(--text-on-accent, white)",
                    }}
                  >
                    {nickname.charAt(0)}
                  </span>
                </div>
              )}
            </div>
            <div className="flex-1 min-w-0">
              <p
                className="text-[15px] tracking-tight"
                style={{ color: "var(--neu-text)" }}
              >
                {nickname}
              </p>
              <p
                className="text-[11px] mt-0.5"
                style={{ color: "var(--neu-text-muted)" }}
              >
                @{userId}
                {email ? ` · ${email}` : ""}
              </p>
            </div>
            <ChevronRight
              style={{
                width: 15,
                height: 15,
                color: "var(--neu-text-muted)",
                flexShrink: 0,
              }}
              strokeWidth={1.5}
            />
          </motion.button>
        </div>

        {/* Theme picker */}
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

        {/* Appearance */}
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

        {/* Notifications */}
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

        {/* Privacy */}
        <SectionLabel>隐私与安全</SectionLabel>
        <Card>
          <ToggleRow
            Icon={Lock}
            label="私密账号"
            sub="仅关注者可查看你的内容"
            on={settings?.privacy_level === "private"}
            onToggle={() =>
              updateSetting({
                privacy_level:
                  settings?.privacy_level === "private" ? "public" : "private",
              })
            }
          />
          <Divider />
          <ToggleRow
            Icon={MapPin}
            label="位置共享"
            sub="允许附近用户发现你"
            on={settings?.location_visible ?? true}
            onToggle={() =>
              updateSetting({
                location_visible: !(settings?.location_visible ?? true),
              })
            }
          />
        </Card>

        {/* General */}
        <SectionLabel>通用</SectionLabel>
        <Card>
          <NavRow Icon={Globe} label="语言" value="中文" />
          <Divider />
          <NavRow Icon={User} label="账号信息" value="" />
          <Divider />
          <NavRow Icon={Info} label="关于" value="v 1.0.0" />
        </Card>

        {/* Log Out */}
        <div className="mt-5">
          <motion.button
            whileTap={hover.tap}
            onClick={handleLogout}
            className="flex items-center justify-center gap-2 px-4 py-3 rounded-[14px] w-full outline-none neu-surface-flat neu-focus-ring"
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
