import { motion } from "motion/react";
import {
  Flame,
  Eye,
  Trophy,
  UserPlus,
  MapPin,
  Sparkles,
  Music,
  ChevronRight,
  Sun,
  Cloud,
  Heart,
} from "lucide-react";

/* ════════════════════════════════════════════════════════════════════
   TabletRail — 平板端右侧"陪伴栏"

   设计意图：在 tablet 横屏时，主体内容居中（max-width: 560px），
   右侧出现 ~300px 的伴生栏，容纳那些"看着上瘾"的小卡片：
   - 连续签到（有节奏的视觉钩子 + gentle gamification）
   - 最近访客（社交好奇心驱动）
   - 今日心情打卡（轻度交互 + 个性化）
   - 下一个成就（进度条诱导）
   - 你可能认识（horizontal pull）
   - 此刻附近（实时感）

   所有卡片用 neu-surface-flat（无沉重 elevation），避免一栏全是漂浮卡。
   每张卡都有 eyebrow + focal + 次要文案三层信息，密度高但不挤。
════════════════════════════════════════════════════════════════════ */

/* ───────────────────────────── 容器 ───────────────────────────── */

export function TabletRail({ children }: { children: React.ReactNode }) {
  return (
    <aside className="flex flex-col gap-3" style={{ width: "100%" }}>
      {children}
    </aside>
  );
}

/* ─────────────────────── 单元：连续签到 ─────────────────────── */

const STREAK_DAYS = [
  { label: "一", done: true },
  { label: "二", done: true },
  { label: "三", done: true },
  { label: "四", done: true },
  { label: "五", done: true },
  { label: "六", done: true },
  { label: "日", done: false, today: true },
];

export function StreakCard() {
  return (
    <motion.div
      whileHover={{ y: -1 }}
      transition={{ duration: 0.18 }}
      className="neu-surface-flat rounded-[18px] p-4"
    >
      <div className="flex items-center justify-between mb-3">
        <span className="neu-eyebrow">连续签到</span>
        <Flame
          style={{ width: 14, height: 14, color: "var(--neu-accent-primary)" }}
          strokeWidth={1.75}
        />
      </div>
      <div className="flex items-baseline gap-1.5 mb-3.5">
        <span
          style={{
            fontSize: 32,
            color: "var(--neu-text)",
            fontWeight: 600,
            lineHeight: 1,
          }}
        >
          27
        </span>
        <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>天</span>
      </div>
      <div className="flex gap-1 mb-2.5">
        {STREAK_DAYS.map((d, i) => (
          <div key={i} className="flex-1 flex flex-col items-center gap-1">
            <span
              style={{
                fontSize: 9,
                color: "var(--neu-text-muted)",
                opacity: d.today ? 1 : 0.7,
              }}
            >
              {d.label}
            </span>
            <div
              style={{
                width: "100%",
                height: 22,
                borderRadius: 5,
                background: d.done
                  ? "var(--neu-accent-primary)"
                  : d.today
                    ? "var(--neu-accent-soft)"
                    : "var(--neu-border)",
                opacity: d.done ? 0.88 : 1,
                border: d.today
                  ? "1.5px dashed var(--neu-accent-soft-border)"
                  : "none",
              }}
            />
          </div>
        ))}
      </div>
      <p
        style={{
          fontSize: 11,
          color: "var(--neu-text-muted)",
          lineHeight: 1.5,
        }}
      >
        明天再来就满 28 天啦 🌱
      </p>
    </motion.div>
  );
}

/* ─────────────────────── 单元：最近访客 ─────────────────────── */

const VISITORS = [
  { name: "Yuki", avatar: "https://i.pravatar.cc/64?img=23", min: 4 },
  { name: "Lin", avatar: "https://i.pravatar.cc/64?img=47", min: 12 },
  { name: "Atlas", avatar: "https://i.pravatar.cc/64?img=12", min: 23 },
  { name: "Niko", avatar: "https://i.pravatar.cc/64?img=33", min: 41 },
  { name: "Sora", avatar: "https://i.pravatar.cc/64?img=54", min: 68 },
];

