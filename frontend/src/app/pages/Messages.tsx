import React, { useState, useEffect } from "react";
import { useNavigate } from "react-router";
import { Search, Plus, MessageSquareDashed, X } from "lucide-react";
import { motion, AnimatePresence } from "motion/react";
import { usePlatform } from "../contexts/PlatformContext";
import { ChatView } from "../components/ui/ChatView";
import { useHoverConfig } from "../hooks/useHoverConfig";
import { NewConversationSheet } from "../components/ui/NewConversationSheet";
import { CreateGroupSheet } from "../components/ui/CreateGroupSheet";
import { chatsService, type Conversation } from "@/lib/api/services";

/* ── Types ────────────────────────────────────────────────────────── */
interface ChatItem {
  id: number;
  name: string;
  avatar: string;
  lastMessage: string;
  time: string;
  unread: number;
  online: boolean;
  group: boolean;
}

function formatTime(iso: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (isNaN(d.getTime())) return iso;
  const now = new Date();
  const diffMs = now.getTime() - d.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  if (diffMins < 1) return "刚刚";
  if (diffMins < 60) return `${diffMins}分钟前`;
  const diffHours = Math.floor(diffMins / 60);
  if (diffHours < 24) return `${diffHours}小时前`;
  const diffDays = Math.floor(diffHours / 24);
  if (diffDays === 1) return "昨天";
  if (diffDays < 7) return `${diffDays}天前`;
  const h = d.getHours().toString().padStart(2, "0");
  const m = d.getMinutes().toString().padStart(2, "0");
  return `${d.getMonth() + 1}/${d.getDate()} ${h}:${m}`;
}

function toChatItem(c: Conversation): ChatItem {
  return {
    id: c.id,
    name: c.target_name,
    avatar: c.target_avatar || "",
    lastMessage: c.last_message || "",
    time: formatTime(c.last_message_at),
    unread: c.unread_count || 0,
    online: false,
    group: c.type === "group",
  };
}

/* ── Shared ChatListItem ─────────────────────────────────────────── */
function ChatListItem({
  chat,
  isSelected = false,
  onClick,
  compact = false,
}: {
  chat: ChatItem;
  isSelected?: boolean;
  onClick: () => void;
  compact?: boolean;
}) {
  const hover = useHoverConfig();
  return (
    <motion.div
      whileTap={hover.tap as any}
      whileHover={!isSelected ? hover.row as any : {}}
      onClick={onClick}
      className="flex items-center gap-3.5 cursor-pointer neu-row-hover"
      style={{
        padding: compact ? "10px 16px" : "12px 16px",
        borderRadius: 18,
        background: isSelected ? "var(--neu-accent-primary)" : "transparent",
        boxShadow: isSelected ? "0 4px 16px var(--neu-glow)" : "none",
        transition: "background 0.16s ease",
      }}
    >
      {/* Avatar */}
      <div className="relative flex-shrink-0">
        <div className="rounded-full overflow-hidden"
          style={{ width: compact ? 40 : 44, height: compact ? 40 : 44,
            border: isSelected ? "1.5px solid rgba(255,255,255,0.35)" : "1.5px solid var(--neu-border)" }}>
          <img src={chat.avatar} alt={chat.name} className="w-full h-full object-cover" />
        </div>
        {chat.online && (
          <span className="absolute bottom-0 right-0 rounded-full"
            style={{ width: 10, height: 10, background: "#22c55e", border: "2px solid var(--neu-bg)" }} />
        )}
      </div>

      {/* Text */}
      <div className="flex-1 min-w-0">
        <div className="flex justify-between items-baseline mb-0.5">
          <span style={{ fontSize: 14, color: isSelected ? "white" : "var(--neu-text)", fontWeight: chat.unread > 0 ? 600 : 400 }}>
            {chat.name}
          </span>
          <span style={{ fontSize: 11, flexShrink: 0, color: isSelected ? "rgba(255,255,255,0.7)" : "var(--neu-text-muted)" }}>
            {chat.time}
          </span>
        </div>
        <p style={{ fontSize: 12, color: isSelected ? "rgba(255,255,255,0.8)" : "var(--neu-text-muted)" }}
          className="truncate pr-4">
          {chat.lastMessage}
        </p>
      </div>

      {/* Unread badge */}
      {chat.unread > 0 && !isSelected && (
        <div className="flex-shrink-0 flex items-center justify-center rounded-full"
          style={{ width: 20, height: 20, background: "var(--neu-accent-primary)", color: "white", fontSize: 11, fontWeight: 600, boxShadow: "0 2px 8px var(--neu-glow)" }}>
          {chat.unread}
        </div>
      )}
    </motion.div>
  );
}

