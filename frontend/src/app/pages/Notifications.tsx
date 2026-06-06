import { useMemo, useState } from "react";
import { useNavigate } from "react-router";
import {
  ArrowLeft,
  Heart,
  UserPlus,
  MessageCircle,
  Bell,
  MapPin,
  CheckCheck,
} from "lucide-react";
import { motion, AnimatePresence } from "motion/react";
import { useHoverConfig } from "../hooks/useHoverConfig";
import { usePlatform } from "../contexts/PlatformContext";
import { useNotificationsData } from "@/lib/hooks/use-notifications-data";
import { useDocumentTitle } from "@/lib/hooks/use-document-title";
import { goBack } from "@/lib/utils/navigation";
import { normalizeAvatar, formatRelativeTime } from "@/lib/utils";
import { ListSkeleton } from "../components/ui/skeleton";
import type { NotificationItem } from "@/lib/types/api";

type DisplayNotifType = "like" | "follow" | "comment" | "nearby" | "system";

const TYPE_CONFIG: Record<
  DisplayNotifType,
  { icon: React.ElementType; color: string; label: string }
> = {
  like: { icon: Heart, color: "#EF4444", label: "点赞" },
  follow: { icon: UserPlus, color: "#6366f1", label: "关注" },
  comment: { icon: MessageCircle, color: "#3B82F6", label: "评论" },
  nearby: { icon: MapPin, color: "#22c55e", label: "附近" },
  system: { icon: Bell, color: "#F59E0B", label: "系统" },
};

function getNotifDisplay(notif: NotificationItem): {
  type: DisplayNotifType;
  name: string;
  avatar: string;
  text: string;
  postImage?: string;
} {
  const p = notif.payload;
  switch (p.type) {
    case "new_follower":
      return {
        type: "follow",
        name: p.actor.nickname,
        avatar: p.actor.avatar_url ?? "",
        text: "开始关注了你",
      };
    case "mutual_follow":
      return {
        type: "follow",
        name: p.actor.nickname,
        avatar: p.actor.avatar_url ?? "",
        text: "与你互相关注了",
      };
    case "moment_liked":
      return {
        type: "like",
        name: p.actor.nickname,
        avatar: p.actor.avatar_url ?? "",
        text: `赞了你的动态「${p.moment_summary.description_preview}」`,
        postImage: p.moment_summary.cover_url,
      };
    case "moment_commented":
      return {
        type: "comment",
        name: p.actor.nickname,
        avatar: p.actor.avatar_url ?? "",
        text: `评论了你的动态：「${p.comment_preview}」`,
        postImage: p.moment_summary.cover_url,
      };
    case "comment_liked":
      return {
        type: "like",
        name: p.actor.nickname,
        avatar: p.actor.avatar_url ?? "",
        text: `赞了你的评论：「${p.comment_preview}」`,
        postImage: p.moment_summary.cover_url,
      };
    case "comment_reply":
      return {
        type: "comment",
        name: p.actor.nickname,
        avatar: p.actor.avatar_url ?? "",
        text: `回复了你：「${p.reply_content_preview}」`,
        postImage: p.moment_summary.cover_url,
      };
    case "mentioned":
      return {
        type: "comment",
        name: p.actor.nickname,
        avatar: p.actor.avatar_url ?? "",
        text: "在动态中提到了你",
        postImage: p.moment_summary.cover_url,
      };
    case "group_invite_received":
      return {
        type: "follow",
        name: p.actor.nickname,
        avatar: p.actor.avatar_url ?? "",
        text: `邀请你加入「${p.group_name}」`,
      };
    case "group_invite_accepted":
      return {
        type: "follow",
        name: p.user.nickname,
        avatar: p.user.avatar_url ?? "",
        text: `接受了「${p.group_name}」的邀请`,
      };
    case "group_application_received":
      return {
        type: "follow",
        name: p.applicant.nickname,
        avatar: p.applicant.avatar_url ?? "",
        text: `申请加入「${p.group_name}」`,
      };
    case "group_application_accepted":
      return {
        type: "follow",
        name: p.group_name,
        avatar: "",
        text: "你的加群申请已通过",
      };
    case "system_announcement":
      return { type: "system", name: "系统通知", avatar: "", text: "系统公告" };
    default:
      return {
        type: "system",
        name: "通知",
        avatar: "",
        text: "你有一条新通知",
      };
  }
}

const FILTER_TABS = [
  { id: "all", label: "全部" },
  { id: "like", label: "点赞" },
  { id: "follow", label: "关注" },
  { id: "comment", label: "评论" },
  { id: "nearby", label: "附近" },
];