export function VisitorsCard() {
  return (
    <motion.button
      whileHover={{ y: -1 }}
      whileTap={{ scale: 0.98 }}
      className="neu-surface-flat rounded-[18px] p-4 text-left outline-none w-full"
    >
      <div className="flex items-center justify-between mb-3">
        <span className="neu-eyebrow">今天看过你</span>
        <Eye
          style={{ width: 14, height: 14, color: "var(--neu-text-muted)" }}
          strokeWidth={1.75}
        />
      </div>
      <div className="flex items-center mb-2.5">
        <div className="flex" style={{ marginLeft: 0 }}>
          {VISITORS.slice(0, 4).map((v, i) => (
            <div
              key={v.name}
              className="rounded-full overflow-hidden"
              style={{
                width: 30,
                height: 30,
                marginLeft: i === 0 ? 0 : -8,
                border: "2px solid var(--neu-bg)",
                zIndex: 10 - i,
              }}
            >
              <img
                src={v.avatar}
                alt={v.name}
                className="w-full h-full object-cover"
              />
            </div>
          ))}
        </div>
        <div className="ml-auto flex items-baseline gap-1">
          <span
            style={{
              fontSize: 22,
              color: "var(--neu-text)",
              fontWeight: 600,
              lineHeight: 1,
            }}
          >
            18
          </span>
          <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>
            位
          </span>
        </div>
      </div>
      <div className="flex items-center justify-between">
        <p style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>
          {VISITORS[0].name} 刚刚 · 还有 {VISITORS.length - 1} 位
        </p>
        <ChevronRight
          style={{ width: 12, height: 12, color: "var(--neu-text-muted)" }}
          strokeWidth={1.5}
        />
      </div>
    </motion.button>
  );
}

/* ─────────────────────── 单元：今日心情 ─────────────────────── */

const MOODS = [
  { emoji: "🌤", label: "舒服" },
  { emoji: "🌊", label: "平静" },
  { emoji: "🔥", label: "热血" },
  { emoji: "🌙", label: "想独处" },
  { emoji: "🍵", label: "慢一点" },
];

export function MoodCheckinCard() {
  return (
    <div className="neu-surface-flat rounded-[18px] p-4">
      <div className="flex items-center justify-between mb-3">
        <span className="neu-eyebrow">今日心情</span>
        <Sparkles
          style={{ width: 14, height: 14, color: "var(--neu-accent-primary)" }}
          strokeWidth={1.75}
        />
      </div>
      <p
        style={{
          fontSize: 13,
          color: "var(--neu-text)",
          marginBottom: 12,
          lineHeight: 1.5,
        }}
      >
        今天的你是怎样的？
      </p>
      <div className="flex gap-1.5 flex-wrap">
        {MOODS.map((m) => (
          <motion.button
            key={m.label}
            whileTap={{ scale: 0.94 }}
            whileHover={{ y: -1 }}
            className="flex items-center gap-1 px-2.5 py-1.5 rounded-full outline-none neu-surface-ghost"
            style={{ borderColor: "var(--neu-border)" }}
          >
            <span style={{ fontSize: 14 }}>{m.emoji}</span>
            <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>
              {m.label}
            </span>
          </motion.button>
        ))}
      </div>
    </div>
  );
}

/* ─────────────────────── 单元：成就进度 ─────────────────────── */

export function AchievementCard() {
  const progress = 0.72;
  return (
    <div className="neu-surface-flat rounded-[18px] p-4">
      <div className="flex items-center justify-between mb-2.5">
        <span className="neu-eyebrow">下一枚徽章</span>
        <Trophy
          style={{ width: 14, height: 14, color: "var(--neu-accent-primary)" }}
          strokeWidth={1.75}
        />
      </div>
      <div className="flex items-center gap-2.5 mb-3">
        <div
          className="flex items-center justify-center rounded-[12px] flex-shrink-0"
          style={{
            width: 38,
            height: 38,
            background: "var(--neu-accent-soft)",
            border: "1px solid var(--neu-accent-soft-border)",
            fontSize: 20,
          }}
        >
          🌟
        </div>
        <div className="flex-1 min-w-0">
          <p
            style={{ fontSize: 13, color: "var(--neu-text)", lineHeight: 1.3 }}
          >
            城市漫游者
          </p>
          <p
            style={{
              fontSize: 11,
              color: "var(--neu-text-muted)",
              marginTop: 1,
            }}
          >
            已探索 27 / 38 个地点
          </p>
        </div>
      </div>
      <div
        style={{
          height: 5,
          borderRadius: 999,
          background: "var(--neu-border)",
          overflow: "hidden",
        }}
      >
        <motion.div
          initial={{ width: 0 }}
          animate={{ width: `${progress * 100}%` }}
          transition={{ duration: 0.9, ease: [0.22, 1, 0.36, 1] }}
          style={{
            height: "100%",
            background: "var(--neu-accent-primary)",
            borderRadius: 999,
          }}
        />
      </div>
      <p style={{ fontSize: 10, color: "var(--neu-text-muted)", marginTop: 6 }}>
        再去 11 个新地方就解锁
      </p>
    </div>
  );
}