/* ══════════════════════════════════════════════════════════════════
   Mobile Messages
══════════════════════════════════════════════════════════════════ */
function MessagesMobile({ chats, loading }: { chats: ChatItem[]; loading: boolean }) {
  const navigate = useNavigate();
  const [searchVal, setSearchVal] = useState("");
  const filtered = chats.filter(c => c.name.toLowerCase().includes(searchVal.toLowerCase()));
  const onlineChats = chats.filter(c => c.online);
  const totalUnread = chats.reduce((s, c) => s + c.unread, 0);
  const hover = useHoverConfig();
  const [showNewChat, setShowNewChat] = useState(false);
  const [newChatTab, setNewChatTab] = useState<"add" | "chat">("chat");
  const [showCreateGroup, setShowCreateGroup] = useState(false);

  return (
    <>
      <div className="sticky top-0 z-10 px-5 pt-3 pb-3"
        style={{ backgroundColor: "var(--neu-bg)", borderBottom: "1px solid var(--neu-border)" }}>
        <div className="flex items-center justify-between mb-3">
          <div>
            <h1 style={{ fontSize: 22, color: "var(--neu-text)" }}>消息</h1>
            <p style={{ fontSize: 12, color: "var(--neu-text-muted)", marginTop: 2 }}>
              {totalUnread > 0 ? `${totalUnread} 条未读` : "全部已读"}
            </p>
          </div>
          <motion.button
            whileTap={hover.tap as any} whileHover={hover.btn as any}
            onClick={() => { setNewChatTab("chat"); setShowNewChat(true); }}
            className="flex items-center justify-center rounded-full outline-none neu-surface-flat"
            style={{ width: 36, height: 36 }}>
            <Plus style={{ width: 16, height: 16, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
          </motion.button>
        </div>
        <div className="flex items-center gap-3 px-4 py-2.5 rounded-[18px] neu-input-surface">
          <Search style={{ width: 15, height: 15, color: "var(--neu-text-muted)", flexShrink: 0 }} strokeWidth={1.5} />
          <input value={searchVal} onChange={e => setSearchVal(e.target.value)}
            placeholder="搜索会话…" className="flex-1 bg-transparent outline-none"
            style={{ fontSize: 13, color: "var(--neu-text)" }} />
        </div>
      </div>

      <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, ease: [0.22, 1, 0.36, 1] }}
        className="flex flex-col px-5 pt-4 pb-4"
        style={{ maxWidth: 640, marginLeft: "auto", marginRight: "auto", width: "100%" }}>
        {onlineChats.length > 0 && (
          <div className="mb-4">
            <p className="uppercase mb-2.5" style={{ fontSize: 10, letterSpacing: "0.14em", color: "var(--neu-text-muted)" }}>在线</p>
            <div className="flex gap-4 overflow-x-auto no-scrollbar pb-1">
              {onlineChats.map(chat => (
                <motion.div key={chat.id} whileTap={{ scale: 0.93 }}
                  onClick={() => navigate(`/messages/${chat.id}`)}
                  className="flex flex-col items-center gap-1.5 flex-shrink-0 cursor-pointer">
                  <div className="relative">
                    <div className="rounded-full overflow-hidden"
                      style={{ width: 52, height: 52, padding: 2, background: "var(--neu-bg)", border: "2px solid var(--neu-accent-primary)" }}>
                      <img src={chat.avatar} alt={chat.name} className="w-full h-full object-cover rounded-full" />
                    </div>
                    <span className="absolute bottom-0.5 right-0.5 rounded-full"
                      style={{ width: 12, height: 12, background: "#22c55e", border: "2px solid var(--neu-bg)" }} />
                  </div>
                  <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>{chat.name}</span>
                </motion.div>
              ))}
            </div>
          </div>
        )}

        <p className="uppercase mb-2.5" style={{ fontSize: 10, letterSpacing: "0.14em", color: "var(--neu-text-muted)" }}>最近</p>

        <div className="rounded-[18px] overflow-hidden neu-surface-flat">
          {loading
            ? Array.from({ length: 4 }).map((_, i) => (
                <div key={i}>
                  <div className="flex items-center gap-3.5" style={{ padding: "12px 16px" }}>
                    <div className="rounded-full flex-shrink-0" style={{ width: 44, height: 44, background: "var(--neu-border)", opacity: 0.6 }} />
                    <div className="flex-1 min-w-0">
                      <div style={{ width: "40%", height: 14, borderRadius: 8, background: "var(--neu-border)", opacity: 0.6, marginBottom: 6 }} />
                      <div style={{ width: "65%", height: 11, borderRadius: 6, background: "var(--neu-border)", opacity: 0.4 }} />
                    </div>
                  </div>
                  {i < 3 && <div className="mx-4" style={{ height: 1, background: "var(--neu-border)" }} />}
                </div>
              ))
            : filtered.length === 0
              ? (<div className="flex flex-col items-center justify-center py-12 gap-3">
                  <MessageSquareDashed style={{ width: 36, height: 36, color: "var(--neu-text-muted)", opacity: 0.5 }} strokeWidth={1.25} />
                  <p style={{ fontSize: 14, color: "var(--neu-text-muted)" }}>没有找到会话</p>
                </div>)
              : filtered.map((chat, i) => (
                  <React.Fragment key={chat.id}>
                    <div className="px-2 py-1">
                      <ChatListItem chat={chat} onClick={() => navigate(`/messages/${chat.id}`)} />
                    </div>
                    {i < filtered.length - 1 && (
                      <div className="mx-4" style={{ height: 1, background: "var(--neu-border)" }} />
                    )}
                  </React.Fragment>
                ))}
        </div>
      </motion.div>

      {/* New conversation / add friend bottom sheet */}
      <AnimatePresence>
        {showNewChat && (
          <>
            <motion.div key="backdrop" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
              transition={{ duration: 0.2 }}
              className="fixed inset-0 z-40"
              style={{ background: "rgba(0,0,0,0.35)", backdropFilter: "blur(3px)" }}
              onClick={() => setShowNewChat(false)} />
            <motion.div key="sheet"
              initial={{ y: "100%" }} animate={{ y: 0 }} exit={{ y: "100%" }}
              transition={{ type: "spring", damping: 30, stiffness: 260, mass: 0.9 }}
              className="fixed bottom-0 left-0 right-0 z-50 overflow-hidden"
              style={{ height: "85%", borderRadius: "28px 28px 0 0", backgroundColor: "var(--neu-bg)", boxShadow: "0 -8px 40px rgba(0,0,0,0.18)" }}>
              <NewConversationSheet
                defaultTab={newChatTab}
                onClose={() => setShowNewChat(false)}
                onNewChat={id => navigate(`/messages/${id}`)}
                onCreateGroup={() => setShowCreateGroup(true)}
              />
            </motion.div>
          </>
        )}
      </AnimatePresence>

      {/* Create group sheet */}
      <AnimatePresence>
        {showCreateGroup && (
          <>
            <motion.div key="backdrop2" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
              transition={{ duration: 0.2 }}
              className="fixed inset-0 z-50"
              style={{ background: "rgba(0,0,0,0.35)", backdropFilter: "blur(3px)" }}
              onClick={() => setShowCreateGroup(false)} />
            <motion.div key="sheet2"
              initial={{ y: "100%" }} animate={{ y: 0 }} exit={{ y: "100%" }}
              transition={{ type: "spring", damping: 30, stiffness: 260, mass: 0.9 }}
              className="fixed bottom-0 left-0 right-0 z-[60] overflow-hidden"
              style={{ height: "92%", borderRadius: "28px 28px 0 0", backgroundColor: "var(--neu-bg)", boxShadow: "0 -8px 40px rgba(0,0,0,0.18)" }}>
              <CreateGroupSheet onClose={() => setShowCreateGroup(false)} />
            </motion.div>
          </>
        )}
      </AnimatePresence>
    </>
  );
}

