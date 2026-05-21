/**
 * NewConversationSheet — bottom sheet for:
 *   1. 添加好友  (tab: "好友")
 *   2. 新建私聊  (tab: "私聊")
 *   3. 新建群聊  → triggers CreateGroupSheet
 */
import { useState } from "react";
import { Search, X, UserPlus, MessageCircle, Users, Check } from "lucide-react";
import { motion, AnimatePresence } from "motion/react";
import { useHoverConfig } from "../../hooks/useHoverConfig";

type Tab = "add" | "chat";

const SUGGESTED = [
  { id: 1, name: "Liam",    subtitle: "@liam.photo · 摄影师",  avatar: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80",  added: false },
  { id: 2, name: "Sophia",  subtitle: "@sophia_t · 旅行博主",   avatar: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80",  added: false },
  { id: 3, name: "Marco",   subtitle: "@marco_m · 音乐人",      avatar: "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=80",  added: false },
  { id: 4, name: "Aiko",    subtitle: "@aiko_e · 美食达人",     avatar: "https://images.unsplash.com/photo-1758613172074-7275c3455034?w=80",  added: false },
  { id: 5, name: "Chen",    subtitle: "@chen_fit · 健身教练",   avatar: "https://images.unsplash.com/photo-1535713875002-d1d0cf377fde?w=80",  added: false },
];

const CONTACTS = [
  { id: 101, name: "Noah",       avatar: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80",  online: true  },
  { id: 102, name: "Isabella",   avatar: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80",  online: true  },
  { id: 103, name: "Kai",        avatar: "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=80",  online: false },
  { id: 104, name: "设计小队",   avatar: "https://images.unsplash.com/photo-1763467669450-e69029c1484d?w=80",  online: false },
];

interface Props {
  defaultTab?: Tab;
  onClose: () => void;
  onNewChat?: (contactId: number) => void;
  onCreateGroup?: () => void;
}

export function NewConversationSheet({ defaultTab = "chat", onClose, onNewChat, onCreateGroup }: Props) {
  const hover = useHoverConfig();
  const [tab, setTab] = useState<Tab>(defaultTab);
  const [searchQ, setSearchQ] = useState("");
  const [addedIds, setAddedIds] = useState<Set<number>>(new Set());

  const filteredSuggested = SUGGESTED.filter(u =>
    !searchQ || u.name.toLowerCase().includes(searchQ.toLowerCase()) || u.subtitle.includes(searchQ)
  );
  const filteredContacts = CONTACTS.filter(c =>
    !searchQ || c.name.toLowerCase().includes(searchQ.toLowerCase())
  );

  return (
    <div className="flex flex-col h-full" style={{ background: "var(--neu-bg)" }}>
      {/* Handle */}
      <div className="flex justify-center pt-3 pb-1 flex-shrink-0">
        <div className="rounded-full" style={{ width: 36, height: 4, background: "var(--neu-border)" }} />
      </div>

      {/* Header */}
      <div className="flex items-center justify-between px-5 py-3 flex-shrink-0"
        style={{ borderBottom: "1px solid var(--neu-border)" }}>
        <h2 style={{ fontSize: 17, color: "var(--neu-text)" }}>
          {tab === "add" ? "添加好友" : "新建会话"}
        </h2>
        <motion.button whileTap={hover.tap} onClick={onClose}
          className="flex items-center justify-center rounded-full outline-none"
          style={{ width: 32, height: 32, background: "var(--neu-bg)", boxShadow: "3px 3px 7px var(--neu-shadow-dark), -3px -3px 7px var(--neu-shadow-light)", border: "1px solid var(--neu-border)" }}>
          <X style={{ width: 14, height: 14, color: "var(--neu-text-muted)" }} strokeWidth={2} />
        </motion.button>
      </div>

      {/* Tab switcher */}
      <div className="flex px-5 pt-3 pb-2 gap-2 flex-shrink-0">
        {[
          { id: "add" as Tab,  icon: UserPlus,        label: "添加好友" },
          { id: "chat" as Tab, icon: MessageCircle,   label: "开始私聊" },
        ].map(t => (
          <motion.button key={t.id} whileTap={hover.tap}
            onClick={() => { setTab(t.id); setSearchQ(""); }}
            className="flex items-center gap-2 px-4 py-2 rounded-full outline-none"
            style={tab === t.id
              ? { background: "var(--neu-accent-primary)", color: "var(--text-on-accent, white)", fontSize: 13, fontWeight: 500, boxShadow: "0 4px 12px var(--neu-glow)" }
              : { background: "var(--neu-bg)", color: "var(--neu-text-muted)", fontSize: 13, boxShadow: "3px 3px 7px var(--neu-shadow-dark), -3px -3px 7px var(--neu-shadow-light)", border: "1px solid var(--neu-border)" }
            }
          >
            <t.icon style={{ width: 13, height: 13 }} strokeWidth={1.75} />
            {t.label}
          </motion.button>
        ))}
      </div>

      {/* Search bar */}
      <div className="px-5 pb-3 flex-shrink-0">
        <div className="flex items-center gap-2.5 px-4 rounded-[14px] neu-input-surface" style={{ height: 40 }}>
          <Search style={{ width: 14, height: 14, color: "var(--neu-text-muted)", flexShrink: 0 }} strokeWidth={1.5} />
          <input className="flex-1 bg-transparent outline-none"
            aria-label="搜索用户"
            style={{ fontSize: 13, color: "var(--neu-text)" }}
            placeholder={tab === "add" ? "搜索用户名或手机号…" : "搜索联系人…"}
            value={searchQ}
            onChange={e => setSearchQ(e.target.value)}
          />
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto no-scrollbar px-5 pb-6">
        <AnimatePresence mode="wait">
          {tab === "add" ? (
            <motion.div key="add" initial={{ opacity: 0, x: -10 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: 10 }}
              transition={{ duration: 0.18 }} className="flex flex-col gap-1">
              <p className="uppercase mb-2" style={{ fontSize: 10, letterSpacing: "0.14em", color: "var(--neu-text-muted)" }}>
                附近推荐
              </p>
              {filteredSuggested.map(u => (
                <motion.div key={u.id} whileTap={hover.tap}
                  className="flex items-center gap-3.5 px-3 py-3 rounded-[16px] neu-row-hover"
                  style={{ border: "1px solid var(--neu-border)" }}>
                  <img src={u.avatar} alt={u.name}
                    className="rounded-full object-cover flex-shrink-0"
                    style={{ width: 44, height: 44, border: "1.5px solid var(--neu-border)" }} />
                  <div className="flex-1 min-w-0">
                    <p style={{ fontSize: 14, color: "var(--neu-text)" }}>{u.name}</p>
                    <p style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>{u.subtitle}</p>
                  </div>
                  <motion.button whileTap={{ scale: 0.94 }}
                    onClick={() => setAddedIds(p => { const n = new Set(p); n.has(u.id) ? n.delete(u.id) : n.add(u.id); return n; })}
                    className="flex items-center gap-1.5 px-3 py-1.5 rounded-full outline-none flex-shrink-0"
                    style={addedIds.has(u.id)
                      ? { boxShadow: "inset 2px 2px 5px var(--neu-shadow-dark), inset -2px -2px 5px var(--neu-shadow-light)", border: "1px solid var(--neu-border-pressed)", color: "var(--neu-accent-primary)", background: "var(--neu-bg)", fontSize: 11 }
                      : { background: "var(--neu-accent-primary)", color: "var(--text-on-accent, white)", fontSize: 11, boxShadow: "0 3px 8px var(--neu-glow)" }
                    }>
                    {addedIds.has(u.id)
                      ? <><Check style={{ width: 10, height: 10 }} strokeWidth={2.5} /> 已添加</>
                      : <><UserPlus style={{ width: 10, height: 10 }} strokeWidth={2} /> 添加</>
                    }
                  </motion.button>
                </motion.div>
              ))}
            </motion.div>
          ) : (
            <motion.div key="chat" initial={{ opacity: 0, x: 10 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: -10 }}
              transition={{ duration: 0.18 }} className="flex flex-col gap-1">

              {/* Create group chat row */}
              <motion.button whileTap={hover.tap} whileHover={hover.row}
                onClick={() => { onClose(); onCreateGroup?.(); }}
                className="flex items-center gap-3.5 px-3 py-3 rounded-[16px] w-full outline-none text-left mb-3 neu-row-hover"
                style={{ background: "var(--neu-bg)", boxShadow: "5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light)", border: "1px solid var(--neu-border)" }}>
                <div className="flex items-center justify-center rounded-full flex-shrink-0"
                  style={{ width: 44, height: 44, background: "var(--neu-accent-primary)", boxShadow: "0 3px 8px var(--neu-glow)" }}>
                  <Users style={{ width: 20, height: 20, color: "var(--text-on-accent, white)" }} strokeWidth={1.75} />
                </div>
                <div>
                  <p style={{ fontSize: 14, color: "var(--neu-text)", fontWeight: 500 }}>创建群聊</p>
                  <p style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>邀请好友一起聊</p>
                </div>
              </motion.button>

              <p className="uppercase mb-2" style={{ fontSize: 10, letterSpacing: "0.14em", color: "var(--neu-text-muted)" }}>
                联系人
              </p>
              {filteredContacts.map(c => (
                <motion.div key={c.id} whileTap={hover.tap}
                  onClick={() => { onNewChat?.(c.id); onClose(); }}
                  className="flex items-center gap-3.5 px-3 py-3 rounded-[16px] cursor-pointer neu-row-hover"
                  style={{ border: "1px solid var(--neu-border)" }}>
                  <div className="relative flex-shrink-0">
                    <img src={c.avatar} alt={c.name}
                      className="rounded-full object-cover"
                      style={{ width: 44, height: 44, border: "1.5px solid var(--neu-border)" }} />
                    {c.online && (
                      <span className="absolute bottom-0 right-0 rounded-full"
                        style={{ width: 10, height: 10, background: "#22c55e", border: "2px solid var(--neu-bg)" }} />
                    )}
                  </div>
                  <div className="flex-1 min-w-0">
                    <p style={{ fontSize: 14, color: "var(--neu-text)" }}>{c.name}</p>
                    <p style={{ fontSize: 11, color: c.online ? "#22c55e" : "var(--neu-text-muted)" }}>
                      {c.online ? "在线" : "离线"}
                    </p>
                  </div>
                  <MessageCircle style={{ width: 16, height: 16, color: "var(--neu-text-muted)", flexShrink: 0 }} strokeWidth={1.5} />
                </motion.div>
              ))}
            </motion.div>
          )}
        </AnimatePresence>
      </div>
    </div>
  );
}