export function Notifications() {
  const navigate = useNavigate();
  const { platform } = usePlatform();
  const {
    notifications,
    unreadCount,
    loading,
    error,
    hasMore,
    loadMore,
    refresh,
    markRead,
    markAllRead,
  } = useNotificationsData();
  const [filter, setFilter] = useState("all");
  useDocumentTitle("通知");
  const isWide = platform !== "mobile";

  const filtered = notifications.filter((n) => {
    if (filter === "all") return true;
    const display = getNotifDisplay(n);
    return display.type === filter;
  });

  const unreadGroup = filtered.filter((n) => !n.is_read);
  const readGroup = filtered.filter((n) => n.is_read);
  const groupedStats = useMemo(() => {
    return notifications.reduce(
      (acc, notif) => {
        const display = getNotifDisplay(notif);
        acc[display.type] += 1;
        return acc;
      },
      { like: 0, follow: 0, comment: 0, nearby: 0, system: 0 } satisfies Record<
        DisplayNotifType,
        number
      >,
    );
  }, [notifications]);

  return (
    <>
      <div
        className="sticky top-0 z-10 px-5 pt-3 pb-0"
        style={{
          backgroundColor: "var(--neu-bg)",
          borderBottom: "1px solid var(--neu-border)",
        }}
      >
        <div className={isWide ? "mx-auto w-full max-w-[1040px]" : ""}>
          <div className="flex items-center justify-between mb-3">
            <div className="flex items-center gap-3">
              <motion.button
                whileTap={{
                  scale: 0.88,
                  transition: { type: "spring", stiffness: 600, damping: 28 },
                }}
                whileHover={{
                  scale: 1.14,
                  transition: { type: "spring", stiffness: 400, damping: 16 },
                }}
                onClick={() => goBack(navigate)}
                aria-label="返回"
                className="flex items-center justify-center rounded-full outline-none neu-circle-hover neu-focus-ring"
                style={{
                  width: 44,
                  height: 44,
                  background: "var(--neu-bg)",
                  boxShadow:
                    "4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light)",
                  border: "1px solid var(--neu-border)",
                }}
              >
                <ArrowLeft
                  style={{
                    width: 15,
                    height: 15,
                    color: "var(--neu-text-muted)",
                  }}
                  strokeWidth={1.5}
                />
              </motion.button>
              <div>
                <h1 style={{ fontSize: 20, color: "var(--neu-text)" }}>通知</h1>
                <p
                  style={{
                    fontSize: 12,
                    color: "var(--neu-text-muted)",
                    marginTop: 1,
                  }}
                >
                  {unreadCount > 0 ? `${unreadCount} 条未读` : "全部已处理"}
                </p>
              </div>
            </div>
            {unreadCount > 0 && (
              <motion.button
                whileTap={{ scale: 0.93 }}
                onClick={markAllRead}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-full outline-none neu-focus-ring"
                style={{
                  background: "var(--neu-bg)",
                  border: "1px solid var(--neu-border)",
                  boxShadow:
                    "3px 3px 8px var(--neu-shadow-dark), -3px -3px 8px var(--neu-shadow-light)",
                  fontSize: 12,
                  color: "var(--neu-accent-primary)",
                }}
              >
                <CheckCheck style={{ width: 13, height: 13 }} strokeWidth={2} />
                全部已读
              </motion.button>
            )}
          </div>

          <div className="flex overflow-x-auto no-scrollbar pb-3 gap-2">
            {FILTER_TABS.map((t) => (
              <motion.button
                key={t.id}
                whileTap={{ scale: 0.94 }}
                onClick={() => setFilter(t.id)}
                className="flex-shrink-0 px-3.5 py-1.5 rounded-full outline-none transition-all neu-focus-ring"
                style={
                  filter === t.id
                    ? {
                        background: "var(--neu-accent-primary)",
                        color: "var(--text-on-accent, white)",
                        fontSize: 12,
                        boxShadow: "0 3px 10px var(--neu-glow)",
                        border: "none",
                      }
                    : {
                        background: "var(--neu-bg)",
                        color: "var(--neu-text-muted)",
                        fontSize: 12,
                        border: "1px solid var(--neu-border)",
                        boxShadow:
                          "3px 3px 6px var(--neu-shadow-dark), -3px -3px 6px var(--neu-shadow-light)",
                      }
                }
              >
                {t.label}
              </motion.button>
            ))}
          </div>
        </div>
      </div>

      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        className={
          isWide
            ? "mx-auto flex w-full max-w-[1040px] gap-6 px-5 pt-4 pb-8"
            : "flex flex-col px-5 pt-3 pb-6 gap-0"
        }
      >
        <div className={isWide ? "min-w-0 flex-1 max-w-[720px]" : ""}>
          {loading ? (
            <ListSkeleton rows={6} />
          ) : error && notifications.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-20 gap-3">
              <p style={{ fontSize: 14, color: "var(--neu-text-muted)" }}>
                {error}
              </p>
              <motion.button
                whileTap={{ scale: 0.94 }}
                onClick={refresh}
                className="px-4 py-2 rounded-full outline-none neu-focus-ring"
                style={{
                  background: "var(--neu-accent-primary)",
                  color: "var(--text-on-accent, white)",
                  fontSize: 13,
                }}
              >
                重试
              </motion.button>
            </div>
          ) : (
            <>
              {unreadGroup.length > 0 && (
                <>
                  <p
                    className="uppercase mb-2"
                    style={{
                      fontSize: 10,
                      letterSpacing: "0.14em",
                      color: "var(--neu-text-muted)",
                    }}
                  >
                    新消息
                  </p>
                  {unreadGroup.map((n) => (
                    <NotifRow
                      key={n.notification_id}
                      notif={n}
                      onRead={(id) => markRead([id])}
                    />
                  ))}
                  {readGroup.length > 0 && (
                    <p
                      className="uppercase mt-4 mb-2"
                      style={{
                        fontSize: 10,
                        letterSpacing: "0.14em",
                        color: "var(--neu-text-muted)",
                      }}
                    >
                      较早
                    </p>
                  )}
                </>
              )}

              <AnimatePresence>
                {readGroup.map((n) => (
                  <NotifRow
                    key={n.notification_id}
                    notif={n}
                    onRead={(id) => markRead([id])}
                  />
                ))}
              </AnimatePresence>

              {hasMore && (
                <div className="flex justify-center mt-4">
                  <motion.button
                    whileTap={{ scale: 0.93 }}
                    onClick={loadMore}
                    className="px-4 py-2 rounded-full outline-none neu-focus-ring"
                    style={{
                      fontSize: 12,
                      color: "var(--neu-accent-primary)",
                      border: "1px solid var(--neu-border)",
                    }}
                  >
                    加载更多
                  </motion.button>
                </div>
              )}

              {filtered.length === 0 && (
                <div className="flex flex-col items-center justify-center py-20 gap-3">
                  <div
                    className="flex items-center justify-center rounded-full"
                    style={{
                      width: 64,
                      height: 64,
                      boxShadow:
                        "6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light)",
                      border: "1px solid var(--neu-border)",
                    }}
                  >
                    <Bell
                      style={{
                        width: 24,
                        height: 24,
                        color: "var(--neu-text-muted)",
                        opacity: 0.5,
                      }}
                      strokeWidth={1.25}
                    />
                  </div>
                  <p style={{ fontSize: 15, color: "var(--neu-text)" }}>
                    暂无通知
                  </p>
                </div>
              )}
            </>
          )}
        </div>

        {isWide && (
          <aside className="hidden md:flex w-[280px] flex-col gap-4">
            <div className="neu-surface-flat rounded-[22px] p-4">
              <p className="neu-eyebrow">通知概览</p>
              <div className="grid grid-cols-2 gap-3 mt-3">
                {[
                  {
                    label: "全部",
                    value: notifications.length,
                    color: "var(--neu-text)",
                  },
                  {
                    label: "未读",
                    value: unreadCount,
                    color: "var(--neu-accent-primary)",
                  },
                  {
                    label: "当前筛选",
                    value:
                      FILTER_TABS.find((item) => item.id === filter)?.label ??
                      "全部",
                    color: "var(--neu-text)",
                  },
                  {
                    label: "已读",
                    value: notifications.length - unreadCount,
                    color: "var(--neu-text-muted)",
                  },
                ].map((item) => (
                  <div
                    key={item.label}
                    className="rounded-[16px] px-3 py-3"
                    style={{ background: "var(--neu-border)" }}
                  >
                    <p
                      style={{
                        fontSize: 10,
                        color: "var(--neu-text-muted)",
                        letterSpacing: "0.08em",
                        textTransform: "uppercase",
                      }}
                    >
                      {item.label}
                    </p>
                    <p
                      style={{
                        fontSize: typeof item.value === "number" ? 22 : 14,
                        color: item.color,
                        marginTop: 6,
                        fontWeight: 600,
                        lineHeight: 1.1,
                      }}
                    >
                      {item.value}
                    </p>
                  </div>
                ))}
              </div>
            </div>

            <div className="neu-surface-flat rounded-[22px] p-4">
              <p className="neu-eyebrow">类型分布</p>
              <div className="flex flex-col gap-2.5 mt-3">
                {[
                  { label: "点赞", value: groupedStats.like, color: "var(--neu-danger)" },
                  {
                    label: "关注",
                    value: groupedStats.follow,
                    color: "#6366f1",
                  },
                  {
                    label: "评论",
                    value: groupedStats.comment,
                    color: "#3B82F6",
                  },
                  {
                    label: "系统",
                    value: groupedStats.system + groupedStats.nearby,
                    color: "var(--neu-warn)",
                  },
                ].map((item) => (
                  <div
                    key={item.label}
                    className="flex items-center justify-between rounded-[16px] px-3 py-2.5"
                    style={{ background: "var(--neu-border)" }}
                  >
                    <div className="flex items-center gap-2">
                      <span
                        className="rounded-full"
                        style={{ width: 8, height: 8, background: item.color }}
                      />
                      <span style={{ fontSize: 13, color: "var(--neu-text)" }}>
                        {item.label}
                      </span>
                    </div>
                    <span
                      style={{ fontSize: 12, color: "var(--neu-text-muted)" }}
                    >
                      {item.value}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          </aside>
        )}
      </motion.div>
    </>
  );
}

function NotifRow({
  notif,
  onRead,
}: {
  notif: NotificationItem;
  onRead: (id: string) => void;
}) {
  const display = getNotifDisplay(notif);
  const cfg = TYPE_CONFIG[display.type];
  const Icon = cfg.icon;
  const hover = useHoverConfig();

  return (
    <motion.button
      layout
      initial={{ opacity: 0, x: -12 }}
      animate={{ opacity: 1, x: 0 }}
      exit={{ opacity: 0, x: 12, height: 0, marginBottom: 0 }}
      transition={{ duration: 0.22 }}
      onClick={() => onRead(notif.notification_id)}
      whileTap={hover.tap}
      whileHover={hover.row}
      className="flex w-full items-center gap-3.5 py-3.5 cursor-pointer rounded-[18px] mb-1.5 px-3 neu-row-hover neu-focus-ring text-left outline-none"
      style={{
        background: notif.is_read ? "transparent" : `${cfg.color}07`,
        border: notif.is_read
          ? "1px solid transparent"
          : `1px solid ${cfg.color}18`,
      }}
    >
      <div className="relative flex-shrink-0">
        {display.avatar ? (
          <div
            className="rounded-full overflow-hidden"
            style={{ width: 44, height: 44 }}
          >
            <img
              src={normalizeAvatar(display.avatar)}
              alt={display.name}
              className="w-full h-full object-cover"
            />
          </div>
        ) : (
          <div
            className="rounded-full flex items-center justify-center"
            style={{ width: 44, height: 44, background: `${cfg.color}18` }}
          >
            <Icon
              style={{ width: 18, height: 18, color: cfg.color }}
              strokeWidth={1.75}
            />
          </div>
        )}
        {display.avatar && (
          <div
            className="absolute -bottom-0.5 -right-0.5 flex items-center justify-center rounded-full"
            style={{
              width: 18,
              height: 18,
              background: cfg.color,
              border: "2px solid var(--neu-bg)",
            }}
          >
            <Icon
              style={{ width: 9, height: 9, color: "white" }}
              strokeWidth={2.5}
            />
          </div>
        )}
      </div>

      <div className="flex-1 min-w-0">
        <p
          className="truncate"
          style={{ fontSize: 13, color: "var(--neu-text)" }}
        >
          <span style={{ fontWeight: notif.is_read ? 400 : 500 }}>
            {display.name}
          </span>{" "}
          {display.text}
        </p>
        <p
          style={{ fontSize: 11, color: "var(--neu-text-muted)", marginTop: 2 }}
        >
          {formatRelativeTime(notif.created_at)}
        </p>
      </div>

      <div className="flex items-center gap-2 flex-shrink-0">
        {display.postImage && (
          <div
            className="rounded-[10px] overflow-hidden"
            style={{ width: 38, height: 38 }}
          >
            <img
              src={normalizeAvatar(display.postImage)}
              alt=""
              className="w-full h-full object-cover"
              loading="lazy"
            />
          </div>
        )}
        {!notif.is_read && (
          <div
            className="rounded-full"
            style={{ width: 8, height: 8, background: cfg.color }}
          />
        )}
      </div>
    </motion.button>
  );
}