/* ─────────────────────── 单元：你可能认识 ─────────────────────── */

const SUGGESTED = [
  {
    name: "Nori",
    handle: "@nori.k",
    avatar: "https://i.pravatar.cc/64?img=15",
    mutual: 8,
  },
  {
    name: "Mei",
    handle: "@meimei",
    avatar: "https://i.pravatar.cc/64?img=28",
    mutual: 5,
  },
  {
    name: "Kai",
    handle: "@kai.dev",
    avatar: "https://i.pravatar.cc/64?img=41",
    mutual: 3,
  },
];

export function SuggestedFollowsCard() {
  return (
    <div className="neu-surface-flat rounded-[18px] p-4">
      <div className="flex items-center justify-between mb-3">
        <span className="neu-eyebrow">你可能认识</span>
        <UserPlus
          style={{ width: 14, height: 14, color: "var(--neu-text-muted)" }}
          strokeWidth={1.75}
        />
      </div>
      <div className="flex flex-col gap-2.5">
        {SUGGESTED.map((s) => (
          <div key={s.name} className="flex items-center gap-2.5">
            <div
              className="rounded-full overflow-hidden flex-shrink-0"
              style={{ width: 32, height: 32 }}
            >
              <img
                src={s.avatar}
                alt={s.name}
                className="w-full h-full object-cover"
              />
            </div>
            <div className="flex-1 min-w-0">
              <p
                style={{
                  fontSize: 12,
                  color: "var(--neu-text)",
                  lineHeight: 1.2,
                }}
              >
                {s.name}
              </p>
              <p
                style={{
                  fontSize: 10,
                  color: "var(--neu-text-muted)",
                  marginTop: 1,
                }}
              >
                {s.mutual} 位共同好友
              </p>
            </div>
            <motion.button
              whileTap={{ scale: 0.94 }}
              whileHover={{ y: -1 }}
              className="px-2.5 py-1 rounded-full outline-none"
              style={{
                background: "var(--neu-accent-soft)",
                border: "1px solid var(--neu-accent-soft-border)",
                color: "var(--neu-accent-primary)",
                fontSize: 10,
                fontWeight: 500,
              }}
            >
              关注
            </motion.button>
          </div>
        ))}
      </div>
    </div>
  );
}

/* ─────────────────────── 单元：此刻附近 ─────────────────────── */

export function NearbyPulseCard() {
  return (
    <motion.button
      whileHover={{ y: -1 }}
      whileTap={{ scale: 0.98 }}
      className="neu-surface-flat rounded-[18px] p-4 text-left outline-none w-full overflow-hidden relative"
    >
      <div className="flex items-center justify-between mb-2.5">
        <span className="neu-eyebrow">此刻附近</span>
        <div className="flex items-center gap-1.5">
          <span
            style={{
              width: 6,
              height: 6,
              borderRadius: 999,
              background: "#22c55e",
              boxShadow: "0 0 0 3px rgba(34,197,94,0.18)",
            }}
          />
          <span style={{ fontSize: 10, color: "var(--neu-text-muted)" }}>
            实时
          </span>
        </div>
      </div>
      <div className="flex items-center gap-2 mb-2">
        <MapPin
          style={{ width: 13, height: 13, color: "var(--neu-accent-primary)" }}
          strokeWidth={1.75}
        />
        <span style={{ fontSize: 13, color: "var(--neu-text)" }}>
          云龙湖周边
        </span>
      </div>
      <div className="flex items-baseline gap-1.5">
        <span
          style={{
            fontSize: 26,
            color: "var(--neu-text)",
            fontWeight: 600,
            lineHeight: 1,
          }}
        >
          142
        </span>
        <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>
          人在线 · 比昨天 +18
        </span>
      </div>
      {/* sparkline 装饰 */}
      <svg
        width="100%"
        height="28"
        viewBox="0 0 200 28"
        preserveAspectRatio="none"
        style={{ marginTop: 8, opacity: 0.55 }}
      >
        <path
          d="M0,20 L20,16 L40,18 L60,12 L80,14 L100,8 L120,11 L140,6 L160,9 L180,4 L200,7"
          fill="none"
          stroke="var(--neu-accent-primary)"
          strokeWidth="1.5"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
      </svg>
    </motion.button>
  );
}

