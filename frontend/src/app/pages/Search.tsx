import React, { useState, useEffect, useRef, useCallback } from "react";
import { useNavigate } from "react-router";
import { ROUTES } from "../lib/routes";
import { ArrowLeft, Search as SearchIcon, TrendingUp, Hash, Users, FileText, MapPin, Clock, X } from "lucide-react";
import { motion } from "motion/react";
import { usePlatform } from "../contexts/PlatformContext";
import { ActionIconButton } from "../components/shared/ActionIconButton";
import { searchAll } from "../../api";

type TabType = "all" | "people" | "posts" | "tags";

interface SearchResultUser { user_id: string; nickname: string; avatar_url?: string; tags?: string[] }
interface SearchResultMoment { moment_id: string; description?: string; user?: SearchResultUser; tag?: string; likes_count?: number }

const HOT_SEARCHES: string[] = [];

export function Search() {
  const navigate = useNavigate();
  const { platform } = usePlatform();
  const isDesktop = platform === "desktop";
  const shellStyle = { maxWidth: isDesktop ? 1040 : 640, marginLeft: "auto", marginRight: "auto", width: "100%" };
  const inputRef = useRef<HTMLInputElement>(null);
  const [query, setQuery] = useState("");
  const [tab, setTab] = useState<TabType>("all");
  const [recent, setRecent] = useState<string[]>(() => {
    try { return JSON.parse(localStorage.getItem("search_recent") ?? "[]"); } catch { return []; }
  });
  const [focused, setFocused] = useState(true);
  const [results, setResults] = useState<{ users: SearchResultUser[]; moments: SearchResultMoment[]; tags: string[] }>({ users: [], moments: [], tags: [] });
  const [loading, setLoading] = useState(false);

  useEffect(() => { inputRef.current?.focus(); }, []);
  useEffect(() => { localStorage.setItem("search_recent", JSON.stringify(recent)); }, [recent]);

  const q = query.toLowerCase().trim();

  const doSearch = useCallback(async (term: string) => {
    if (!term) { setResults({ users: [], moments: [], tags: [] }); return; }
    setLoading(true);
    try {
      const res = await searchAll({ keyword: term });
      if (res.code === 0 && res.data) {
        const d = res.data as any;
        const rawUsers = d.users?.items ?? d.users ?? [];
        const rawMoments = d.moments?.items ?? d.moments ?? [];
        // searchAll returns nested UserItem:{user:{user_id,nickname,avatar_url}} and MomentItem:{user:{...}}.
        const users = rawUsers.map((u: any) => {
          const inner = u.user ?? u;
          return { user_id: inner.user_id ?? "", nickname: inner.nickname ?? "", avatar_url: inner.avatar_url };
        });
        const moments = rawMoments.map((m: any) => ({
          ...m,
          user: m.user ?? (m as any).user_info,
        }));
        setResults({ users, moments, tags: [] });
      } else if (res.message) {
        setResults({ users: [], moments: [], tags: [] });
      }
    } catch (err) {
      setResults({ users: [], moments: [], tags: [] });
    }
    finally { setLoading(false); }
  }, []);

  useEffect(() => {
    const timer = setTimeout(() => doSearch(q), 300);
    return () => clearTimeout(timer);
  }, [q, doSearch]);

  const addRecent = (s: string) => {
    setRecent(prev => [s, ...prev.filter(r => r !== s)].slice(0, 6));
  };

  const handleSearch = (s: string) => {
    setQuery(s);
    addRecent(s);
  };

  const hasResults = results.users.length > 0 || results.moments.length > 0 || results.tags.length > 0;

  const RESULT_TABS: { id: TabType; label: string }[] = [
    { id: "all",    label: "全部" },
    { id: "people", label: `用户 ${results.users.length ? `(${results.users.length})` : ""}` },
    { id: "posts",  label: `动态 ${results.moments.length ? `(${results.moments.length})` : ""}` },
    { id: "tags",   label: `标签 ${results.tags.length ? `(${results.tags.length})` : ""}` },
  ];

  return (
    <div className="flex flex-col" style={{ height: "100%", backgroundColor: "var(--neu-bg)" }}>
      {/* Search Header */}
      <div className="flex-shrink-0" style={{ borderBottom: "1px solid var(--neu-border)", backgroundColor: "var(--neu-bg)" }}>
        <div className="flex items-center gap-3 px-5 py-3" style={shellStyle}>
        <ActionIconButton icon={ArrowLeft} label="返回" onClick={() => navigate(-1)} />

        <div className="flex-1 flex items-center gap-2.5 px-4 rounded-[18px] neu-input-surface"
          style={{ height: 38 }}>
          <SearchIcon style={{ width: 14, height: 14, color: "var(--neu-text-muted)", flexShrink: 0 }} strokeWidth={1.5} />
          <input
            ref={inputRef}
            value={query}
            onChange={e => setQuery(e.target.value)}
            onFocus={() => setFocused(true)}
            onKeyDown={e => { if (e.key === "Enter" && query.trim()) handleSearch(query.trim()); }}
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
      </div>

      {/* Scrollable body */}
      <div className="flex-1 overflow-y-auto no-scrollbar">
        {/* No query: show discovery */}
        {!q && (
          <div className="px-5 pt-4 pb-6" style={shellStyle}>
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
                      className="flex items-center gap-1.5 px-3.5 py-1.5 rounded-full outline-none neu-surface-ghost"
                      style={{ fontSize: 12, color: "var(--neu-text-muted)" }}>
                      <Clock style={{ width: 10, height: 10 }} strokeWidth={1.5} />
                      {r}
                    </motion.button>
                  ))}
                </div>
              </div>
            )}

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
                    style={{ background: "color-mix(in srgb, var(--semantic-hot) 12%, transparent)", color: "var(--semantic-hot)" }}>热</span>}
                </motion.button>
              ))}
            </div>
          </div>
        )}

        {/* Has query: show results */}
        {q && (
          <div style={shellStyle}>
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
              {loading && (
                <div className="flex items-center justify-center py-12">
                  <p style={{ fontSize: 13, color: "var(--neu-text-muted)" }}>搜索中...</p>
                </div>
              )}

              {!loading && !hasResults && (
                <div className="flex flex-col items-center justify-center py-16 gap-3">
                  <SearchIcon style={{ width: 36, height: 36, color: "var(--neu-text-muted)", opacity: 0.4 }} strokeWidth={1.25} />
                  <p style={{ fontSize: 15, color: "var(--neu-text)" }}>未找到相关内容</p>
                  <p style={{ fontSize: 13, color: "var(--neu-text-muted)" }}>尝试其他关键词</p>
                </div>
              )}

              {(tab === "all" || tab === "people") && results.users.length > 0 && (
                <div>
                  {tab === "all" && <p className="uppercase mb-2.5" style={{ fontSize: 10, letterSpacing: "0.14em", color: "var(--neu-text-muted)" }}>用户</p>}
                  <div className="flex flex-col gap-2.5">
                    {results.users.map(user => (
                      <motion.div key={user.user_id} whileTap={{ scale: 0.98 }}
                        onClick={() => navigate(ROUTES.user(user.user_id))}
                        className="app-list-row app-list-row--rich neu-row-hover cursor-pointer rounded-[20px] px-4 neu-surface-flat">
                        <div className="rounded-full overflow-hidden flex-shrink-0" style={{ width: 42, height: 42 }}>
                          <img src={user.avatar_url ?? "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80"} alt={user.nickname} className="w-full h-full object-cover" />
                        </div>
                        <div className="flex-1">
                          <p style={{ fontSize: 14, color: "var(--neu-text)" }}>{user.nickname}</p>
                          {user.tags && user.tags.length > 0 && (
                            <span className="px-2 py-0.5 rounded-full" style={{ fontSize: 10, background: "var(--neu-accent-soft)", color: "var(--neu-accent-primary)" }}>#{user.tags[0]}</span>
                          )}
                        </div>
                        <Users style={{ width: 15, height: 15, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
                      </motion.div>
                    ))}
                  </div>
                </div>
              )}

              {(tab === "all" || tab === "posts") && results.moments.length > 0 && (
                <div>
                  {tab === "all" && <p className="uppercase mb-2.5" style={{ fontSize: 10, letterSpacing: "0.14em", color: "var(--neu-text-muted)" }}>动态</p>}
                  <div className="flex flex-col gap-2.5">
                    {results.moments.map(moment => (
                      <motion.div key={moment.moment_id} whileTap={{ scale: 0.98 }}
                        onClick={() => navigate(ROUTES.moment(moment.moment_id))}
                        className="app-list-row app-list-row--rich neu-row-hover cursor-pointer rounded-[20px] px-4 neu-surface-flat">
                        <div className="rounded-[12px] overflow-hidden flex-shrink-0" style={{ width: 42, height: 42, background: "var(--neu-accent-soft)" }} />
                        <div className="flex-1 min-w-0">
                          <p className="truncate" style={{ fontSize: 14, color: "var(--neu-text)" }}>{moment.description ?? "无标题"}</p>
                          <div className="flex items-center gap-1.5 mt-0.5">
                            <span style={{ fontSize: 12, color: "var(--neu-text-muted)" }}>{moment.user?.nickname ?? ""}</span>
                            {moment.likes_count !== undefined && (
                              <>
                                <span style={{ fontSize: 10, color: "var(--neu-text-muted)", opacity: 0.5 }}>·</span>
                                <span style={{ fontSize: 11, color: "var(--neu-accent-primary)" }}>❤️ {moment.likes_count}</span>
                              </>
                            )}
                          </div>
                        </div>
                        <FileText style={{ width: 15, height: 15, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
                      </motion.div>
                    ))}
                  </div>
                </div>
              )}

              {(tab === "all" || tab === "tags") && results.tags.length > 0 && (
                <div>
                  {tab === "all" && <p className="uppercase mb-2.5" style={{ fontSize: 10, letterSpacing: "0.14em", color: "var(--neu-text-muted)" }}>标签</p>}
                  <div className="flex flex-wrap gap-2">
                    {results.tags.map(t => (
                      <motion.button key={t} whileTap={{ scale: 0.93 }}
                        className="px-4 py-2 rounded-full outline-none neu-theme-chip"
                        style={{ fontSize: 13 }}>
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
