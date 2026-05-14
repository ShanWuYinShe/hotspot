import React, { useState, useEffect, useRef } from "react";
import { useNavigate } from "react-router";
import { ArrowLeft, Search as SearchIcon, TrendingUp, Hash, Users, FileText, MapPin, Clock, X } from "lucide-react";
import { motion, AnimatePresence } from "motion/react";

/* ── All searchable items ─────────────────────────────────────────── */
const PEOPLE = [
  { id: 1, name: "Liam",     handle: "@liam.photos", avatar: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80", tag: "摄影" },
  { id: 2, name: "Sophia",   handle: "@sophia.wander", avatar: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80", tag: "旅行" },
  { id: 3, name: "Marco",    handle: "@marco.music",  avatar: "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=80", tag: "音乐" },
];
const POSTS = [
  { id: 40, title: "植物园晨曦",   user: "Aria Sol", image: "https://images.unsplash.com/photo-1768776179834-93e6cafc6d97?w=120", tag: "自然",  likes: 124 },
  { id: 41, title: "城市夜光",     user: "本地向导", image: "https://images.unsplash.com/photo-1748737349771-e2b6605e5961?w=120", tag: "城市",  likes: 89 },
  { id: 42, title: "老街咖啡馆",   user: "Marco",    image: "https://images.unsplash.com/photo-1593536488177-1eb3c2d4e3d2?w=120", tag: "生活",  likes: 56 },
  { id: 43, title: "云龙湖日落",   user: "Isabella", image: "https://images.unsplash.com/photo-1759734282634-b4958ea9a6e0?w=120", tag: "风景",  likes: 201 },
];
const TAGS = ["摄影", "城市探索", "周末去哪", "隐藏咖啡馆", "徐州美食", "本地生活", "户外运动", "街头摄影"];
const HOT_SEARCHES = ["植物园", "云龙湖", "周末市集", "咖啡馆", "摄影"];
const RECENT_SEARCHES = ["Aria Sol", "街头音乐节", "徐州骑行"];

type TabType = "all" | "people" | "posts" | "tags";

export function Search() {
  const navigate = useNavigate();
  const inputRef = useRef<HTMLInputElement>(null);
  const [query,   setQuery]   = useState("");
  const [tab,     setTab]     = useState<TabType>("all");
  const [recent,  setRecent]  = useState(RECENT_SEARCHES);
  const [focused, setFocused] = useState(true);

  useEffect(() => { inputRef.current?.focus(); }, []);

  const q = query.toLowerCase().trim();

  const filteredPeople = q
    ? PEOPLE.filter(p => p.name.toLowerCase().includes(q) || p.handle.includes(q) || p.tag.includes(q))
    : [];
  const filteredPosts = q
    ? POSTS.filter(p => p.title.toLowerCase().includes(q) || p.user.toLowerCase().includes(q) || p.tag.includes(q))
    : [];
  const filteredTags = q
    ? TAGS.filter(t => t.includes(q))
    : [];

  const hasResults = filteredPeople.length > 0 || filteredPosts.length > 0 || filteredTags.length > 0;

  const addRecent = (s: string) => {
    setRecent(prev => [s, ...prev.filter(r => r !== s)].slice(0, 6));
  };

  const handleSearch = (s: string) => {
    setQuery(s);
    addRecent(s);
  };

  const RESULT_TABS: { id: TabType; label: string }[] = [
    { id: "all",    label: "全部" },
    { id: "people", label: `用户 ${filteredPeople.length ? `(${filteredPeople.length})` : ""}` },
    { id: "posts",  label: `动态 ${filteredPosts.length ? `(${filteredPosts.length})` : ""}` },
    { id: "tags",   label: `标签 ${filteredTags.length ? `(${filteredTags.length})` : ""}` },
  ];

  return (
    <div className="flex flex-col" style={{ height: "100%", backgroundColor: "var(--neu-bg)" }}>
      {/* Search Header */}
      <div className="flex-shrink-0 flex items-center gap-3 px-5 py-3"
        style={{ borderBottom: "1px solid var(--neu-border)", backgroundColor: "var(--neu-bg)" }}>
        <motion.button whileTap={{ scale: 0.88 }} onClick={() => navigate(-1)}
          className="flex items-center justify-center rounded-full outline-none flex-shrink-0"
          style={{ width: 36, height: 36, background: "var(--neu-bg)", boxShadow: "4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light)", border: "1px solid var(--neu-border)" }}>
          <ArrowLeft style={{ width: 15, height: 15, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
        </motion.button>

        <div className="flex-1 flex items-center gap-2.5 px-4 rounded-[18px] neu-input-surface"
          style={{ height: 38 }}>
          <SearchIcon style={{ width: 14, height: 14, color: "var(--neu-text-muted)", flexShrink: 0 }} strokeWidth={1.5} />
          <input
            ref={inputRef}
            value={query}
            onChange={e => setQuery(e.target.value)}
            onFocus={() => setFocused(true)}
            onKeyDown={e => { if (e.key === "Enter" && query.trim()) addRecent(query.trim()); }}
            placeholder="搜索用户、地点、话题…"
            className="flex-1 bg-transparent outline-none"
            style={{ fontSize: 13, color: "var(--neu-text)" }}
          />
          {query && (
            <motion.button whileTap={{ scale: 0.85 }} onClick={() => setQuery("")} className="outline-none flex-shrink-0">
              <X style={{ width: 13, height: 13, color: "var(--neu-text-muted)" }} strokeWidth={2} />
            </motion.button>
          )}
        </div>
      </div>

      {/* Scrollable body */}
      <div className="flex-1 overflow-y-auto no-scrollbar">
        {/* No query: show discovery */}
        {!q && (
          <div className="px-5 pt-4 pb-6"
            style={{ maxWidth: 640, marginLeft: "auto", marginRight: "auto", width: "100%" }}>
            {/* Recent searches */}
            {recent.length > 0 && (
              <div className="mb-5">
                <div className="flex items-center justify-between mb-2.5">
                  <div className="flex items-center gap-1.5">
                    <Clock style={{ width: 13, height: 13, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
                    <p className="uppercase" style={{ fontSize: 10, letterSpacing: "0.12em", color: "var(--neu-text-muted)" }}>最近搜索</p>
                  </div>
                  <motion.button whileTap={{ scale: 0.93 }} onClick={() => setRecent([])} className="outline-none" style={{ fontSize: 12, color: "var(--neu-accent-primary)" }}>清空</motion.button>
                </div>
                <div className="flex flex-wrap gap-2">
                  {recent.map(r => (
                    <motion.button key={r} whileTap={{ scale: 0.93 }}
                      onClick={() => handleSearch(r)}
                      className="flex items-center gap-1.5 px-3.5 py-1.5 rounded-full outline-none"
                      style={{ background: "var(--neu-bg)", border: "1px solid var(--neu-border)", boxShadow: "3px 3px 6px var(--neu-shadow-dark), -3px -3px 6px var(--neu-shadow-light)", fontSize: 12, color: "var(--neu-text-muted)" }}>
                      <Clock style={{ width: 10, height: 10 }} strokeWidth={1.5} />
                      {r}
                    </motion.button>
                  ))}
                </div>
              </div>
            )}

            {/* Hot searches */}
            <div className="mb-5">
              <div className="flex items-center gap-1.5 mb-2.5">
                <TrendingUp style={{ width: 13, height: 13, color: "var(--neu-accent-primary)" }} strokeWidth={2} />
                <p className="uppercase" style={{ fontSize: 10, letterSpacing: "0.12em", color: "var(--neu-text-muted)" }}>热门搜索</p>
              </div>
              {HOT_SEARCHES.map((h, i) => (
                <motion.button key={h} whileTap={{ scale: 0.98 }}
                  onClick={() => handleSearch(h)}
                  className="flex items-center gap-3 w-full py-2.5 outline-none"
                  style={{ borderBottom: i < HOT_SEARCHES.length - 1 ? "1px solid var(--neu-border)" : "none" }}>
                  <span style={{ fontSize: 14, color: "var(--neu-accent-primary)", fontWeight: 700, width: 20, textAlign: "center" }}>{i + 1}</span>
                  <span style={{ fontSize: 14, color: "var(--neu-text)" }}>{h}</span>
                  {i < 3 && <span className="ml-auto px-2 py-0.5 rounded-full text-[10px]"
                    style={{ background: "rgba(239,68,68,0.12)", color: "#EF4444" }}>热</span>}
                </motion.button>
              ))}
            </div>

            {/* Tag suggestions */}
            <div>
              <div className="flex items-center gap-1.5 mb-2.5">
                <Hash style={{ width: 13, height: 13, color: "var(--neu-accent-primary)" }} strokeWidth={2} />
                <p className="uppercase" style={{ fontSize: 10, letterSpacing: "0.12em", color: "var(--neu-text-muted)" }}>热门话题</p>
              </div>
              <div className="flex flex-wrap gap-2">
                {TAGS.map(t => (
                  <motion.button key={t} whileTap={{ scale: 0.93 }}
                    onClick={() => handleSearch(t)}
                    className="px-3.5 py-1.5 rounded-full outline-none"
                    style={{ background: "var(--neu-bg)", color: "var(--neu-accent-primary)", fontSize: 12, boxShadow: "3px 3px 6px var(--neu-shadow-dark), -3px -3px 6px var(--neu-shadow-light)", border: "1px solid var(--neu-accent-soft-border)" }}>
                    #{t}
                  </motion.button>
                ))}
              </div>
            </div>
          </div>
        )}

        {/* Has query: show results */}
        {q && (
          <div style={{ maxWidth: 640, marginLeft: "auto", marginRight: "auto", width: "100%" }}>
            {/* Result tabs */}
            <div className="flex overflow-x-auto no-scrollbar px-5 pt-3 pb-0 gap-2">
              {RESULT_TABS.map(t => (
                <motion.button key={t.id} whileTap={{ scale: 0.92 }}
                  onClick={() => setTab(t.id)}
                  className="flex-shrink-0 px-3.5 py-1.5 rounded-full outline-none"
                  style={tab === t.id
                    ? { background: "var(--neu-accent-primary)", color: "white", fontSize: 12, boxShadow: "0 3px 10px var(--neu-glow)" }
                    : { background: "var(--neu-bg)", color: "var(--neu-text-muted)", fontSize: 12, border: "1px solid var(--neu-border)", boxShadow: "3px 3px 6px var(--neu-shadow-dark), -3px -3px 6px var(--neu-shadow-light)" }}>
                  {t.label}
                </motion.button>
              ))}
            </div>

            <div className="px-5 pt-4 pb-6 flex flex-col gap-4">
              {/* No results */}
              {!hasResults && (
                <div className="flex flex-col items-center justify-center py-16 gap-3">
                  <SearchIcon style={{ width: 36, height: 36, color: "var(--neu-text-muted)", opacity: 0.4 }} strokeWidth={1.25} />
                  <p style={{ fontSize: 15, color: "var(--neu-text)" }}>未找到相关内容</p>
                  <p style={{ fontSize: 13, color: "var(--neu-text-muted)" }}>尝试其他关键词</p>
                </div>
              )}

              {/* People results */}
              {(tab === "all" || tab === "people") && filteredPeople.length > 0 && (
                <div>
                  {tab === "all" && <p className="uppercase mb-2.5" style={{ fontSize: 10, letterSpacing: "0.14em", color: "var(--neu-text-muted)" }}>用户</p>}
                  <div className="flex flex-col gap-2.5">
                    {filteredPeople.map(person => (
                      <motion.div key={person.id} whileTap={{ scale: 0.98 }}
                        onClick={() => navigate(`/user/${person.id}`)}
                        className="flex items-center gap-3.5 px-4 py-3.5 rounded-[20px] cursor-pointer"
                        style={{ background: "var(--neu-bg)", boxShadow: "5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light)", border: "1px solid var(--neu-border)" }}>
                        <div className="rounded-full overflow-hidden flex-shrink-0" style={{ width: 42, height: 42 }}>
                          <img src={person.avatar} alt={person.name} className="w-full h-full object-cover" />
                        </div>
                        <div className="flex-1">
                          <p style={{ fontSize: 14, color: "var(--neu-text)" }}>{person.name}</p>
                          <div className="flex items-center gap-2 mt-0.5">
                            <span style={{ fontSize: 12, color: "var(--neu-text-muted)" }}>{person.handle}</span>
                            <span className="px-2 py-0.5 rounded-full" style={{ fontSize: 10, background: "var(--neu-accent-soft)", color: "var(--neu-accent-primary)" }}>#{person.tag}</span>
                          </div>
                        </div>
                        <Users style={{ width: 15, height: 15, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
                      </motion.div>
                    ))}
                  </div>
                </div>
              )}

              {/* Post results */}
              {(tab === "all" || tab === "posts") && filteredPosts.length > 0 && (
                <div>
                  {tab === "all" && <p className="uppercase mb-2.5" style={{ fontSize: 10, letterSpacing: "0.14em", color: "var(--neu-text-muted)" }}>动态</p>}
                  <div className="flex flex-col gap-2.5">
                    {filteredPosts.map(post => (
                      <motion.div key={post.id} whileTap={{ scale: 0.98 }}
                        onClick={() => navigate(`/post/${post.id}`)}
                        className="flex items-center gap-3.5 px-4 py-3.5 rounded-[20px] cursor-pointer"
                        style={{ background: "var(--neu-bg)", boxShadow: "5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light)", border: "1px solid var(--neu-border)" }}>
                        <div className="rounded-[12px] overflow-hidden flex-shrink-0" style={{ width: 42, height: 42 }}>
                          <img src={post.image} alt={post.title} className="w-full h-full object-cover" />
                        </div>
                        <div className="flex-1 min-w-0">
                          <p className="truncate" style={{ fontSize: 14, color: "var(--neu-text)" }}>{post.title}</p>
                          <div className="flex items-center gap-1.5 mt-0.5">
                            <span style={{ fontSize: 12, color: "var(--neu-text-muted)" }}>{post.user}</span>
                            <span style={{ fontSize: 10, color: "var(--neu-text-muted)", opacity: 0.5 }}>·</span>
                            <span style={{ fontSize: 11, color: "var(--neu-accent-primary)" }}>❤️ {post.likes}</span>
                          </div>
                        </div>
                        <FileText style={{ width: 15, height: 15, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
                      </motion.div>
                    ))}
                  </div>
                </div>
              )}

              {/* Tag results */}
              {(tab === "all" || tab === "tags") && filteredTags.length > 0 && (
                <div>
                  {tab === "all" && <p className="uppercase mb-2.5" style={{ fontSize: 10, letterSpacing: "0.14em", color: "var(--neu-text-muted)" }}>标签</p>}
                  <div className="flex flex-wrap gap-2">
                    {filteredTags.map(t => (
                      <motion.button key={t} whileTap={{ scale: 0.93 }}
                        className="px-4 py-2 rounded-full outline-none"
                        style={{ background: "var(--neu-bg)", color: "var(--neu-accent-primary)", fontSize: 13, boxShadow: "4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light)", border: "1px solid var(--neu-accent-soft-border)" }}>
                        #{t}
                      </motion.button>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}