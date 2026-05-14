import React, { useState } from "react";
import { useNavigate } from "react-router";
import { ArrowLeft, Heart, UserPlus, MessageCircle, Bell, MapPin, CheckCheck, Filter } from "lucide-react";
import { motion, AnimatePresence } from "motion/react";
import { useHoverConfig } from "../hooks/useHoverConfig";

/* ── Types & data ───────────────────────────────────────────────── */
type NotifType = "like" | "follow" | "comment" | "nearby" | "event";

interface Notif {
  id: number; type: NotifType; read: boolean; time: string;
  user: { name: string; avatar: string };
  text: string; postImage?: string;
}

const NOTIFS: Notif[] = [
  { id: 1,  type:"like",    read:false, time:"2分钟前",  user:{ name:"Liam",    avatar:"https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80" }, text:"赞了你的动态「植物园晨曦」", postImage:"https://images.unsplash.com/photo-1768776179834-93e6cafc6d97?w=120" },
  { id: 2,  type:"follow",  read:false, time:"8分钟前",  user:{ name:"Sophia",  avatar:"https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80" }, text:"开始关注了你" },
  { id: 3,  type:"comment", read:false, time:"15分钟前", user:{ name:"Marco",   avatar:"https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=80" }, text:"评论了你的动态：「太美了！我也想去」", postImage:"https://images.unsplash.com/photo-1768776179834-93e6cafc6d97?w=120" },
  { id: 4,  type:"nearby",  read:false, time:"32分钟前", user:{ name:"Aiko",    avatar:"https://images.unsplash.com/photo-1758613172074-7275c3455034?w=80" }, text:"在你附近 520m · 在线中" },
  { id: 5,  type:"event",   read:true,  time:"1小时前",  user:{ name:"活动助手", avatar:"" }, text:"「晨间咖啡会」明天早9点开始，别忘了！" },
  { id: 6,  type:"like",    read:true,  time:"2小时前",  user:{ name:"Noah",    avatar:"https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80" }, text:"赞了你的动态「城市夜光」", postImage:"https://images.unsplash.com/photo-1748737349771-e2b6605e5961?w=120" },
  { id: 7,  type:"follow",  read:true,  time:"3小时前",  user:{ name:"Isabella",avatar:"https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80" }, text:"开始关注了你" },
  { id: 8,  type:"comment", read:true,  time:"昨天",     user:{ name:"Kai",     avatar:"https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=80" }, text:"评论了你的动态：「请问地址方便分享吗？」", postImage:"https://images.unsplash.com/photo-1748737349771-e2b6605e5961?w=120" },
  { id: 9,  type:"nearby",  read:true,  time:"昨天",     user:{ name:"Elena",   avatar:"https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80" }, text:"在你附近 820m · 上次活跃：1小时前" },
  { id: 10, type:"event",   read:true,  time:"2天前",    user:{ name:"活动助手", avatar:"" }, text:"「街头音乐节」收到了你的报名！" },
];

const TYPE_CONFIG: Record<NotifType, { icon: React.ElementType; color: string; label: string }> = {
  like:    { icon: Heart,          color: "#EF4444", label: "点赞" },
  follow:  { icon: UserPlus,       color: "#6366f1", label: "关注" },
  comment: { icon: MessageCircle,  color: "#3B82F6", label: "评论" },
  nearby:  { icon: MapPin,         color: "#22c55e", label: "附近" },
  event:   { icon: Bell,           color: "#F59E0B", label: "活动" },
};

const FILTER_TABS = [
  { id: "all",     label: "全部" },
  { id: "like",    label: "点赞" },
  { id: "follow",  label: "关注" },
  { id: "comment", label: "评论" },
  { id: "nearby",  label: "附近" },
];

