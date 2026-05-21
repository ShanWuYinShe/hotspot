import { useState, useEffect, useRef } from "react";
import { useNavigate } from "react-router";
import { ArrowLeft, Search as SearchIcon, Users, FileText, Clock, X } from "lucide-react";
import { motion } from "motion/react";

import { useSearchData } from "@/lib/hooks/use-search-data";
import { useDocumentTitle } from "@/lib/hooks/use-document-title";
import { goBack } from "@/lib/utils/navigation";
import { ListSkeleton } from "../components/ui/skeleton";
import type { UserSuggestItem, MomentSuggestItem, TagSuggestItem } from "@/lib/types/api";

type TabType = "all" | "people" | "posts" | "tags";

const RECENT_KEY = "hotspot_recent_searches";

function loadRecent(): string[] {
  try {
    return JSON.parse(localStorage.getItem(RECENT_KEY) || "[]");
  } catch { return []; }
}

function saveRecent(items: string[]) {
  try { localStorage.setItem(RECENT_KEY, JSON.stringify(items)); } catch {}
}

export function Search() {
  const navigate = useNavigate();
  const inputRef = useRef<HTMLInputElement>(null);
  const [query, setQuery] = useState("");
  const [tab, setTab] = useState<TabType>("all");
  const [recent, setRecent] = useState(loadRecent);
  const { results, loading, search, clear } = useSearchData();
  useDocumentTitle("搜索");

  useEffect(() => { inputRef.current?.focus(); }, []);

  const q = query.toLowerCase().trim();

  useEffect(() => {
    search(q);
    if (!q) clear();
  }, [q, search, clear]);

  const addRecent = (s: string) => {
    setRecent(prev => {
      const next = [s, ...prev.filter(r => r !== s)].slice(0, 6);
      saveRecent(next);
      return next;
    });
  };

  const handleSearch = (s: string) => {
    setQuery(s);
    addRecent(s);
  };

  const people = results.filter(r => r.type === "user");
  const moments = results.filter(r => r.type === "moment");
  const tags = results.filter(r => r.type === "tag");
  const hasResults = results.length > 0;

  const RESULT_TABS: { id: TabType; label: string }[] = [
    { id: "all",    label: "全部" },
    { id: "people", label: `用户${people.length ? ` (${people.length})` : ""}` },
    { id: "posts",  label: `动态${moments.length ? ` (${moments.length})` : ""}` },
    { id: "tags",   label: `标签${tags.length ? ` (${tags.length})` : ""}` },
  ];

  const renderUserItem = (item: UserSuggestItem) => (
    <motion.div key={item.user_id} whileTap={{ scale: 0.98 }}
      onClick={() => navigate(`/user/${item.user_id}`)}
      className="flex items-center gap-3.5 px-4 py-3.5 rounded-[20px] cursor-pointer"
      style={{ background: "var(--neu-bg)", boxShadow: "5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light)", border: "1px solid var(--neu-border)" }}>
      <div className="rounded-full overflow-hidden flex-shrink-0" style={{ width: 42, height: 42 }}>
        {item.avatar_url
          ? <img src={item.avatar_url} alt={item.nickname} className="w-full h-full object-cover" />
          : <div className="w-full h-full flex items-center justify-center" style={{ background: "var(--neu-accent-soft)" }}>
              <span style={{ color: "var(--neu-accent-primary)" }}>{item.nickname.charAt(0)}</span>
            </div>
        }
      </div>
      <div className="flex-1 min-w-0">
        <p className="truncate" style={{ fontSize: 14, color: "var(--neu-text)" }}>{item.nickname}</p>
        {item.bio && <p style={{ fontSize: 12, color: "var(--neu-text-muted)", marginTop: 1 }}>{item.bio}</p>}
      </div>
      <Users style={{ width: 15, height: 15, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
    </motion.div>
  );

  const renderMomentItem = (item: MomentSuggestItem) => (
    <motion.div key={item.moment_id} whileTap={{ scale: 0.98 }}
      onClick={() => navigate(`/post/${item.moment_id}`)}
      className="flex items-center gap-3.5 px-4 py-3.5 rounded-[20px] cursor-pointer"
      style={{ background: "var(--neu-bg)", boxShadow: "5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light)", border: "1px solid var(--neu-border)" }}>
      <div className="rounded-[12px] overflow-hidden flex-shrink-0 flex items-center justify-center"
        style={{ width: 42, height: 42, background: "var(--neu-accent-soft)" }}>
        <FileText style={{ width: 18, height: 18, color: "var(--neu-accent-primary)" }} strokeWidth={1.5} />
      </div>
      <div className="flex-1 min-w-0">
        <p className="truncate" style={{ fontSize: 14, color: "var(--neu-text)" }}>{item.description_preview}</p>
        <div className="flex items-center gap-1.5 mt-0.5 min-w-0">
          <span className="truncate" style={{ fontSize: 12, color: "var(--neu-text-muted)" }}>{item.author.nickname}</span>
        </div>
      </div>
      <FileText style={{ width: 15, height: 15, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
    </motion.div>
  );

  return (
    <div className="flex flex-col" style={{ height: "100%", backgroundColor: "var(--neu-bg)" }}>
      {/* Search Header */}
      <div className="flex-shrink-0 flex items-center gap-3 px-5 py-3"
        style={{ borderBottom: "1px solid var(--neu-border)", backgroundColor: "var(--neu-bg)" }}>
        <motion.button whileTap={{ scale: 0.88 }} onClick={() => goBack(navigate)} aria-label="返回"
          className="flex items-center justify-center rounded-full outline-none flex-shrink-0"
          style={{ width: 36, height: 36, background: "var(--neu-bg)", boxShadow: "4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light)", border: "1px solid var(--neu-border)" }}>
          <ArrowLeft style={{ width: 15, height: 15, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
        </motion.button>

        <div className="flex-1 flex items-center gap-2.5 px-4 rounded-[18px] neu-input-surface"
          style={{ height: 38 }}>
          <SearchIcon style={{ width: 14, height: 14, color: "var(--neu-text-muted)", flexShrink: 0 }} strokeWidth={1.5} />
          <input
            ref={inputRef}
            aria-label="搜索"
            value={query}
            onChange={e => setQuery(e.target.value)}
            onFocus={() => {}}
            onKeyDown={e => { if (e.key === "Enter" && query.trim()) addRecent(query.trim()); if (e.key === "Escape") { if (query) { setQuery(""); clear(); } else { goBack(navigate); } } }}
            placeholder="搜索用户、地点、话题…"
            className="flex-1 bg-transparent outline-none"
            style={{ fontSize: 13, color: "var(--neu-text)" }}
          />
          {query && (
            <motion.button whileTap={{ scale: 0.85 }} onClick={() => { setQuery(""); clear(); }} aria-label="清除搜索" className="outline-none flex-shrink-0">
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
                  <motion.button whileTap={{ scale: 0.93 }} onClick={() => { setRecent([]); saveRecent([]); }} className="outline-none" style={{ fontSize: 12, color: "var(--neu-accent-primary)" }}>清空</motion.button>
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

            {/* Empty state */}
            {recent.length === 0 && (
              <div className="flex flex-col items-center justify-center py-16 gap-3">
                <SearchIcon style={{ width: 36, height: 36, color: "var(--neu-text-muted)", opacity: 0.4 }} strokeWidth={1.25} />
                <p style={{ fontSize: 15, color: "var(--neu-text)" }}>搜索用户、群组、话题</p>
                <p style={{ fontSize: 13, color: "var(--neu-text-muted)" }}>输入关键词开始搜索</p>
              </div>
            )}
          </div>
        )}

        {/* Has query: show results */}
        {q && (
          <div style={{ maxWidth: 640, marginLeft: "auto", marginRight: "auto", width: "100%" }}>
            {/* Result tabs */}
            <div className="flex overflow-x-auto no-scrollbar px-5 pt-3 pb-0 gap-2">
              {RESULT_TABS.map(t => (
                <motion.button key={t.id} whileTap={{ scale: 0.94 }}
                  onClick={() => setTab(t.id)}
                  className="flex-shrink-0 px-3.5 py-1.5 rounded-full outline-none"
                  style={tab === t.id
                    ? { background: "var(--neu-accent-primary)", color: "var(--text-on-accent, white)", fontSize: 12, boxShadow: "0 3px 10px var(--neu-glow)" }
                    : { background: "var(--neu-bg)", color: "var(--neu-text-muted)", fontSize: 12, border: "1px solid var(--neu-border)", boxShadow: "3px 3px 6px var(--neu-shadow-dark), -3px -3px 6px var(--neu-shadow-light)" }}>
                  {t.label}
                </motion.button>
              ))}
            </div>

            <div className="px-5 pt-4 pb-6 flex flex-col gap-4">
              {/* Loading */}
              {loading && (
                <ListSkeleton rows={4} />
              )}

              {/* No results */}
              {!loading && !hasResults && (
                <div className="flex flex-col items-center justify-center py-16 gap-3">
                  <SearchIcon style={{ width: 36, height: 36, color: "var(--neu-text-muted)", opacity: 0.4 }} strokeWidth={1.25} />
                  <p style={{ fontSize: 15, color: "var(--neu-text)" }}>未找到相关内容</p>
                  <p style={{ fontSize: 13, color: "var(--neu-text-muted)" }}>尝试其他关键词</p>
                </div>
              )}

              {/* People results */}
              {!loading && (tab === "all" || tab === "people") && people.length > 0 && (
                <div>
                  {tab === "all" && <p className="uppercase mb-2.5" style={{ fontSize: 10, letterSpacing: "0.14em", color: "var(--neu-text-muted)" }}>用户</p>}
                  <div className="flex flex-col gap-2.5">
                    {people.map(r => renderUserItem(r.payload as UserSuggestItem))}
                  </div>
                </div>
              )}

              {/* Moment results */}
              {!loading && (tab === "all" || tab === "posts") && moments.length > 0 && (
                <div>
                  {tab === "all" && <p className="uppercase mb-2.5" style={{ fontSize: 10, letterSpacing: "0.14em", color: "var(--neu-text-muted)" }}>动态</p>}
                  <div className="flex flex-col gap-2.5">
                    {moments.map(r => renderMomentItem(r.payload as MomentSuggestItem))}
                  </div>
                </div>
              )}

              {/* Tag results */}
              {!loading && (tab === "all" || tab === "tags") && tags.length > 0 && (
                <div>
                  {tab === "all" && <p className="uppercase mb-2.5" style={{ fontSize: 10, letterSpacing: "0.14em", color: "var(--neu-text-muted)" }}>标签</p>}
                  <div className="flex flex-wrap gap-2">
                    {tags.map(r => {
                      const t = r.payload as TagSuggestItem;
                      return (
                        <motion.button key={t.name} whileTap={{ scale: 0.93 }}
                          className="px-4 py-2 rounded-full outline-none"
                          style={{ background: "var(--neu-bg)", color: "var(--neu-accent-primary)", fontSize: 13, boxShadow: "4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light)", border: "1px solid var(--neu-accent-soft-border)" }}>
                          #{t.name}
                          <span style={{ fontSize: 11, color: "var(--neu-text-muted)", marginLeft: 4 }}>{t.usage_count}</span>
                        </motion.button>
                      );
                    })}
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
