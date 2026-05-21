import { useState } from "react";
import { useParams, useNavigate } from "react-router";
import { ArrowLeft, MapPin, Clock, Users, Share2, Calendar, ChevronRight, Check, Star } from "lucide-react";
import { motion } from "motion/react";

import { MOCK_EVENTS, MOCK_DEFAULT_EVENT } from "./EventDetail.mock";
import { useDocumentTitle } from "@/lib/hooks/use-document-title";
import { goBack } from "@/lib/utils/navigation";
import { ImageWithFallback } from "../components/figma/ImageWithFallback";

export function EventDetail() {
  const { eventId } = useParams<{ eventId: string }>();
  const navigate = useNavigate();
  const ev = MOCK_EVENTS[Number(eventId)] ?? MOCK_DEFAULT_EVENT;
  const [joined, setJoined] = useState(false);
  useDocumentTitle("活动详情");

  const fillRate = Math.round((ev.participants.length / ev.maxParticipants) * 100);
  const walkDist = parseInt(ev.distance);
  const walkMin = isNaN(walkDist) ? "" : ` · 步行约 ${Math.ceil(walkDist * 12)} 分钟`;

  return (
    <div className="flex flex-col" style={{ height: "100%", backgroundColor: "var(--neu-bg)" }}>

      {/* ── Scrollable area ──────────────────────────────── */}
      <div className="flex-1 overflow-y-auto no-scrollbar">

        {/* Cover image — full bleed */}
        <div className="relative flex-shrink-0 overflow-hidden" style={{ height: "clamp(200px, 32vw, 280px)" }}>
          <ImageWithFallback src={ev.cover} alt="活动封面" className="absolute inset-0 w-full h-full object-cover" loading="lazy" />
          <div className="absolute inset-0" style={{ background: `linear-gradient(to bottom, transparent 30%, ${ev.bg} 100%)` }} />

          {/* Back button */}
          <div className="absolute top-4 left-4 flex gap-2">
            <motion.button whileTap={{ scale: 0.88 }} onClick={() => goBack(navigate)} aria-label="返回"
              className="flex items-center justify-center rounded-full outline-none"
              style={{ width: 36, height: 36, background: "rgba(0,0,0,0.35)", backdropFilter: "blur(10px)", border: "1px solid rgba(255,255,255,0.25)" }}>
              <ArrowLeft style={{ width: 15, height: 15, color: "white" }} strokeWidth={1.75} />
            </motion.button>
          </div>
          <div className="absolute top-4 right-4">
            <motion.button whileTap={{ scale: 0.88 }} aria-label="分享"
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
          <h1 className="line-clamp-2" style={{ fontSize: 24, color: "var(--neu-text)", marginBottom: 8 }}>{ev.title}</h1>

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
              <ImageWithFallback src={ev.host.avatar} alt={ev.host.name} className="w-full h-full object-cover" />
            </div>
            <div className="flex-1">
              <p className="truncate" style={{ fontSize: 13, color: "var(--neu-text)" }}>{ev.host.name}</p>
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
            <div className="flex items-center gap-1 overflow-hidden flex-shrink-0">
              {ev.participants.map((p, i) => (
                <div key={p.id} className="rounded-full overflow-hidden"
                  style={{ width: 36, height: 36, border: "2px solid var(--neu-bg)", marginLeft: i > 0 ? -10 : 0, zIndex: ev.participants.length - i }}>
                  <ImageWithFallback src={p.avatar} alt={p.name} className="w-full h-full object-cover" />
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
            <span style={{ fontSize: 13, color: "var(--neu-text-muted)" }}>距你 {ev.distance}{walkMin}</span>
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