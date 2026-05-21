import { useState, useMemo } from "react";
import { ArrowLeft, Users, Search, X, Plus, Globe } from "lucide-react";
import { motion, AnimatePresence } from "motion/react";
import { useNavigate } from "react-router";
import { CreateGroupSheet } from "../components/ui/CreateGroupSheet";
import { useFocusTrap } from "../hooks/useFocusTrap";
import { ListSkeleton } from "../components/ui/skeleton";
import { useHomeData } from "@/lib/hooks/use-home-data";
import { useGroupToggle } from "@/lib/hooks/use-group-toggle";
import { useDocumentTitle } from "@/lib/hooks/use-document-title";
import { goBack } from "@/lib/utils/navigation";
import { normalizeAvatar, formatDistance } from "@/lib/utils";

export function NearbyGroups() {
  const navigate = useNavigate();
  const { nearbyGroups, loading, error, refresh } = useHomeData();
  const { joinedIds, toggle: toggleJoin } = useGroupToggle();
  const [search, setSearch] = useState("");
  const [showCreate, setShowCreate] = useState(false);
  useDocumentTitle("附近群组");

  const tags = useMemo(
    () => ["全部", ...new Set(nearbyGroups.map((g) => g.tags ?? []).flat().filter(Boolean))],
    [nearbyGroups],
  );
  const [tagFilt, setTagFilt] = useState("全部");

  const filtered = useMemo(() => {
    let list = nearbyGroups.filter(
      (g) => search === "" || g.group.name.toLowerCase().includes(search.toLowerCase()),
    );
    if (tagFilt !== "全部") list = list.filter((g) => (g.tags ?? []).includes(tagFilt));
    return list;
  }, [nearbyGroups, search, tagFilt]);

  return (
    <>
      <div className="sticky top-0 z-10 pt-3 pb-3"
        style={{ backgroundColor: "var(--neu-bg)", borderBottom: "1px solid var(--neu-border)" }}>
        <div className="px-5" style={{ maxWidth: 640, marginLeft: "auto", marginRight: "auto", width: "100%" }}>
          <div className="flex items-center gap-3 mb-3">
            <motion.button whileTap={{ scale: 0.88 }} onClick={() => goBack(navigate)} aria-label="返回"
              className="flex items-center justify-center rounded-full outline-none neu-surface-flat"
              style={{ width: 40, height: 40 }}>
              <ArrowLeft style={{ width: 15, height: 15, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
            </motion.button>
            <h1 className="flex-1" style={{ fontSize: 18, color: "var(--neu-text)" }}>附近群组</h1>
            <motion.button whileTap={{ scale: 0.88 }} onClick={() => setShowCreate(true)} aria-label="创建群组"
              className="flex items-center justify-center rounded-full outline-none"
              style={{ width: 40, height: 40, background: "var(--neu-accent-primary)", color: "var(--text-on-accent, white)", boxShadow: "0 4px 12px var(--neu-glow)" }}>
              <Plus style={{ width: 15, height: 15 }} strokeWidth={2} />
            </motion.button>
          </div>

          <div className="flex items-center gap-3 px-4 py-2.5 rounded-[14px] neu-input-surface mb-3">
            <Search style={{ width: 14, height: 14, color: "var(--neu-text-muted)", flexShrink: 0 }} strokeWidth={1.5} />
            <input value={search} onChange={(e) => setSearch(e.target.value)}
              aria-label="搜索群组"
              placeholder="搜索群组…" className="flex-1 bg-transparent outline-none"
              style={{ fontSize: 13, color: "var(--neu-text)" }} />
            {search && <motion.button whileTap={{ scale: 0.85 }} onClick={() => setSearch("")} aria-label="清除搜索" className="outline-none">
              <X style={{ width: 12, height: 12, color: "var(--neu-text-muted)" }} strokeWidth={2} />
            </motion.button>}
          </div>

          <div className="flex gap-1.5 overflow-x-auto no-scrollbar">
            {tags.slice(0, 12).map((t) => {
              const active = tagFilt === t;
              return (
                <motion.button key={t} whileTap={{ scale: 0.94 }} onClick={() => setTagFilt(t)}
                  className="flex-shrink-0 px-3 py-1 rounded-full outline-none transition-all"
                  style={active
                    ? { background: "var(--neu-accent-primary)", color: "var(--text-on-accent, white)", fontSize: 12 }
                    : { background: "var(--neu-bg)", color: "var(--neu-text-muted)", fontSize: 12, border: "1px solid var(--neu-border)" }}>
                  {t}
                </motion.button>
              );
            })}
          </div>
        </div>
      </div>

      <div className="flex flex-col px-5 pt-4 pb-6 gap-3"
        style={{ maxWidth: 640, marginLeft: "auto", marginRight: "auto", width: "100%" }}>
        {loading ? (
          <ListSkeleton rows={4} />
        ) : error && filtered.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12 gap-3">
            <p style={{ fontSize: 14, color: "var(--neu-text-muted)" }}>{error}</p>
            <motion.button whileTap={{ scale: 0.94 }} onClick={refresh}
              className="px-4 py-2 rounded-full outline-none"
              style={{ background: "var(--neu-accent-primary)", color: "var(--text-on-accent, white)", fontSize: 13 }}>
              重试
            </motion.button>
          </div>
        ) : filtered.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12 gap-3">
            <Users style={{ width: 36, height: 36, color: "var(--neu-text-muted)", opacity: 0.4 }} strokeWidth={1.25} />
            <p style={{ fontSize: 14, color: "var(--neu-text-muted)" }}>暂无附近群组</p>
          </div>
        ) : (
          filtered.map((g) => {
            const joined = joinedIds.has(g.group.group_id) || !!g.my_state;
            return (
              <motion.div key={g.group.group_id}
                onClick={() => navigate(`/messages/group-${g.group.group_id}`)}
                className="rounded-[20px] overflow-hidden cursor-pointer"
                style={{ background: "var(--neu-bg)", boxShadow: "6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light)", border: "1px solid var(--neu-border)" }}>
                <div className="relative" style={{ height: "clamp(60px, 12vw, 90px)" }}>
                  {g.group.avatar_url ? (
                    <img src={normalizeAvatar(g.group.avatar_url)} alt={g.group.name} className="w-full h-full object-cover" />
                  ) : (
                    <div className="w-full h-full" style={{ background: "linear-gradient(135deg, var(--neu-border), var(--neu-bg))" }} />
                  )}
                  <div className="absolute inset-0" style={{ background: "linear-gradient(to bottom, transparent 30%, rgba(0,0,0,0.5) 100%)" }} />
                  <div className="absolute bottom-2 left-3 flex items-center gap-1.5">
                    <span className="px-2 py-0.5 rounded-full"
                      style={{ fontSize: 10, background: "rgba(0,0,0,0.5)", color: "white", backdropFilter: "blur(8px)" }}>
                      {g.join_method === "invite_only" ? "邀请制" : g.join_method === "approval" ? "需审批" : "公开"}
                    </span>
                    <span className="px-2 py-0.5 rounded-full"
                      style={{ fontSize: 10, background: "rgba(99,102,241,0.7)", color: "white", backdropFilter: "blur(8px)" }}>
                      {formatDistance(g.location?.distance)}
                    </span>
                  </div>
                </div>

                <div className="p-3.5">
                  <div className="flex items-center justify-between mb-1">
                    <p className="truncate flex-1 min-w-0" style={{ fontSize: 14, color: "var(--neu-text)", fontWeight: 500 }}>{g.group.name}</p>
                    <div className="flex items-center gap-2 flex-shrink-0">
                      <div className="flex items-center gap-1">
                        <Users style={{ width: 11, height: 11, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
                        <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>{g.group.members_count}</span>
                      </div>
                      {g.friends_in_group ? (
                        <span style={{ fontSize: 10, color: "var(--neu-accent-primary)" }}>
                          {g.friends_in_group}位好友在群
                        </span>
                      ) : null}
                    </div>
                  </div>
                  <p style={{ fontSize: 12, color: "var(--neu-text-muted)", marginTop: 4, lineHeight: 1.5 }}>
                    {g.description || ""}
                  </p>
                  <div className="flex items-center justify-between mt-3">
                    <div className="flex items-center gap-1.5">
                      <Globe style={{ width: 11, height: 11, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
                      <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>
                        {g.location?.location_name ?? "附近"}
                      </span>
                    </div>
                    <motion.button whileTap={{ scale: 0.93 }}
                      onClick={(e) => { e.stopPropagation(); toggleJoin(g.group.group_id); }}
                      className="px-4 py-1.5 rounded-full outline-none"
                      style={joined
                        ? { background: "transparent", color: "var(--neu-text-muted)", border: "1px solid var(--neu-border)", fontSize: 12 }
                        : { background: "var(--neu-accent-primary)", color: "var(--text-on-accent, white)", fontSize: 12, boxShadow: "0 3px 8px var(--neu-glow)" }}>
                      {joined ? "已加入" : "加入"}
                    </motion.button>
                  </div>
                </div>
              </motion.div>
            );
          })
        )}
      </div>

      <GroupSheetOverlay show={showCreate} onClose={() => setShowCreate(false)} />
    </>
  );
}

function GroupSheetOverlay({ show, onClose }: { show: boolean; onClose: () => void }) {
  const trapRef = useFocusTrap(show, onClose);
  return (
    <AnimatePresence>
      {show && (
        <>
          <motion.div key="backdrop" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
            transition={{ duration: 0.2 }}
            className="fixed inset-0 z-40"
            style={{ background: "rgba(0,0,0,0.35)", backdropFilter: "blur(3px)" }}
            onClick={onClose} />
          <motion.div key="sheet"
            ref={trapRef}
            role="dialog"
            aria-modal="true"
            aria-label="创建群组"
            initial={{ y: "100%" }} animate={{ y: 0 }} exit={{ y: "100%" }}
            transition={{ type: "spring", damping: 30, stiffness: 260, mass: 0.9 }}
            className="fixed bottom-0 left-0 right-0 z-50 overflow-hidden"
            style={{ height: "92%", borderRadius: "28px 28px 0 0", backgroundColor: "var(--neu-bg)", boxShadow: "0 -8px 40px rgba(0,0,0,0.18)" }}>
            <CreateGroupSheet onClose={onClose} />
          </motion.div>
        </>
      )}
    </AnimatePresence>
  );
}
