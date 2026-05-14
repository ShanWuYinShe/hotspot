import React, { useState } from "react";
import { useNavigate } from "react-router";
import { ArrowLeft, MapPin, Search, Map, Radar } from "lucide-react";
import { motion } from "motion/react";
import { usePlatform } from "../contexts/PlatformContext";
import {
  TabletRail, NearbyPulseCard, MoodCheckinCard, SuggestedFollowsCard,
  StreakCard, VibeCard,
} from "../components/TabletRail";

const ALL_USERS = [
  { id: 1, name: "Liam",     distance: "246m",  avatar:"https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80",  status:"Online",  tag:"摄影", mutual:3 },
  { id: 2, name: "Sophia",   distance: "252m",  avatar:"https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80",  status:"Online",  tag:"旅行", mutual:7 },
  { id: 3, name: "Marco",    distance: "388m",  avatar:"https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=80",  status:"Offline", tag:"音乐", mutual:1 },
  { id: 4, name: "Aiko",     distance: "520m",  avatar:"https://images.unsplash.com/photo-1758613172074-7275c3455034?w=80",  status:"Online",  tag:"美食", mutual:5 },
  { id: 5, name: "Elena",    distance: "820m",  avatar:"https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80",  status:"Offline", tag:"艺术", mutual:2 },
  { id: 6, name: "Kai",      distance: "910m",  avatar:"https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=80",  status:"Online",  tag:"运动", mutual:4 },
  { id: 7, name: "Mia",      distance: "1.1km", avatar:"https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80",  status:"Offline", tag:"阅读", mutual:0 },
  { id: 8, name: "Noah",     distance: "1.3km", avatar:"https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80",  status:"Online",  tag:"游戏", mutual:6 },
  { id: 9, name: "Isabella", distance: "1.6km", avatar:"https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80",  status:"Online",  tag:"电影", mutual:8 },
  { id: 10,name: "Carlos",   distance: "2.1km", avatar:"https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=80",  status:"Offline", tag:"科技", mutual:0 },
  { id: 11,name: "Yuki",     distance: "2.4km", avatar:"https://images.unsplash.com/photo-1758613172074-7275c3455034?w=80",  status:"Online",  tag:"时尚", mutual:3 },
  { id: 12,name: "Oliver",   distance: "3.0km", avatar:"https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80",  status:"Offline", tag:"户外", mutual:1 },
];

const SORT_OPTIONS = ["距离", "在线", "共同好友"];
const FILTER_TAGS  = ["全部", "摄影", "旅行", "音乐", "美食", "运动", "阅读", "艺术"];