export function Notifications() {
  const navigate = useNavigate();
  const [notifs, setNotifs] = useState<Notif[]>(NOTIFS);
  const [filter, setFilter] = useState("all");

  const unread = notifs.filter(n => !n.read).length;

  const markAllRead = () => setNotifs(prev => prev.map(n => ({ ...n, read: true })));
  const markRead = (id: number) => setNotifs(prev => prev.map(n => n.id === id ? { ...n, read: true } : n));
  const remove = (id: number) => setNotifs(prev => prev.filter(n => n.id !== id));

  const filtered = notifs.filter(n => filter === "all" || n.type === filter);
  const unreadGroup = filtered.filter(n => !n.read);
  const readGroup   = filtered.filter(n => n.read);

  return (
    <>
      {/* Header */}
      <div className="sticky top-0 z-10 px-5 pt-3 pb-0"
        style={{ backgroundColor: "var(--neu-bg)", borderBottom: "1px solid var(--neu-border)" }}>
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-3">
            <motion.button whileTap={{ scale: 0.88, transition: { type: "spring", stiffness: 600, damping: 28 } }} whileHover={{ scale: 1.14, transition: { type: "spring", stiffness: 400, damping: 16 } }} onClick={() => navigate(-1)}
              className="flex items-center justify-center rounded-full outline-none neu-circle-hover"
              style={{ width: 36, height: 36, background: "var(--neu-bg)", boxShadow: "4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light)", border: "1px solid var(--neu-border)" }}>
              <ArrowLeft style={{ width: 15, height: 15, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
            </motion.button>
            <div>
              <h1 style={{ fontSize: 20, color: "var(--neu-text)" }}>通知</h1>
              {unread > 0 && <p style={{ fontSize: 12, color: "var(--neu-text-muted)", marginTop: 1 }}>{unread} 条未读</p>}
            </div>
          </div>
          {unread > 0 && (
            <motion.button whileTap={{ scale: 0.93 }} onClick={markAllRead}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-full outline-none"
              style={{ background: "var(--neu-bg)", border: "1px solid var(--neu-border)", boxShadow: "3px 3px 8px var(--neu-shadow-dark), -3px -3px 8px var(--neu-shadow-light)", fontSize: 12, color: "var(--neu-accent-primary)" }}>
              <CheckCheck style={{ width: 13, height: 13 }} strokeWidth={2} />
              全部已读
            </motion.button>
          )}
        </div>

        {/* Filter tabs */}
        <div className="flex overflow-x-auto no-scrollbar pb-3 gap-2">
          {FILTER_TABS.map(t => (
            <motion.button key={t.id} whileTap={{ scale: 0.92 }}
              onClick={() => setFilter(t.id)}
              className="flex-shrink-0 px-3.5 py-1.5 rounded-full outline-none transition-all"
              style={filter === t.id
                ? { background: "var(--neu-accent-primary)", color: "white", fontSize: 12, boxShadow: "0 3px 10px var(--neu-glow)", border: "none" }
                : { background: "var(--neu-bg)", color: "var(--neu-text-muted)", fontSize: 12, border: "1px solid var(--neu-border)", boxShadow: "3px 3px 6px var(--neu-shadow-dark), -3px -3px 6px var(--neu-shadow-light)" }}>
              {t.label}
            </motion.button>
          ))}
        </div>
      </div>

      {/* List */}
      <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="flex flex-col px-5 pt-3 pb-6 gap-0"
        style={{ maxWidth: 640, marginLeft: "auto", marginRight: "auto", width: "100%" }}>
        {/* Unread group */}
        {unreadGroup.length > 0 && (
          <>
            <p className="uppercase mb-2" style={{ fontSize: 10, letterSpacing: "0.14em", color: "var(--neu-text-muted)" }}>新消息</p>
            {unreadGroup.map(n => <NotifRow key={n.id} notif={n} onRead={markRead} onRemove={remove} navigate={navigate} />)}
            {readGroup.length > 0 && <p className="uppercase mt-4 mb-2" style={{ fontSize: 10, letterSpacing: "0.14em", color: "var(--neu-text-muted)" }}>较早</p>}
          </>
        )}

        {/* Read group */}
        <AnimatePresence>
          {readGroup.map(n => <NotifRow key={n.id} notif={n} onRead={markRead} onRemove={remove} navigate={navigate} />)}
        </AnimatePresence>

        {filtered.length === 0 && (
          <div className="flex flex-col items-center justify-center py-20 gap-3">
            <div className="flex items-center justify-center rounded-full"
              style={{ width: 64, height: 64, boxShadow: "6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light)", border: "1px solid var(--neu-border)" }}>
              <Bell style={{ width: 24, height: 24, color: "var(--neu-text-muted)", opacity: 0.5 }} strokeWidth={1.25} />
            </div>
            <p style={{ fontSize: 15, color: "var(--neu-text)" }}>暂无通知</p>
          </div>
        )}
      </motion.div>
    </>
  );
}

function NotifRow({ notif, onRead, onRemove, navigate }: {
  notif: Notif;
  onRead: (id: number) => void;
  onRemove: (id: number) => void;
  navigate: ReturnType<typeof useNavigate>;
}) {
  const cfg = TYPE_CONFIG[notif.type];
  const Icon = cfg.icon;
  const hover = useHoverConfig();
  return (
    <motion.div
      layout
      initial={{ opacity: 0, x: -12 }}
      animate={{ opacity: 1, x: 0 }}
      exit={{ opacity: 0, x: 12, height: 0, marginBottom: 0 }}
      transition={{ duration: 0.22 }}
      onClick={() => onRead(notif.id)}
      whileTap={hover.tap as any}
      whileHover={hover.row as any}
      className="flex items-center gap-3.5 py-3.5 cursor-pointer rounded-[18px] mb-1.5 px-3 neu-row-hover"
      style={{
        background: notif.read ? "transparent" : `${cfg.color}07`,
        border: notif.read ? "1px solid transparent" : `1px solid ${cfg.color}18`,
      }}
    >
      {/* Avatar + icon badge */}
      <div className="relative flex-shrink-0">
        {notif.user.avatar ? (
          <div className="rounded-full overflow-hidden" style={{ width: 44, height: 44 }}>
            <img src={notif.user.avatar} alt={notif.user.name} className="w-full h-full object-cover" />
          </div>
        ) : (
          <div className="rounded-full flex items-center justify-center" style={{ width: 44, height: 44, background: `${cfg.color}18` }}>
            <Icon style={{ width: 18, height: 18, color: cfg.color }} strokeWidth={1.75} />
          </div>
        )}
        {notif.user.avatar && (
          <div className="absolute -bottom-0.5 -right-0.5 flex items-center justify-center rounded-full"
            style={{ width: 18, height: 18, background: cfg.color, border: "2px solid var(--neu-bg)" }}>
            <Icon style={{ width: 9, height: 9, color: "white" }} strokeWidth={2.5} />
          </div>
        )}
      </div>

      {/* Text */}
      <div className="flex-1 min-w-0">
        <p className="truncate" style={{ fontSize: 13, color: "var(--neu-text)" }}>
          <span style={{ fontWeight: notif.read ? 400 : 500 }}>{notif.user.name}</span>
          {" "}{notif.text}
        </p>
        <p style={{ fontSize: 11, color: "var(--neu-text-muted)", marginTop: 2 }}>{notif.time}</p>
      </div>

      {/* Post thumb or unread dot */}
      <div className="flex items-center gap-2 flex-shrink-0">
        {notif.postImage && (
          <div className="rounded-[10px] overflow-hidden" style={{ width: 38, height: 38 }}>
            <img src={notif.postImage} alt="" className="w-full h-full object-cover" />
          </div>
        )}
        {!notif.read && <div className="rounded-full" style={{ width: 8, height: 8, background: cfg.color }} />}
      </div>
    </motion.div>
  );
}