import React, { useState } from "react";
import { ArrowLeft, Users, MapPin, Search, X, Lock, Globe, Plus, Sparkles } from "lucide-react";
import { motion, AnimatePresence } from "motion/react";
import { useNavigate } from "react-router";
import { CreateGroupSheet } from "../components/ui/CreateGroupSheet";

const CATEGORIES = ["全部", "摄影", "音乐", "户外", "阅读", "美食", "健身", "游戏"];

const GROUPS = [
  {
    id: 1, name: "徐州街头摄影师联盟",
    banner: "https://images.unsplash.com/photo-1637354098670-9fb8b3d765fc?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=800",
    category: "摄影", members: 143, online: 12, distance: "0.4km", privacy: "public", joined: false,
    desc: "每周三次组织街头纪实拍摄，欢迎所有摄影爱好者",
    avatars: [
      "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80",
      "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80",
      "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=80",
    ],
  },
  {
    id: 2, name: "云龙湖晨跑团",
    banner: "https://images.unsplash.com/photo-1758599669009-5a9002c09487?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=800",
    category: "户外", members: 89, online: 6, distance: "0.8km", privacy: "public", joined: true,
    desc: "每天早上 6:30 云龙湖西门集合，共同开启活力一天",
    avatars: [
      "https://images.unsplash.com/photo-1758613172074-7275c3455034?w=80",
      "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80",
      "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=80",
    ],
  },
  {
    id: 3, name: "流浪乐手 · 徐州场",
    banner: "https://images.unsplash.com/photo-1776717494824-2036ea6cab02?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=800",
    category: "音乐", members: 62, online: 8, distance: "1.1km", privacy: "public", joined: false,
    desc: "每周末广场街头演出，各类乐器均欢迎，也可来做观众",
    avatars: [
      "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80",
      "https://images.unsplash.com/photo-1758613172074-7275c3455034?w=80",
    ],
  },
  {
    id: 4, name: "周末阅读角",
    banner: "https://images.unsplash.com/photo-1774125384717-5c5a05ce8e9c?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=800",
    category: "阅读", members: 34, online: 3, distance: "1.4km", privacy: "private", joined: false,
    desc: "每周日下午 2:00 彭城广场附近咖啡馆，分享好书，沉浸阅读",
    avatars: [
      "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=80",
      "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80",
    ],
  },
  {
    id: 5, name: "徐州家常菜研究所",
    banner: "https://images.unsplash.com/photo-1758279745466-f5f4087a87d6?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=800",
    category: "美食", members: 211, online: 19, distance: "1.9km", privacy: "public", joined: false,
    desc: "分享徐州特色美食，不定期组织聚餐试吃，菜谱交流",
    avatars: [
      "https://images.unsplash.com/photo-1758613172074-7275c3455034?w=80",
      "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80",
      "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=80",
    ],
  },
];