export function NearbyAll() {
  const navigate = useNavigate();
  const { platform } = usePlatform();
  const isTablet = platform === "tablet";
  const [search,  setSearch]  = useState("");
  const [sort,    setSort]    = useState("距离");
  const [tagFilt, setTagFilt] = useState("全部");
  const [followedIds, setFollowedIds] = useState<Set<number>>(new Set());

  const toggle = (id: number) => setFollowedIds(p => { const n = new Set(p); n.has(id) ? n.delete(id) : n.add(id); return n; });

  let list = ALL_USERS.filter(u =>
    (tagFilt === "全部" || u.tag === tagFilt) &&
    (search === "" || u.name.toLowerCase().includes(search.toLowerCase()))
  );
  if (sort === "在线") list = [...list].sort((a, b) => (b.status === "Online" ? 1 : 0) - (a.status === "Online" ? 1 : 0));
  if (sort === "共同好友") list = [...list].sort((a, b) => b.mutual - a.mutual);

  const onlineCount = list.filter(u => u.status === "Online").length;

  return (
    <>
      {/* Header — flat, no shadow */}
      <div className="sticky top-0 z-10 pt-3 pb-3"
        style={{ backgroundColor: "var(--neu-bg)", borderBottom: "1px solid var(--neu-border)" }}>
        <div className={isTablet ? "flex gap-6 mx-auto" : "px-5"}
          style={isTablet ? { maxWidth: 964, width: "100%" } : {}}>
          <div className={isTablet ? "mx-auto w-full lg:mx-0 px-5 min-w-0" : "w-full"}
            style={isTablet ? { maxWidth: 640 } : {}}>
        <div className="flex items-center gap-3 mb-3">
          <motion.button whileTap={{ scale: 0.92 }} onClick={() => navigate(-1)}
            className="flex items-center justify-center rounded-full outline-none neu-surface-flat"
            style={{ width: 34, height: 34 }}>
            <ArrowLeft style={{ width: 15, height: 15, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
          </motion.button>
          <h1 className="flex-1" style={{ fontSize: 18, color: "var(--neu-text)" }}>附近的人</h1>
          <motion.button whileTap={{ scale: 0.92 }}
            onClick={() => navigate("/location")}
            className="flex items-center justify-center rounded-full outline-none neu-surface-flat"
            style={{ width: 34, height: 34 }}>
            <Map style={{ width: 15, height: 15, color: "var(--neu-accent-primary)" }} strokeWidth={1.5} />
          </motion.button>
        </div>

        {/* Search — pressed input */}
        <div className="flex items-center gap-3 px-4 py-2.5 rounded-[14px] neu-input-surface">
          <Search style={{ width: 14, height: 14, color: "var(--neu-text-muted)", flexShrink: 0 }} strokeWidth={1.5} />
          <input value={search} onChange={e => setSearch(e.target.value)}
            placeholder="搜索用户…" className="flex-1 bg-transparent outline-none"
            style={{ fontSize: 13, color: "var(--neu-text)" }} />
        </div>
          </div>
          {isTablet && <div className="hidden lg:block flex-shrink-0" style={{ width: 300 + 20 }} />}
        </div>
      </div>

      <div className={isTablet ? "flex gap-6 mx-auto pt-5 pb-8" : "px-5 pt-5 pb-8"}
        style={isTablet ? { maxWidth: 964, width: "100%" } : { maxWidth: 640, marginLeft: "auto", marginRight: "auto", width: "100%" }}>
        <div className={isTablet ? "mx-auto w-full lg:mx-0 px-5 min-w-0" : "w-full"}
          style={isTablet ? { maxWidth: 640 } : {}}>

        {/* Hero summary with ambient glow */}
        <div className="neu-ambient-glow mb-6">
          <div className="neu-surface-hero rounded-[22px] flex items-center gap-4"
            style={{ padding: "16px 18px" }}>
            <div className="flex items-center justify-center rounded-full flex-shrink-0"
              style={{
                width: 44, height: 44,
                background: "var(--neu-accent-primary)",
                boxShadow: "0 4px 14px var(--neu-glow)",
              }}>
              <Radar style={{ width: 19, height: 19, color: "white" }} strokeWidth={2} />
            </div>
            <div className="flex-1">
              <p style={{ fontSize: 13, color: "var(--neu-text)", fontWeight: 500 }}>
                徐州 · 5km 范围
              </p>
              <p style={{ fontSize: 11, color: "var(--neu-text-muted)", marginTop: 2 }}>
                <span style={{ color: "#22c55e" }}>{onlineCount} 人在线</span>
                <span style={{ marginLeft: 8 }}>· 共 {list.length} 人</span>
              </p>
            </div>
          </div>
        </div>

        {/* Sort — segmented */}
        <div className="flex items-center gap-1 mb-3 p-1 rounded-full neu-input-surface w-fit">
          {SORT_OPTIONS.map(s => (
            <motion.button key={s} whileTap={{ scale: 0.95 }}
              onClick={() => setSort(s)}
              className="px-3.5 py-1 rounded-full outline-none transition-all"
              style={sort === s
                ? { background: "var(--neu-accent-primary)", color: "white", fontSize: 12, fontWeight: 500 }
                : { color: "var(--neu-text-muted)", fontSize: 12 }}>
              {s}
            </motion.button>
          ))}
        </div>

        {/* Tag filters — ghost chips */}
        <div className="flex gap-1.5 overflow-x-auto no-scrollbar mb-5">
          {FILTER_TAGS.map(t => {
            const active = tagFilt === t;
            return (
              <motion.button key={t} whileTap={{ scale: 0.94 }}
                onClick={() => setTagFilt(t)}
                className="flex-shrink-0 px-3 py-1 rounded-full outline-none transition-all neu-surface-ghost"
                style={active
                  ? {
                      background: "var(--neu-accent-soft)",
                      color: "var(--neu-accent-primary)",
                      fontSize: 12,
                      fontWeight: 500,
                      borderColor: "var(--neu-accent-soft-border)",
                    }
                  : { fontSize: 12, color: "var(--neu-text-muted)" }}>
                {t}
              </motion.button>
            );
          })}
        </div>

        {/* List — flat joined rows */}
        <div className="flex flex-col rounded-[16px] overflow-hidden"
          style={{ gap: 1, background: "var(--neu-border)", border: "1px solid var(--neu-border)" }}>
          {list.map(user => {
            const followed = followedIds.has(user.id);
            return (
              <div key={user.id}
                onClick={() => navigate(`/user/${user.id}`)}
                className="flex items-center gap-3 px-4 py-3 cursor-pointer transition-colors"
                style={{ background: "var(--neu-bg)" }}
                onMouseEnter={e => (e.currentTarget.style.background = "var(--neu-border)")}
                onMouseLeave={e => (e.currentTarget.style.background = "var(--neu-bg)")}>
                <div className="relative flex-shrink-0">
                  <div className="rounded-full overflow-hidden" style={{ width: 42, height: 42 }}>
                    <img src={user.avatar} alt={user.name} className="w-full h-full object-cover" />
                  </div>
                  {user.status === "Online" && (
                    <span className="absolute bottom-0 right-0 rounded-full"
                      style={{ width: 10, height: 10, background: "#22c55e", border: "2px solid var(--neu-bg)" }} />
                  )}
                </div>
                <div className="flex-1 min-w-0">
                  <p style={{ fontSize: 14, color: "var(--neu-text)" }}>{user.name}</p>
                  <div className="flex items-center gap-1.5 mt-0.5">
                    <span style={{ fontSize: 11, color: "var(--neu-accent-primary)" }}>#{user.tag}</span>
                    {user.mutual > 0 && (
                      <>
                        <span style={{ fontSize: 10, color: "var(--neu-text-muted)", opacity: 0.5 }}>·</span>
                        <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>{user.mutual}位共同</span>
                      </>
                    )}
                  </div>
                </div>
                <div className="flex flex-col items-end gap-1.5">
                  <div className="flex items-center gap-0.5">
                    <MapPin style={{ width: 10, height: 10, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
                    <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>{user.distance}</span>
                  </div>
                  <motion.button whileTap={{ scale: 0.93 }}
                    onClick={e => { e.stopPropagation(); toggle(user.id); }}
                    className="px-3 py-0.5 rounded-full outline-none transition-all"
                    style={followed
                      ? { background: "transparent", color: "var(--neu-text-muted)", border: "1px solid var(--neu-border-pressed)", fontSize: 11 }
                      : { background: "var(--neu-accent-primary)", color: "white", border: "1px solid transparent", fontSize: 11 }}>
                    {followed ? "已关注" : "关注"}
                  </motion.button>
                </div>
              </div>
            );
          })}
        </div>
        </div>

        {isTablet && (
          <aside className="hidden lg:block flex-shrink-0 pr-5" style={{ width: 300 + 20 }}>
            <div style={{ width: 300 }}>
              <TabletRail>
                <NearbyPulseCard />
                <MoodCheckinCard />
                <SuggestedFollowsCard />
                <StreakCard />
                <VibeCard />
              </TabletRail>
            </div>
          </aside>
        )}
      </div>
    </>
  );
}
