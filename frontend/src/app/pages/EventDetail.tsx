import React, { useState } from "react";
import { useParams, useNavigate } from "react-router";
import { ArrowLeft, MapPin, Clock, Users, Share2, Calendar, ChevronRight, Check, Star } from "lucide-react";
import { motion } from "motion/react";

/* ── Mock data ─────────────────────────────────────────────────── */
const EVENTS: Record<number, {
  id: number; title: string; tag: string; color: string; bg: string;
  cover: string; description: string; location: string; date: string;
  time: string; host: { name: string; avatar: string };
  participants: Array<{ id: number; name: string; avatar: string }>;
  maxParticipants: number; distance: string;
  highlights: string[];
}> = {
  1: {
    id: 1, title: "晨间咖啡会", tag: "社交", color: "#F59E0B", bg: "rgba(245,158,11,0.12)",
    cover: "https://images.unsplash.com/photo-1593536488177-1eb3c2d4e3d2?w=1080",
    description: "每周六早上九点，我们聚集在老街咖啡馆，分享一周的所见所闻。无论是设计师、程序员还是艺术家，都欢迎加入！带上你的好奇心就够了。",
    location: "老街咖啡馆·徐州市鼓楼区", date: "2026年4月18日 周六", time: "09:00 - 11:00",
    host: { name: "Aria Sol", avatar: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80" },
    participants: [
      { id: 1, name: "Liam",    avatar: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80" },
      { id: 2, name: "Sophia",  avatar: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80" },
      { id: 3, name: "Marco",   avatar: "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=80" },
      { id: 4, name: "Aiko",    avatar: "https://images.unsplash.com/photo-1758613172074-7275c3455034?w=80" },
    ],
    maxParticipants: 20, distance: "0.3km",
    highlights: ["免费咖啡 ☕", "结识新朋友", "分享创意", "每周固定"],
  },
  2: {
    id: 2, title: "街头音乐节", tag: "音乐", color: "#8B5CF6", bg: "rgba(139,92,246,0.12)",
    cover: "https://images.unsplash.com/photo-1768776179834-93e6cafc6d97?w=1080",
    description: "在城市的中心，用音乐连接每一颗心。本次街头音乐节将邀请本地独立乐队和歌手进行现场表演，免费入场，欢迎所有人来感受音乐的力量。",
    location: "彭城广场·徐州市云龙区", date: "2026年4月19日 周日", time: "14:00 - 20:00",
    host: { name: "本地向导", avatar: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80" },
    participants: [
      { id: 1, name: "Noah",     avatar: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80" },
      { id: 2, name: "Isabella", avatar: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80" },
      { id: 3, name: "Kai",      avatar: "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=80" },
    ],
    maxParticipants: 200, distance: "0.7km",
    highlights: ["免费入场 🎸", "现场表演", "多个乐队", "本地艺术家"],
  },
};

const DEFAULT_EVENT = EVENTS[1];

export function EventDetail() {
  const { eventId } = useParams<{ eventId: string }>();
  const navigate = useNavigate();
  const ev = EVENTS[Number(eventId)] ?? DEFAULT_EVENT;
  const [joined, setJoined] = useState(false);

  const fillRate = Math.round((ev.participants.length / ev.maxParticipants) * 100);

  return (
    <div className="flex flex-col" style={{ height: "100%", backgroundColor: "var(--neu-bg)" }}>

      {/* ── Scrollable area ──────────────────────────────── */}
      <div className="flex-1 overflow-y-auto no-scrollbar">

        {/* Cover image — full bleed */}
        <div className="relative flex-shrink-0 overflow-hidden" style={{ height: "clamp(200px, 32vw, 280px)" }}>
          <img src={ev.cover} alt={ev.title} className="absolute inset-0 w-full h-full object-cover" />
          <div className="absolute inset-0" style={{ background: `linear-gradient(to bottom, transparent 30%, ${ev.bg} 100%)` }} />

          {/* Back button */}
          <div className="absolute top-4 left-4 flex gap-2">
            <motion.button whileTap={{ scale: 0.88 }} onClick={() => navigate(-1)}
              className="flex items-center justify-center rounded-full outline-none"
              style={{ width: 36, height: 36, background: "rgba(0,0,0,0.35)", backdropFilter: "blur(10px)", border: "1px solid rgba(255,255,255,0.25)" }}>
              <ArrowLeft style={{ width: 15, height: 15, color: "white" }} strokeWidth={1.75} />
            </motion.button>
          </div>
          <div className="absolute top-4 right-4">
            <motion.button whileTap={{ scale: 0.88 }}
              className="flex items-center justify-center rounded-full outline-none"
              style={{ width: 36, height: 36, background: "rgba(0,0,0,0.35)", backdropFilter: "blur(10px)", border: "1px solid rgba(255,255,255,0.25)" }}>
              <Share2 style={{ width: 14, height: 14, color: "white" }} strokeWidth={1.75} />
            </motion.button>
          </div>

          {/* Tag */}
          <div className="absolute bottom-4 left-4">
            <span className="px-3 py-1 rounded-full" style={{ fontSize: 12, background: ev.color, color: "white", boxShadow: `0 3px 10px ${ev.color}50` }}>
              {ev.tag}
            </span>
          </div>
        </div>

        {/* Content */}
        <div className="px-5 pt-4 pb-6"
          style={{ maxWidth: 640, marginLeft: "auto", marginRight: "auto", width: "100%" }}>
          <h1 style={{ fontSize: 24, color: "var(--neu-text)", marginBottom: 8 }}>{ev.title}</h1>

          {/* Meta info */}
          <div className="flex flex-col gap-2 mb-5">
            {[
              { icon: Calendar, text: ev.date },
              { icon: Clock,    text: ev.time },
              { icon: MapPin,   text: ev.location },
            ].map(({ icon: Icon, text }) => (
              <div key={text} className="flex items-center gap-2.5">
                <div className="flex items-center justify-center rounded-full flex-shrink-0"
                  style={{ width: 30, height: 30, background: `${ev.color}15` }}>
                  <Icon style={{ width: 13, height: 13, color: ev.color }} strokeWidth={1.75} />
                </div>
                <span style={{ fontSize: 13, color: "var(--neu-text-muted)" }}>{text}</span>
              </div>
            ))}
          </div>

          {/* Host */}
          <div className="flex items-center gap-3 py-3.5 px-4 rounded-[20px] mb-4"
            style={{ background: "var(--neu-bg)", boxShadow: "5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light)", border: "1px solid var(--neu-border)" }}>
            <div className="rounded-full overflow-hidden flex-shrink-0" style={{ width: 40, height: 40, border: `2px solid ${ev.color}` }}>
              <img src={ev.host.avatar} alt={ev.host.name} className="w-full h-full object-cover" />
            </div>
            <div className="flex-1">
              <p style={{ fontSize: 13, color: "var(--neu-text)" }}>{ev.host.name}</p>
              <div className="flex items-center gap-1">
                <Star style={{ width: 10, height: 10, color: "#F59E0B" }} fill="#F59E0B" strokeWidth={1.5} />
                <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>活动发起人</span>
              </div>
            </div>
            <motion.button whileTap={{ scale: 0.93 }} className="px-3 py-1.5 rounded-full outline-none"
              style={{ background: ev.color, color: "white", fontSize: 12, boxShadow: `0 3px 10px ${ev.color}40` }}>
              关注
            </motion.button>
          </div>

          {/* Description */}
          <p style={{ fontSize: 14, color: "var(--neu-text-muted)", lineHeight: 1.75, marginBottom: 20 }}>{ev.description}</p>

          {/* Highlights */}
          <div className="grid gap-2.5 mb-5"
            style={{ gridTemplateColumns: "repeat(auto-fill, minmax(180px, 1fr))" }}>
            {ev.highlights.map((h, i) => (
              <div key={i} className="flex items-center gap-2.5 px-3.5 py-3 rounded-[16px]"
                style={{ background: "var(--neu-bg)", boxShadow: "4px 4px 10px var(--neu-shadow-dark), -4px -4px 10px var(--neu-shadow-light)", border: "1px solid var(--neu-border)" }}>
                <div className="flex items-center justify-center rounded-full flex-shrink-0"
                  style={{ width: 24, height: 24, background: `${ev.color}15` }}>
                  <Check style={{ width: 11, height: 11, color: ev.color }} strokeWidth={2.5} />
                </div>
                <span style={{ fontSize: 12, color: "var(--neu-text)" }}>{h}</span>
              </div>
            ))}
          </div>

          {/* Participants */}
          <div className="mb-5">
            <div className="flex items-center justify-between mb-3">
              <p style={{ fontSize: 14, color: "var(--neu-text)" }}>参与者 · {ev.participants.length}/{ev.maxParticipants}</p>
              <motion.button whileTap={{ scale: 0.93 }} className="flex items-center gap-0.5 outline-none" style={{ color: "var(--neu-accent-primary)" }}>
                <span style={{ fontSize: 12 }}>查看全部</span>
                <ChevronRight style={{ width: 13, height: 13 }} strokeWidth={1.5} />
              </motion.button>
            </div>

            {/* Progress bar */}
            <div className="rounded-full overflow-hidden mb-3" style={{ height: 6, background: "var(--neu-border)" }}>
              <motion.div
                initial={{ width: 0 }}
                animate={{ width: `${fillRate}%` }}
                transition={{ duration: 0.8, ease: "easeOut" }}
                style={{ height: "100%", background: ev.color, borderRadius: 3 }}
              />
            </div>

            {/* Avatar stack */}
            <div className="flex items-center gap-1">
              {ev.participants.map((p, i) => (
                <div key={p.id} className="rounded-full overflow-hidden"
                  style={{ width: 36, height: 36, border: "2px solid var(--neu-bg)", marginLeft: i > 0 ? -10 : 0, zIndex: ev.participants.length - i }}>
                  <img src={p.avatar} alt={p.name} className="w-full h-full object-cover" />
                </div>
              ))}
              {ev.maxParticipants - ev.participants.length > 0 && (
                <div className="rounded-full flex items-center justify-center"
                  style={{ width: 36, height: 36, border: "2px solid var(--neu-bg)", marginLeft: -10, background: `${ev.color}20` }}>
                  <span style={{ fontSize: 11, color: ev.color, fontWeight: 600 }}>+{ev.maxParticipants - ev.participants.length}</span>
                </div>
              )}
              <span style={{ fontSize: 12, color: "var(--neu-text-muted)", marginLeft: 8 }}>已报名</span>
            </div>
          </div>

          {/* Distance info */}
          <div className="flex items-center gap-2 px-4 py-3 rounded-[16px]"
            style={{ background: `${ev.color}10`, border: `1px solid ${ev.color}25` }}>
            <MapPin style={{ width: 14, height: 14, color: ev.color }} strokeWidth={1.5} />
            <span style={{ fontSize: 13, color: "var(--neu-text-muted)" }}>距你 {ev.distance} · 步行约 {Math.ceil(parseInt(ev.distance) * 12)} 分钟</span>
          </div>
        </div>

      </div>{/* end scrollable */}

      {/* ── Sticky CTA — always visible at bottom of card ── */}
      <div
        className="flex-shrink-0 px-5 pb-6 pt-3"
        style={{
          backgroundColor: "var(--neu-bg)",
          borderTop: "1px solid var(--neu-border)",
          maxWidth: 640,
          marginLeft: "auto",
          marginRight: "auto",
          width: "100%",
        }}
      >
        <motion.button
          whileTap={{ scale: 0.97 }}
          onClick={() => setJoined(v => !v)}
          className="w-full py-4 rounded-[22px] outline-none flex items-center justify-center gap-2 transition-all duration-300"
          style={joined
            ? { background: "var(--neu-bg)", border: `2px solid ${ev.color}`, color: ev.color, boxShadow: `0 0 0 4px ${ev.color}18`, fontSize: 15 }
            : { background: ev.color, color: "white", boxShadow: `0 8px 24px ${ev.color}45`, border: "none", fontSize: 15 }}>
          {joined
            ? <><Check style={{ width: 18, height: 18 }} strokeWidth={2.5} /> 已报名 · 点击取消</>
            : <><Users style={{ width: 18, height: 18 }} strokeWidth={2} /> 立即报名参加</>}
        </motion.button>
      </div>

    </div>
  );
}