/* ── Hero featured group card ───────────────────────────────────── */
function FeaturedGroupCard({ group, onJoinToggle }: { group: typeof GROUPS[0]; onJoinToggle: (id: number) => void }) {
  return (
    <div className="neu-ambient-glow">
      <motion.div
        whileTap={{ scale: 0.995 }}
        className="neu-surface-hero rounded-[24px] overflow-hidden cursor-pointer"
      >
        <div className="relative" style={{ height: 140 }}>
          <img src={group.banner} alt={group.name} className="w-full h-full object-cover" />
          <div className="absolute inset-0" style={{ background: "linear-gradient(to top, rgba(0,0,0,0.65) 0%, rgba(0,0,0,0.05) 60%)" }} />
          <div className="absolute top-3 left-3 flex items-center gap-1.5 px-2 py-0.5 rounded-full"
            style={{ background: "rgba(255,255,255,0.18)", backdropFilter: "blur(8px)" }}>
            <Sparkles style={{ width: 10, height: 10, color: "white" }} strokeWidth={2} />
            <span style={{ fontSize: 10, color: "white", fontWeight: 500 }}>精选</span>
          </div>
          <div className="absolute top-3 right-3 px-2 py-0.5 rounded-full"
            style={{ background: "var(--neu-accent-primary)" }}>
            <span style={{ fontSize: 10, color: "white" }}>{group.category}</span>
          </div>
          <div className="absolute bottom-3 left-4 right-4">
            <p style={{ fontSize: 18, color: "white", fontWeight: 600 }} className="truncate">{group.name}</p>
            <p style={{ fontSize: 12, color: "rgba(255,255,255,0.85)", marginTop: 2 }} className="line-clamp-1">
              {group.desc}
            </p>
          </div>
        </div>
        <div className="flex items-center justify-between px-4 py-3">
          <div className="flex items-center gap-3.5">
            <div className="flex -space-x-1.5">
              {group.avatars.slice(0, 3).map((av, i) => (
                <img key={i} src={av} alt=""
                  className="rounded-full object-cover"
                  style={{ width: 22, height: 22, border: "2px solid var(--neu-bg)" }} />
              ))}
            </div>
            <div className="flex items-center gap-1">
              <Users style={{ width: 11, height: 11, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
              <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>{group.members}</span>
            </div>
            <div className="flex items-center gap-1">
              <span className="rounded-full" style={{ width: 6, height: 6, background: "#22c55e" }} />
              <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>{group.online}在线</span>
            </div>
            <div className="flex items-center gap-0.5">
              <MapPin style={{ width: 10, height: 10, color: "var(--neu-accent-primary)" }} strokeWidth={1.5} />
              <span style={{ fontSize: 11, color: "var(--neu-accent-primary)" }}>{group.distance}</span>
            </div>
          </div>
          <motion.button whileTap={{ scale: 0.93 }}
            onClick={e => { e.stopPropagation(); onJoinToggle(group.id); }}
            className="px-3.5 py-1.5 rounded-full outline-none"
            style={group.joined
              ? { background: "transparent", color: "var(--neu-text-muted)", border: "1px solid var(--neu-border-pressed)", fontSize: 12 }
              : { background: "var(--neu-accent-primary)", color: "white", fontSize: 12, fontWeight: 500, boxShadow: "0 3px 10px var(--neu-glow)" }}>
            {group.joined ? "已加入" : "加入"}
          </motion.button>
        </div>
      </motion.div>
    </div>
  );
}

/* ── Compact flat group row ─────────────────────────────────────── */
function GroupRow({ group, onJoinToggle }: { group: typeof GROUPS[0]; onJoinToggle: (id: number) => void }) {
  return (
    <div className="flex items-center gap-3 px-3 py-3 cursor-pointer transition-colors"
      style={{ background: "var(--neu-bg)" }}
      onMouseEnter={e => (e.currentTarget.style.background = "var(--neu-border)")}
      onMouseLeave={e => (e.currentTarget.style.background = "var(--neu-bg)")}>
      <div className="relative flex-shrink-0 rounded-[12px] overflow-hidden" style={{ width: 56, height: 56 }}>
        <img src={group.banner} alt={group.name} className="w-full h-full object-cover" />
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-1.5">
          <p style={{ fontSize: 13, color: "var(--neu-text)", fontWeight: 500 }} className="truncate">{group.name}</p>
          {group.privacy === "private"
            ? <Lock style={{ width: 10, height: 10, color: "var(--neu-text-muted)", flexShrink: 0 }} strokeWidth={2} />
            : <Globe style={{ width: 10, height: 10, color: "var(--neu-text-muted)", flexShrink: 0 }} strokeWidth={2} />
          }
        </div>
        <p style={{ fontSize: 11, color: "var(--neu-text-muted)", marginTop: 1 }} className="line-clamp-1">{group.desc}</p>
        <div className="flex items-center gap-2.5 mt-1">
          <div className="flex items-center gap-0.5">
            <Users style={{ width: 9, height: 9, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
            <span style={{ fontSize: 10, color: "var(--neu-text-muted)" }}>{group.members}</span>
          </div>
          <div className="flex items-center gap-0.5">
            <span className="rounded-full" style={{ width: 5, height: 5, background: "#22c55e" }} />
            <span style={{ fontSize: 10, color: "var(--neu-text-muted)" }}>{group.online}</span>
          </div>
          <div className="flex items-center gap-0.5">
            <MapPin style={{ width: 9, height: 9, color: "var(--neu-accent-primary)" }} strokeWidth={1.5} />
            <span style={{ fontSize: 10, color: "var(--neu-accent-primary)" }}>{group.distance}</span>
          </div>
        </div>
      </div>
      <motion.button whileTap={{ scale: 0.92 }}
        onClick={e => { e.stopPropagation(); onJoinToggle(group.id); }}
        className="px-3 py-1 rounded-full outline-none flex-shrink-0"
        style={group.joined
          ? { background: "transparent", color: "var(--neu-text-muted)", border: "1px solid var(--neu-border-pressed)", fontSize: 11 }
          : { background: "var(--neu-accent-primary)", color: "white", border: "1px solid transparent", fontSize: 11 }}>
        {group.joined ? "已加入" : "加入"}
      </motion.button>
    </div>
  );
}

export function NearbyGroups() {
  const navigate = useNavigate();
  const [activeCategory, setActiveCategory] = useState("全部");
  const [searchQ, setSearchQ] = useState("");
  const [showCreate, setShowCreate] = useState(false);
  const [groups, setGroups] = useState(GROUPS);

  const toggleJoin = (id: number) =>
    setGroups(prev => prev.map(g => g.id === id ? { ...g, joined: !g.joined } : g));

  const filtered = groups.filter(g => {
    const matchCat = activeCategory === "全部" || g.category === activeCategory;
    const matchSearch = !searchQ || g.name.includes(searchQ) || g.desc.includes(searchQ);
    return matchCat && matchSearch;
  });

  const [featured, ...rest] = filtered;

  return (
    <>
      <div className="sticky top-0 z-10 flex flex-col"
        style={{ backgroundColor: "var(--neu-bg)", borderBottom: "1px solid var(--neu-border)" }}>
        <div className="flex items-center gap-3 px-5 py-3">
          <motion.button whileTap={{ scale: 0.92 }}
            onClick={() => navigate(-1)}
            className="flex items-center justify-center rounded-full outline-none flex-shrink-0 neu-surface-flat"
            style={{ width: 34, height: 34 }}>
            <ArrowLeft style={{ width: 15, height: 15, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
          </motion.button>
          <div className="flex-1">
            <h1 style={{ fontSize: 18, color: "var(--neu-text)" }}>附近群组</h1>
            <div className="flex items-center gap-1">
              <MapPin style={{ width: 10, height: 10, color: "var(--neu-accent-primary)" }} strokeWidth={1.5} />
              <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>徐州 · 5km内 · {filtered.length} 个</span>
            </div>
          </div>
          <motion.button whileTap={{ scale: 0.93 }}
            onClick={() => setShowCreate(true)}
            className="flex items-center gap-1 px-3 py-1.5 rounded-full outline-none"
            style={{ background: "var(--neu-accent-primary)", color: "white", fontSize: 12, fontWeight: 500, boxShadow: "0 3px 10px var(--neu-glow)" }}>
            <Plus style={{ width: 13, height: 13 }} strokeWidth={2.5} />
            创建
          </motion.button>
        </div>

        <div className="px-5 pb-3">
          <div className="flex items-center gap-2.5 px-4 rounded-[14px] neu-input-surface" style={{ height: 38 }}>
            <Search style={{ width: 14, height: 14, color: "var(--neu-text-muted)", flexShrink: 0 }} strokeWidth={1.5} />
            <input
              className="flex-1 bg-transparent outline-none"
              style={{ fontSize: 13, color: "var(--neu-text)", minWidth: 0 }}
              placeholder="搜索附近群组…"
              value={searchQ}
              onChange={e => setSearchQ(e.target.value)}
            />
            <AnimatePresence>
              {searchQ && (
                <motion.button initial={{ opacity: 0, scale: 0.7 }} animate={{ opacity: 1, scale: 1 }} exit={{ opacity: 0, scale: 0.7 }}
                  whileTap={{ scale: 0.9 }}
                  onClick={() => setSearchQ("")}
                  className="outline-none">
                  <X style={{ width: 13, height: 13, color: "var(--neu-text-muted)" }} strokeWidth={2} />
                </motion.button>
              )}
            </AnimatePresence>
          </div>
        </div>

        {/* Category — ghost chips, active uses accent text */}
        <div className="overflow-x-auto no-scrollbar px-5 pb-3">
          <div className="flex gap-1.5" style={{ width: "max-content" }}>
            {CATEGORIES.map(cat => {
              const active = activeCategory === cat;
              return (
                <motion.button key={cat} whileTap={{ scale: 0.94 }}
                  onClick={() => setActiveCategory(cat)}
                  className="flex-shrink-0 px-3.5 py-1 rounded-full outline-none transition-all neu-surface-ghost"
                  style={active
                    ? {
                        background: "var(--neu-accent-soft)",
                        color: "var(--neu-accent-primary)",
                        fontSize: 12,
                        fontWeight: 500,
                        borderColor: "var(--neu-accent-soft-border)",
                      }
                    : { fontSize: 12, color: "var(--neu-text-muted)" }}>
                  {cat}
                </motion.button>
              );
            })}
          </div>
        </div>
      </div>

      <motion.div
        initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.28, ease: [0.22, 1, 0.36, 1] }}
        className="px-5 pt-5 pb-8"
        style={{ maxWidth: 680, marginLeft: "auto", marginRight: "auto", width: "100%" }}>

        {filtered.length === 0 && (
          <div className="flex flex-col items-center gap-3 py-20">
            <Users style={{ width: 40, height: 40, color: "var(--neu-text-muted)", opacity: 0.4 }} strokeWidth={1} />
            <p style={{ fontSize: 14, color: "var(--neu-text-muted)" }}>没有找到相关群组</p>
          </div>
        )}

        {featured && (
          <>
            <FeaturedGroupCard group={featured} onJoinToggle={toggleJoin} />

            {rest.length > 0 && (
              <>
                <div className="flex items-center justify-between mt-7 mb-3">
                  <span className="neu-eyebrow">更多群组</span>
                  <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>{rest.length} 个</span>
                </div>
                <div className="rounded-[16px] overflow-hidden flex flex-col"
                  style={{ gap: 1, background: "var(--neu-border)", border: "1px solid var(--neu-border)" }}>
                  {rest.map(g => (
                    <GroupRow key={g.id} group={g} onJoinToggle={toggleJoin} />
                  ))}
                </div>
              </>
            )}
          </>
        )}
      </motion.div>

      {/* Create group sheet */}
      <AnimatePresence>
        {showCreate && (
          <>
            <motion.div key="backdrop" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
              transition={{ duration: 0.2 }}
              className="fixed inset-0 z-40"
              style={{ background: "rgba(0,0,0,0.35)", backdropFilter: "blur(3px)" }}
              onClick={() => setShowCreate(false)} />
            <motion.div key="sheet"
              initial={{ y: "100%" }} animate={{ y: 0 }} exit={{ y: "100%" }}
              transition={{ type: "spring", damping: 30, stiffness: 260, mass: 0.9 }}
              className="fixed bottom-0 left-0 right-0 z-50 overflow-hidden"
              style={{ height: "92%", borderRadius: "28px 28px 0 0", backgroundColor: "var(--neu-bg)", boxShadow: "0 -8px 40px rgba(0,0,0,0.18)" }}>
              <CreateGroupSheet onClose={() => setShowCreate(false)} />
            </motion.div>
          </>
        )}
      </AnimatePresence>
    </>
  );
}
