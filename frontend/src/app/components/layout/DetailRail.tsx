import React, { useState } from "react";
import { motion } from "motion/react";
import { ChevronRight, MapPin } from "lucide-react";
import { useNavigate } from "react-router";
import { usePlatform } from "../../contexts/PlatformContext";
import { useHoverConfig } from "../../hooks/useHoverConfig";
import { ContextRail } from "./ContextRail";
import { ROUTES } from "../../lib/routes";

/* ════════════════════════════════════════════════════════════════════
   DetailRail — 桌面端详情页右侧"上下文陪伴栏"

   设计意图：详情页（PostDetail / UserProfile）在桌面
   宽度下，主体内容居中（max-width ≈ 640px），右侧大片空白。
   通过 320px 的固定宽度陪伴栏填充语义相关上下文，
   减少跳页、增加把玩感，同时和 TabletRail 共享视觉语言：
   - neu-surface-flat 卡片
   - neu-eyebrow 小节标题
   - 浅色填充 + 单像素分隔
════════════════════════════════════════════════════════════════════ */

/* ── DesktopDetailSplit — 容器：左主体 + 右栏 ─────────────────────── */
export function DesktopDetailSplit({
  main,
  rail,
}: {
  main: React.ReactNode;
  rail: React.ReactNode;
}) {
  const { platform } = usePlatform();
  if (platform !== "desktop") return <>{main}</>;
  return (
    <div className="flex h-full w-full">
      <div className="flex-1 min-w-0 flex flex-col">{main}</div>
      <ContextRail attached>{rail}</ContextRail>
    </div>
  );
}

/* ── RailSection — 通用小节包装 ───────────────────────────────────── */
export function RailSection({
  title,
  action,
  onAction,
  children,
}: {
  title: string;
  action?: string;
  onAction?: () => void;
  children: React.ReactNode;
}) {
  return (
    <div className="neu-surface-flat rounded-[18px] p-4">
      <div className="flex items-end justify-between mb-3">
        <span className="neu-eyebrow">{title}</span>
        {action && onAction && (
          <motion.button
            whileTap={{ scale: 0.94 }}
            onClick={onAction}
            className="flex items-center gap-0.5 outline-none"
            style={{ color: "var(--neu-accent-primary)" }}
          >
            <span style={{ fontSize: 11 }}>{action}</span>
            <ChevronRight style={{ width: 12, height: 12 }} strokeWidth={1.5} />
          </motion.button>
        )}
      </div>
      {children}
    </div>
  );
}

/* ── RailUserRow — 复用的"小头像 + 名称 + 副标 + 操作"行 ─────────── */
export function RailUserRow({
  avatar,
  name,
  sub,
  trailing,
  onClick,
}: {
  avatar: string;
  name: string;
  sub?: string;
  trailing?: React.ReactNode;
  onClick?: () => void;
}) {
  return (
    <motion.div
      whileTap={onClick ? { scale: 0.985 } : undefined}
      onClick={onClick}
      className={`flex items-center gap-2.5 ${onClick ? "cursor-pointer" : ""}`}
    >
      <div
        className="rounded-full overflow-hidden flex-shrink-0"
        style={{ width: 34, height: 34, border: "1px solid var(--neu-border)" }}
      >
        <img src={avatar} alt={name} className="w-full h-full object-cover" />
      </div>
      <div className="flex-1 min-w-0">
        <p style={{ fontSize: 13, color: "var(--neu-text)" }} className="truncate">{name}</p>
        {sub && (
          <p style={{ fontSize: 11, color: "var(--neu-text-muted)", marginTop: 1 }} className="truncate">
            {sub}
          </p>
        )}
      </div>
      {trailing}
    </motion.div>
  );
}

/* ── RailPostThumb — 小尺寸帖子缩略 ───────────────────────────────── */
export function RailPostThumb({
  image,
  title,
  meta,
  onClick,
}: {
  image: string;
  title: string;
  meta: string;
  onClick?: () => void;
}) {
  return (
    <motion.div
      whileTap={onClick ? { scale: 0.985 } : undefined}
      onClick={onClick}
      className={`flex items-center gap-2.5 ${onClick ? "cursor-pointer" : ""}`}
    >
      <div
        className="rounded-[10px] overflow-hidden flex-shrink-0"
        style={{ width: 46, height: 46 }}
      >
        <img src={image} alt={title} className="w-full h-full object-cover" />
      </div>
      <div className="flex-1 min-w-0">
        <p style={{ fontSize: 12, color: "var(--neu-text)" }} className="truncate">{title}</p>
        <p style={{ fontSize: 10, color: "var(--neu-text-muted)", marginTop: 2 }} className="truncate">
          {meta}
        </p>
      </div>
      <ChevronRight
        style={{ width: 12, height: 12, color: "var(--neu-text-muted)", flexShrink: 0 }}
        strokeWidth={1.5}
      />
    </motion.div>
  );
}

/* ── RailLocationMini — 位置 mini 卡 ───────────────────────────────── */
export function RailLocationMini({ name, desc }: { name: string; desc?: string }) {
  const navigate = useNavigate();
  const [hovered, setHovered] = useState(false);
  const hover = useHoverConfig();
  return (
    <motion.button
      whileTap={{ scale: 0.985 }}
      onClick={() => navigate(ROUTES.map)}
      onHoverStart={() => setHovered(true)}
      onHoverEnd={() => setHovered(false)}
      className="flex items-center gap-2.5 w-full outline-none text-left rounded-[12px] p-2 -m-2"
      style={{ background: hovered ? hover.bgHover : "transparent", transition: "background 0.18s ease" }}
    >
      <div
        className="flex items-center justify-center rounded-full flex-shrink-0"
        style={{ width: 32, height: 32, background: "var(--neu-accent-soft)" }}
      >
        <MapPin style={{ width: 13, height: 13, color: "var(--neu-accent-primary)" }} strokeWidth={1.75} />
      </div>
      <div className="flex-1 min-w-0">
        <p style={{ fontSize: 12, color: "var(--neu-text)" }} className="truncate">{name}</p>
        {desc && (
          <p style={{ fontSize: 10, color: "var(--neu-text-muted)", marginTop: 1 }} className="truncate">
            {desc}
          </p>
        )}
      </div>
      <ChevronRight
        style={{ width: 12, height: 12, color: "var(--neu-text-muted)", flexShrink: 0 }}
        strokeWidth={1.5}
      />
    </motion.button>
  );
}