/* ══════════════════════════════════════════════════════════════════
   Desktop Messages — split pane (list left + chat right)
══════════════════════════════════════════════════════════════════ */
function MessagesDesktop({ chats, loading }: { chats: ChatItem[]; loading: boolean }) {
  const [searchVal, setSearchVal] = useState("");
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const filtered = chats.filter(c => c.name.toLowerCase().includes(searchVal.toLowerCase()));
  const totalUnread = chats.reduce((s, c) => s + c.unread, 0);
  const onlineChats = chats.filter(c => c.online);
  const hover = useHoverConfig();
  const [showNewChat, setShowNewChat] = useState(false);
  const [newChatTab, setNewChatTab] = useState<"add" | "chat">("chat");
  const [showCreateGroup, setShowCreateGroup] = useState(false);

  return (
    <div className="flex h-full">
      {/* Left: chat list */}
      <div className="flex flex-col flex-shrink-0 overflow-hidden"
        style={{ width: 300, borderRight: "1px solid var(--neu-border)" }}>
        {/* List header */}
        <div className="flex-shrink-0 px-4 pt-4 pb-3"
          style={{ borderBottom: "1px solid var(--neu-border)" }}>
          <div className="flex items-center justify-between mb-3">
            <h1 style={{ fontSize: 18, color: "var(--neu-text)" }}>消息</h1>
            <motion.button whileTap={hover.tap as any} whileHover={hover.btn as any}
              onClick={() => { setNewChatTab("chat"); setShowNewChat(true); }}
              className="flex items-center justify-center rounded-full outline-none neu-surface-flat"
              style={{ width: 30, height: 30 }}>
              <Plus style={{ width: 14, height: 14, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
            </motion.button>
          </div>
          <div className="flex items-center gap-2 px-3 py-2 rounded-[14px] neu-input-surface">
            <Search style={{ width: 14, height: 14, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
            <input value={searchVal} onChange={e => setSearchVal(e.target.value)}
              placeholder="搜索…" className="flex-1 bg-transparent outline-none"
              style={{ fontSize: 12, color: "var(--neu-text)" }} />
          </div>
        </div>

        {/* Online strip */}
        {onlineChats.length > 0 && (
          <div className="flex-shrink-0 px-4 py-2.5" style={{ borderBottom: "1px solid var(--neu-border)" }}>
            <p className="uppercase mb-2" style={{ fontSize: 9, letterSpacing: "0.12em", color: "var(--neu-text-muted)" }}>在线</p>
            <div className="flex gap-3 overflow-x-auto no-scrollbar">
              {onlineChats.map(c => (
                <motion.div key={c.id} whileTap={{ scale: 0.93 }}
                  onClick={() => setSelectedId(c.id)}
                  className="flex flex-col items-center gap-1 flex-shrink-0 cursor-pointer">
                  <div className="relative">
                    <div className="rounded-full overflow-hidden"
                      style={{ width: 36, height: 36, border: selectedId === c.id ? "2px solid var(--neu-accent-primary)" : "1.5px solid var(--neu-border)" }}>
                      <img src={c.avatar} alt={c.name} className="w-full h-full object-cover" />
                    </div>
                    <span className="absolute bottom-0 right-0 rounded-full"
                      style={{ width: 8, height: 8, background: "#22c55e", border: "1.5px solid var(--neu-bg)" }} />
                  </div>
                  <span style={{ fontSize: 10, color: "var(--neu-text-muted)" }}>{c.name}</span>
                </motion.div>
              ))}
            </div>
          </div>
        )}

        {/* Chat list */}
        <div className="flex-1 overflow-y-auto no-scrollbar px-2 py-2">
          {loading ? (
            Array.from({ length: 5 }).map((_, i) => (
              <div key={i} className="flex items-center gap-3" style={{ padding: "10px 16px" }}>
                <div className="rounded-full flex-shrink-0" style={{ width: 40, height: 40, background: "var(--neu-border)", opacity: 0.6 }} />
                <div className="flex-1 min-w-0">
                  <div style={{ width: "45%", height: 12, borderRadius: 6, background: "var(--neu-border)", opacity: 0.6, marginBottom: 5 }} />
                  <div style={{ width: "70%", height: 10, borderRadius: 5, background: "var(--neu-border)", opacity: 0.4 }} />
                </div>
              </div>
            ))
          ) : (
            <>
              {totalUnread > 0 && (
                <p className="uppercase px-2 mb-1" style={{ fontSize: 9, letterSpacing: "0.12em", color: "var(--neu-text-muted)" }}>未读</p>
              )}
              {filtered.map(chat => (
                <ChatListItem
                  key={chat.id}
                  chat={chat}
                  isSelected={selectedId === chat.id}
                  onClick={() => setSelectedId(chat.id)}
                  compact
                />
              ))}
            </>
          )}
        </div>
      </div>

      {/* Right: chat view or empty state */}
      <div className="flex-1 overflow-hidden">
        <AnimatePresence mode="wait">
          {selectedId !== null ? (
            <motion.div
              key={selectedId}
              initial={{ opacity: 0, x: 10 }}
              animate={{ opacity: 1, x: 0 }}
              exit={{ opacity: 0, x: -10 }}
              transition={{ duration: 0.18, ease: "easeOut" }}
              className="h-full"
            >
              <ChatView chatId={selectedId} showBackButton={false} compact />
            </motion.div>
          ) : (
            <motion.div
              key="empty"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              className="h-full flex flex-col items-center justify-center gap-4"
            >
              <div className="flex items-center justify-center rounded-full neu-surface-flat"
                style={{ width: 72, height: 72 }}>
                <MessageSquareDashed style={{ width: 28, height: 28, color: "var(--neu-text-muted)", opacity: 0.5 }} strokeWidth={1.25} />
              </div>
              <div className="text-center">
                <p style={{ fontSize: 16, color: "var(--neu-text)" }}>选择一个会话</p>
                <p style={{ fontSize: 13, color: "var(--neu-text-muted)", marginTop: 4 }}>从左侧选择联系人开始聊天</p>
              </div>
            </motion.div>
          )}
        </AnimatePresence>
      </div>

      {/* New conversation modal (desktop center dialog) */}
      <AnimatePresence>
        {showNewChat && (
          <>
            <motion.div key="backdrop" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
              transition={{ duration: 0.18 }}
              className="fixed inset-0 z-50"
              style={{ background: "rgba(0,0,0,0.42)", backdropFilter: "blur(5px)" }}
              onClick={() => setShowNewChat(false)} />
            <div className="fixed inset-0 z-[60] flex items-center justify-center pointer-events-none">
              <motion.div key="modal"
                initial={{ opacity: 0, scale: 0.93, y: 14 }} animate={{ opacity: 1, scale: 1, y: 0 }} exit={{ opacity: 0, scale: 0.93, y: 14 }}
                transition={{ type: "spring", damping: 26, stiffness: 300 }}
                className="overflow-hidden pointer-events-auto"
                style={{ width: 440, height: "68vh", maxHeight: 600, borderRadius: 24, background: "var(--neu-bg)", boxShadow: "0 24px 80px rgba(0,0,0,0.24), 0 0 0 1px var(--neu-border)" }}>
                <NewConversationSheet
                  defaultTab={newChatTab}
                  onClose={() => setShowNewChat(false)}
                  onNewChat={id => { setSelectedId(id); setShowNewChat(false); }}
                  onCreateGroup={() => { setShowNewChat(false); setShowCreateGroup(true); }}
                />
              </motion.div>
            </div>
          </>
        )}
      </AnimatePresence>

      {/* Create group modal */}
      <AnimatePresence>
        {showCreateGroup && (
          <>
            <motion.div key="backdrop2" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
              transition={{ duration: 0.18 }}
              className="fixed inset-0 z-50"
              style={{ background: "rgba(0,0,0,0.42)", backdropFilter: "blur(5px)" }}
              onClick={() => setShowCreateGroup(false)} />
            <div className="fixed inset-0 z-[60] flex items-center justify-center pointer-events-none">
              <motion.div key="modal2"
                initial={{ opacity: 0, scale: 0.93, y: 14 }} animate={{ opacity: 1, scale: 1, y: 0 }} exit={{ opacity: 0, scale: 0.93, y: 14 }}
                transition={{ type: "spring", damping: 26, stiffness: 300 }}
                className="overflow-hidden pointer-events-auto"
                style={{ width: 480, height: "78vh", maxHeight: 700, borderRadius: 24, background: "var(--neu-bg)", boxShadow: "0 24px 80px rgba(0,0,0,0.24), 0 0 0 1px var(--neu-border)" }}>
                <CreateGroupSheet onClose={() => setShowCreateGroup(false)} />
              </motion.div>
            </div>
          </>
        )}
      </AnimatePresence>
    </div>
  );
}

/* ══════════════════════════════════════════════════════════════════
   Entry — fetches chat list, passes to platform variant
══════════════════════════════════════════════════════════════════ */
export function Messages() {
  const { platform } = usePlatform();
  const [chats, setChats] = useState<ChatItem[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    chatsService.list().then((res) => {
      if (cancelled) return;
      if (res.code === 0 && Array.isArray(res.data)) {
        setChats(res.data.map(toChatItem));
      }
      setLoading(false);
    }).catch(() => {
      if (!cancelled) setLoading(false);
    });
    return () => { cancelled = true; };
  }, []);

  if (platform === "desktop") return <MessagesDesktop chats={chats} loading={loading} />;
  return <MessagesMobile chats={chats} loading={loading} />;
}