/* ─────────────────────── 单元：今日精选 ─────────────────────── */

export function DailyPickCard() {
  return (
    <motion.button
      whileHover={{ y: -1 }}
      whileTap={{ scale: 0.98 }}
      className="neu-surface-flat rounded-[18px] overflow-hidden text-left outline-none w-full"
    >
      <div
        className="relative"
        style={{
          height: 110,
          background:
            "linear-gradient(135deg, var(--neu-accent-soft) 0%, var(--neu-border) 100%)",
        }}
      >
        <img
          src="https://images.unsplash.com/photo-1502082553048-f009c37129b9?w=400&q=80"
          alt="今日精选"
          className="w-full h-full object-cover"
          style={{ opacity: 0.92 }}
        />
        <div
          className="absolute top-2 left-2 px-2 py-0.5 rounded-full backdrop-blur-sm flex items-center gap-1"
          style={{ background: "rgba(0,0,0,0.32)" }}
        >
          <Sparkles
            style={{ width: 9, height: 9, color: "white" }}
            strokeWidth={2}
          />
          <span
            style={{ fontSize: 9, color: "white", letterSpacing: "0.05em" }}
          >
            今日精选
          </span>
        </div>
      </div>
      <div className="p-3">
        <p style={{ fontSize: 12, color: "var(--neu-text)", lineHeight: 1.4 }}>
          雨后的旧书店，光线像被泡软了
        </p>
        <div className="flex items-center gap-2 mt-1.5">
          <Heart
            style={{ width: 11, height: 11, color: "var(--neu-text-muted)" }}
            strokeWidth={1.5}
          />
          <span style={{ fontSize: 10, color: "var(--neu-text-muted)" }}>
            1.2k
          </span>
          <span style={{ fontSize: 10, color: "var(--neu-text-muted)" }}>
            ·
          </span>
          <span style={{ fontSize: 10, color: "var(--neu-text-muted)" }}>
            来自 @lumi
          </span>
        </div>
      </div>
    </motion.button>
  );
}

/* ─────────────────────── 单元：今日氛围（天气+音乐） ─────────────────────── */

export function VibeCard() {
  return (
    <div className="neu-surface-flat rounded-[18px] p-4">
      <div className="flex items-center justify-between mb-3">
        <span className="neu-eyebrow">今日氛围</span>
        <Sun
          style={{ width: 14, height: 14, color: "var(--neu-accent-primary)" }}
          strokeWidth={1.75}
        />
      </div>
      <div className="flex items-center gap-3 mb-3">
        <div className="flex items-center gap-1.5">
          <Cloud
            style={{ width: 16, height: 16, color: "var(--neu-text-muted)" }}
            strokeWidth={1.5}
          />
          <span style={{ fontSize: 13, color: "var(--neu-text)" }}>
            多云 · 21°
          </span>
        </div>
        <span style={{ fontSize: 10, color: "var(--neu-text-muted)" }}>
          适合散步
        </span>
      </div>
      <div
        className="flex items-center gap-2.5 p-2 rounded-[12px]"
        style={{ background: "var(--neu-border)" }}
      >
        <div
          className="flex items-center justify-center rounded-[8px] flex-shrink-0"
          style={{
            width: 32,
            height: 32,
            background: "var(--neu-accent-primary)",
            opacity: 0.9,
          }}
        >
          <Music
            style={{
              width: 14,
              height: 14,
              color: "var(--text-on-accent, white)",
            }}
            strokeWidth={2}
          />
        </div>
        <div className="flex-1 min-w-0">
          <p
            style={{ fontSize: 11, color: "var(--neu-text)", lineHeight: 1.2 }}
            className="truncate"
          >
            漫长的告白
          </p>
          <p
            style={{
              fontSize: 9,
              color: "var(--neu-text-muted)",
              marginTop: 1,
            }}
          >
            陈鸿宇 · 单曲循环
          </p>
        </div>
      </div>
    </div>
  );
}
