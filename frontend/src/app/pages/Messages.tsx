import React, { useState } from "react";
import { useNavigate } from "react-router";
import { ROUTES } from "../lib/routes";
import { Search, Plus, MessageSquareDashed, X } from "lucide-react";
import { motion, AnimatePresence } from "motion/react";
import { usePlatform } from "../contexts/PlatformContext";
import { ChatView } from "../features/messaging/ChatView";
import { useHoverConfig } from "../hooks/useHoverConfig";
import { useDismissibleLayer } from "../hooks/useDismissibleLayer";
import { NewConversationSheet } from "../features/messaging/NewConversationSheet";
import { CreateGroupSheet } from "../features/groups/CreateGroupSheet";

/* ── Data ────────────────────────────────────────────────────────── */
const CHATS = [
  { id: 1, name: "林一",      avatar: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400", lastMessage: "嘿，我们今晚的计划还在吗？", time: "13:27", unread: 2, online: true,  group: false },
  { id: 2, name: "设计小队",  avatar: "https://images.unsplash.com/photo-1763467669450-e69029c1484d?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400", lastMessage: "新原型看起来太棒了！",         time: "09:41", unread: 0, online: false, group: true  },
  { id: 3, name: "小满",  avatar: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400", lastMessage: "给你发了张照片 📷",            time: "昨天",  unread: 0, online: true,  group: false },
  { id: 4, name: "阿川",       avatar: "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400", lastMessage: "在老地方见！",                time: "周一",  unread: 0, online: false, group: false },
];

/* ── Shared ChatListItem ─────────────────────────────────────────── */
function ChatListItem({
  chat,
  isSelected = false,
  onClick,
  compact = false,
}: {
  chat: typeof CHATS[0];
  isSelected?: boolean;
  onClick: () => void;
  compact?: boolean;
}) {
  const hover = useHoverConfig();
  return (
    <motion.button
      type="button"
      data-active={isSelected}
      aria-current={isSelected ? "page" : undefined}
      whileTap={hover.tap as any}
      // A selected row keeps its anchored state. Its CSS active-hover treatment
      // adds contrast without applying the ordinary row lift a second time.
      whileHover={isSelected ? undefined : hover.row as any}
      onClick={onClick}
      className="app-list-row app-list-row--rich w-full cursor-pointer text-left outline-none neu-row-hover"
      style={{
        padding: compact ? "10px 16px" : "12px 16px",
        borderRadius: 18,
      }}
    >
      <span className="relative flex-shrink-0">
        <span
          className="block overflow-hidden rounded-full"
          style={{
            width: compact ? 40 : 44,
            height: compact ? 40 : 44,
            border: isSelected ? "1.5px solid var(--neu-accent-primary)" : "1.5px solid var(--neu-border)",
          }}
        >
          <img src={chat.avatar} alt="" className="h-full w-full object-cover" />
        </span>
        {chat.online && (
          <span
            className="absolute bottom-0 right-0 rounded-full"
            style={{ width: 10, height: 10, background: "var(--semantic-online)", border: "2px solid var(--neu-bg)" }}
          />
        )}
      </span>

      <span className="min-w-0 flex-1">
        <span className="mb-0.5 flex items-baseline justify-between">
          <span style={{ fontSize: 14, color: isSelected ? "var(--neu-accent-primary)" : "var(--neu-text)", fontWeight: chat.unread > 0 || isSelected ? 600 : 400 }}>
            {chat.name}
          </span>
          <span style={{ fontSize: 11, flexShrink: 0, color: "var(--neu-text-muted)" }}>
            {chat.time}
          </span>
        </span>
        <span className="block truncate pr-4" style={{ fontSize: 12, color: "var(--neu-text-muted)" }}>
          {chat.lastMessage}
        </span>
      </span>

      {chat.unread > 0 && !isSelected && (
        <span
          className="flex flex-shrink-0 items-center justify-center rounded-full"
          style={{ width: 20, height: 20, background: "var(--neu-accent-primary)", color: "white", fontSize: 11, fontWeight: 600, boxShadow: "0 2px 8px var(--neu-glow)" }}
        >
          {chat.unread}
        </span>
      )}
    </motion.button>
  );
}

/* ══════════════════════════════════════════════════════════════════
   Mobile Messages
══════════════════════════════════════════════════════════════════ */
function MessagesMobile() {
  const navigate = useNavigate();
  const [searchVal, setSearchVal] = useState("");
  const filtered = CHATS.filter(c => c.name.toLowerCase().includes(searchVal.toLowerCase()));
  const onlineChats = CHATS.filter(c => c.online);
  const totalUnread = CHATS.reduce((s, c) => s + c.unread, 0);
  const hover = useHoverConfig();
  const [showNewChat, setShowNewChat] = useState(false);
  const [newChatTab, setNewChatTab] = useState<"add" | "chat">("chat");
  const [showCreateGroup, setShowCreateGroup] = useState(false);
  useDismissibleLayer({ open: showNewChat, onDismiss: () => setShowNewChat(false) });
  useDismissibleLayer({ open: showCreateGroup, onDismiss: () => setShowCreateGroup(false) });

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
                  onClick={() => navigate(ROUTES.message(chat.id))}
                  className="flex flex-col items-center gap-1.5 flex-shrink-0 cursor-pointer">
                  <div className="relative">
                    <div className="rounded-full overflow-hidden"
                      style={{ width: 52, height: 52, padding: 2, background: "var(--neu-bg)", border: "2px solid var(--neu-accent-primary)" }}>
                      <img src={chat.avatar} alt={chat.name} className="w-full h-full object-cover rounded-full" />
                    </div>
                    <span className="absolute bottom-0.5 right-0.5 rounded-full"
                      style={{ width: 12, height: 12, background: "var(--semantic-online)", border: "2px solid var(--neu-bg)" }} />
                  </div>
                  <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>{chat.name}</span>
                </motion.div>
              ))}
            </div>
          </div>
        )}

        <p className="uppercase mb-2.5" style={{ fontSize: 10, letterSpacing: "0.14em", color: "var(--neu-text-muted)" }}>最近</p>

        <div className="rounded-[18px] overflow-hidden neu-surface-flat">
          {filtered.length === 0
            ? (<div className="flex flex-col items-center justify-center py-12 gap-3">
                <MessageSquareDashed style={{ width: 36, height: 36, color: "var(--neu-text-muted)", opacity: 0.5 }} strokeWidth={1.25} />
                <p style={{ fontSize: 14, color: "var(--neu-text-muted)" }}>没有找到会话</p>
              </div>)
            : filtered.map((chat, i) => (
                <React.Fragment key={chat.id}>
                  <div className="px-2 py-1">
                    <ChatListItem chat={chat} onClick={() => navigate(ROUTES.message(chat.id))} />
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
                onNewChat={id => navigate(ROUTES.message(id))}
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
function MessagesDesktop() {
  const [searchVal, setSearchVal] = useState("");
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const filtered = CHATS.filter(c => c.name.toLowerCase().includes(searchVal.toLowerCase()));
  const totalUnread = CHATS.reduce((s, c) => s + c.unread, 0);
  const onlineChats = CHATS.filter(c => c.online);
  const hover = useHoverConfig();
  const [showNewChat, setShowNewChat] = useState(false);
  const [newChatTab, setNewChatTab] = useState<"add" | "chat">("chat");
  const [showCreateGroup, setShowCreateGroup] = useState(false);
  useDismissibleLayer({ open: showNewChat, onDismiss: () => setShowNewChat(false) });
  useDismissibleLayer({ open: showCreateGroup, onDismiss: () => setShowCreateGroup(false) });

  return (
    <div className="content-mode-wide flex h-full">
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

        {/* 在线 strip */}
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
                      style={{ width: 8, height: 8, background: "var(--semantic-online)", border: "1.5px solid var(--neu-bg)" }} />
                  </div>
                  <span style={{ fontSize: 10, color: "var(--neu-text-muted)" }}>{c.name}</span>
                </motion.div>
              ))}
            </div>
          </div>
        )}

        {/* Chat list */}
        <div className="flex-1 overflow-y-auto no-scrollbar px-2 py-2">
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
        </div>
      </div>

      {/* Right: chat view or empty state */}
      <div className="flex min-w-0 flex-1 bg-[color-mix(in_srgb,var(--page-bg)_55%,transparent)] p-4">
        <div className="min-h-0 flex-1 overflow-hidden rounded-[22px] border border-[var(--neu-border)] bg-[var(--neu-bg)] shadow-[0_10px_28px_color-mix(in_srgb,var(--neu-shadow-dark)_32%,transparent)]">
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
                role="dialog"
                aria-modal="true"
                aria-label="发起新会话"
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
                role="dialog"
                aria-modal="true"
                aria-label="创建群组"
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
   Entry
══════════════════════════════════════════════════════════════════ */
export function Messages() {
  const { platform } = usePlatform();
  if (platform === "desktop") return <MessagesDesktop />;
  return <MessagesMobile />;
}