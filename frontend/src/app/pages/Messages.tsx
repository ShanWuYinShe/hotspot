import { useState, useMemo } from "react";
import { useNavigate } from "react-router";
import { Search, Plus, MessageSquareDashed, Users } from "lucide-react";
import { EmptyState } from "../components/ui/EmptyState";
import { motion, AnimatePresence } from "motion/react";
import { usePlatform } from "../contexts/PlatformContext";
import { ChatView } from "../components/ui/ChatView";
import { useHoverConfig } from "../hooks/useHoverConfig";
import { NewConversationSheet } from "../components/ui/NewConversationSheet";
import { CreateGroupSheet } from "../components/ui/CreateGroupSheet";
import { useFocusTrap } from "../hooks/useFocusTrap";
import { useMessagesData } from "@/lib/hooks/use-messages-data";
import { useDocumentTitle } from "@/lib/hooks/use-document-title";
import { normalizeAvatar, formatRelativeTime } from "@/lib/utils";
import type { PrivateChatItem, GroupItem } from "@/lib/types/api";

/* ── Private Chat List Item ─────────────────────────────────────── */
function PrivateChatListItem({
  chat,
  isSelected = false,
  onClick,
  compact = false,
}: {
  chat: PrivateChatItem;
  isSelected?: boolean;
  onClick: () => void;
  compact?: boolean;
}) {
  const hover = useHoverConfig();
  const user = chat.other_user;
  const isOnline = chat.presence_status === "online";

  return (
    <motion.div
      whileTap={hover.tap}
      whileHover={!isSelected ? hover.row : {}}
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
      <div className="relative flex-shrink-0">
        <div
          className="rounded-full overflow-hidden"
          style={{
            width: compact ? 40 : 44,
            height: compact ? 40 : 44,
            border: isSelected
              ? "1.5px solid rgba(255,255,255,0.35)"
              : "1.5px solid var(--neu-border)",
          }}
        >
          <img
            src={normalizeAvatar(user.avatar_url)}
            alt={user.nickname}
            className="w-full h-full object-cover"
          />
        </div>
        {isOnline && (
          <span
            className="absolute bottom-0 right-0 rounded-full"
            style={{
              width: 10,
              height: 10,
              background: "var(--neu-success)",
              border: "2px solid var(--neu-bg)",
            }}
          />
        )}
      </div>

      <div className="flex-1 min-w-0">
        <div className="flex justify-between items-baseline mb-0.5">
          <span
            className="truncate"
            style={{
              fontSize: 14,
              color: isSelected
                ? "var(--text-on-accent, white)"
                : "var(--neu-text)",
              fontWeight: chat.unread_count > 0 ? 600 : 400,
            }}
          >
            {user.nickname}
          </span>
          <span
            style={{
              fontSize: 11,
              flexShrink: 0,
              color: isSelected
                ? "rgba(255,255,255,0.7)"
                : "var(--neu-text-muted)",
            }}
          >
            {formatRelativeTime(chat.last_activity_at)}
          </span>
        </div>
        <p
          style={{
            fontSize: 12,
            color: isSelected
              ? "rgba(255,255,255,0.8)"
              : "var(--neu-text-muted)",
          }}
          className="truncate pr-4"
        >
          {chat.last_message
            ? chat.last_message.content || "[媒体消息]"
            : "暂无消息"}
        </p>
      </div>

      {chat.unread_count > 0 && !isSelected && (
        <div
          className="flex-shrink-0 flex items-center justify-center rounded-full"
          style={{
            width: 20,
            height: 20,
            background: "var(--neu-accent-primary)",
            color: "var(--text-on-accent, white)",
            fontSize: 11,
            fontWeight: 600,
            boxShadow: "0 2px 8px var(--neu-glow)",
          }}
        >
          {chat.unread_count > 99 ? "99+" : chat.unread_count}
        </div>
      )}
    </motion.div>
  );
}

/* ── Group Chat List Item ────────────────────────────────────────── */
function GroupChatListItem({
  group,
  isSelected = false,
  onClick,
}: {
  group: GroupItem;
  isSelected?: boolean;
  onClick: () => void;
}) {
  const hover = useHoverConfig();
  const g = group.group;
  const unreadCount = group.my_state?.unread_count ?? 0;

  return (
    <motion.div
      whileTap={hover.tap}
      whileHover={!isSelected ? hover.row : {}}
      onClick={onClick}
      className="flex items-center gap-3.5 cursor-pointer neu-row-hover"
      style={{
        padding: "10px 16px",
        borderRadius: 18,
        background: isSelected ? "var(--neu-accent-primary)" : "transparent",
        boxShadow: isSelected ? "0 4px 16px var(--neu-glow)" : "none",
        transition: "background 0.16s ease",
      }}
    >
      <div className="relative flex-shrink-0">
        <div
          className="rounded-full overflow-hidden flex items-center justify-center"
          style={{
            width: 40,
            height: 40,
            border: isSelected
              ? "1.5px solid rgba(255,255,255,0.35)"
              : "1.5px solid var(--neu-border)",
            background: g.avatar_url ? "transparent" : "var(--neu-bg)",
          }}
        >
          {g.avatar_url ? (
            <img
              src={normalizeAvatar(g.avatar_url)}
              alt={g.name}
              className="w-full h-full object-cover"
            />
          ) : (
            <Users
              style={{ width: 16, height: 16, color: "var(--neu-text-muted)" }}
              strokeWidth={1.5}
            />
          )}
        </div>
      </div>

      <div className="flex-1 min-w-0">
        <div className="flex justify-between items-baseline mb-0.5">
          <span
            style={{
              fontSize: 14,
              fontWeight: 500,
              color: isSelected
                ? "var(--text-on-accent, white)"
                : "var(--neu-text)",
            }}
          >
            {g.name}
          </span>
          <span
            style={{
              fontSize: 11,
              flexShrink: 0,
              color: isSelected
                ? "rgba(255,255,255,0.7)"
                : "var(--neu-text-muted)",
            }}
          >
            {g.members_count}人
          </span>
        </div>
        <p
          style={{
            fontSize: 12,
            color: isSelected
              ? "rgba(255,255,255,0.8)"
              : "var(--neu-text-muted)",
          }}
          className="truncate pr-4"
        >
          {formatRelativeTime(
            group.last_activity_at ?? group.my_state?.joined_at ?? "",
          )}
        </p>
      </div>

      {unreadCount > 0 && !isSelected && (
        <div
          className="flex-shrink-0 flex items-center justify-center rounded-full"
          style={{
            width: 20,
            height: 20,
            background: "var(--neu-accent-primary)",
            color: "var(--text-on-accent, white)",
            fontSize: 11,
            fontWeight: 600,
            boxShadow: "0 2px 8px var(--neu-glow)",
          }}
        >
          {unreadCount > 99 ? "99+" : unreadCount}
        </div>
      )}
    </motion.div>
  );
}

/* ── Loading Skeleton ────────────────────────────────────────────── */
function ChatSkeleton() {
  return (
    <>
      {Array.from({ length: 5 }).map((_, i) => (
        <div key={i} className="flex items-center gap-3 px-3 py-2.5">
          <div
            className="rounded-full"
            style={{ width: 40, height: 40, background: "var(--neu-border)" }}
          />
          <div className="flex-1">
            <div
              className="mb-1 rounded"
              style={{ width: 60, height: 11, background: "var(--neu-border)" }}
            />
            <div
              className="rounded"
              style={{ width: 100, height: 9, background: "var(--neu-border)" }}
            />
          </div>
        </div>
      ))}
    </>
  );
}

/* ══════════════════════════════════════════════════════════════════
   Mobile Messages
══════════════════════════════════════════════════════════════════ */
function MessagesMobile() {
  const navigate = useNavigate();
  const {
    privateChats,
    groupChats,
    loadingPrivate,
    loadingGroup,
    error,
    totalUnread,
    refreshPrivate,
    loadGroups,
  } = useMessagesData();

  const [searchVal, setSearchVal] = useState("");
  const [currentTab, setCurrentTab] = useState<"private" | "group">("private");
  const [showNewChat, setShowNewChat] = useState(false);
  const [newChatTab, setNewChatTab] = useState<"add" | "chat">("chat");
  const [showCreateGroup, setShowCreateGroup] = useState(false);
  const hover = useHoverConfig();
  const newChatTrapRef = useFocusTrap(showNewChat, () => setShowNewChat(false));
  const groupTrapRef = useFocusTrap(showCreateGroup, () =>
    setShowCreateGroup(false),
  );

  const filteredPrivate = useMemo(
    () =>
      searchVal.trim()
        ? privateChats.filter((c) =>
            c.other_user.nickname
              .toLowerCase()
              .includes(searchVal.toLowerCase()),
          )
        : privateChats,
    [searchVal, privateChats],
  );

  const filteredGroups = useMemo(
    () =>
      searchVal.trim()
        ? groupChats.filter((g) =>
            g.group.name.toLowerCase().includes(searchVal.toLowerCase()),
          )
        : groupChats,
    [searchVal, groupChats],
  );

  const onlineChats = useMemo(
    () => privateChats.filter((c) => c.presence_status === "online"),
    [privateChats],
  );

  function handleTabChange(tab: "private" | "group") {
    setCurrentTab(tab);
    if (tab === "group") loadGroups();
  }

  return (
    <>
      <div
        className="sticky top-0 z-10 px-5 pt-3 pb-3"
        style={{
          backgroundColor: "var(--neu-bg)",
          borderBottom: "1px solid var(--neu-border)",
        }}
      >
        <div className="flex items-center justify-between mb-3">
          <div>
            <h1 style={{ fontSize: 22, color: "var(--neu-text)" }}>消息</h1>
            <p
              style={{
                fontSize: 12,
                color: "var(--neu-text-muted)",
                marginTop: 2,
              }}
            >
              {totalUnread > 0 ? `${totalUnread} 条未读` : "全部已读"}
            </p>
          </div>
          <motion.button
            whileTap={hover.tap}
            whileHover={hover.btn}
            onClick={() => {
              setNewChatTab("chat");
              setShowNewChat(true);
            }}
            aria-label="新建会话"
            className="flex items-center justify-center rounded-full outline-none neu-surface-flat"
            style={{ width: 36, height: 36 }}
          >
            <Plus
              style={{ width: 16, height: 16, color: "var(--neu-text-muted)" }}
              strokeWidth={1.5}
            />
          </motion.button>
        </div>
        <div className="flex items-center gap-3 px-4 py-2.5 rounded-[18px] neu-input-surface">
          <Search
            style={{
              width: 15,
              height: 15,
              color: "var(--neu-text-muted)",
              flexShrink: 0,
            }}
            strokeWidth={1.5}
          />
          <input
            value={searchVal}
            onChange={(e) => setSearchVal(e.target.value)}
            placeholder="搜索会话…"
            className="flex-1 bg-transparent outline-none"
            style={{ fontSize: 13, color: "var(--neu-text)" }}
          />
        </div>
      </div>

      <motion.div
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, ease: [0.22, 1, 0.36, 1] }}
        className="flex flex-col px-5 pt-4 pb-4"
        style={{
          maxWidth: 640,
          marginLeft: "auto",
          marginRight: "auto",
          width: "100%",
        }}
      >
        {/* Online strip */}
        {onlineChats.length > 0 && (
          <div className="mb-4">
            <p
              className="uppercase mb-2.5"
              style={{
                fontSize: 10,
                letterSpacing: "0.14em",
                color: "var(--neu-text-muted)",
              }}
            >
              在线
            </p>
            <div className="flex gap-4 overflow-x-auto no-scrollbar pb-1">
              {onlineChats.map((chat) => (
                <motion.div
                  key={chat.other_user.user_id}
                  whileTap={{ scale: 0.93 }}
                  onClick={() =>
                    navigate(`/messages/${chat.other_user.user_id}`)
                  }
                  className="flex flex-col items-center gap-1.5 flex-shrink-0 cursor-pointer"
                >
                  <div className="relative">
                    <div
                      className="rounded-full overflow-hidden"
                      style={{
                        width: 52,
                        height: 52,
                        padding: 2,
                        background: "var(--neu-bg)",
                        border: "2px solid var(--neu-accent-primary)",
                      }}
                    >
                      <img
                        src={normalizeAvatar(chat.other_user.avatar_url)}
                        alt={chat.other_user.nickname}
                        className="w-full h-full object-cover rounded-full"
                      />
                    </div>
                    <span
                      className="absolute bottom-0.5 right-0.5 rounded-full"
                      style={{
                        width: 12,
                        height: 12,
                        background: "var(--neu-success)",
                        border: "2px solid var(--neu-bg)",
                      }}
                    />
                  </div>
                  <span
                    style={{ fontSize: 11, color: "var(--neu-text-muted)" }}
                  >
                    {chat.other_user.nickname}
                  </span>
                </motion.div>
              ))}
            </div>
          </div>
        )}

        {/* Tab pills */}
        <div className="flex items-center gap-2 mb-3">
          {(["private", "group"] as const).map((tab) => (
            <motion.button
              whileTap={{ scale: 0.94 }}
              key={tab}
              className="rounded-full px-2 py-0.5 outline-none transition-all"
              style={
                currentTab === tab
                  ? {
                      background: "var(--neu-accent-primary)",
                      color: "var(--text-on-accent, white)",
                      fontSize: 11,
                      boxShadow: "0 2px 8px var(--neu-glow)",
                    }
                  : {
                      background: "var(--neu-bg)",
                      color: "var(--neu-text-muted)",
                      fontSize: 11,
                      border: "1px solid var(--neu-border)",
                    }
              }
              onClick={() => handleTabChange(tab)}
            >
              {tab === "private" ? "私聊" : "群聊"}
            </motion.button>
          ))}
        </div>

        {/* Chat list */}
        <div className="rounded-[18px] overflow-hidden neu-surface-flat">
          {currentTab === "private" ? (
            loadingPrivate ? (
              <ChatSkeleton />
            ) : error && filteredPrivate.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-12 gap-3">
                <p style={{ fontSize: 14, color: "var(--neu-text-muted)" }}>
                  {error}
                </p>
                <motion.button
                  whileTap={{ scale: 0.94 }}
                  onClick={refreshPrivate}
                  className="px-4 py-2 rounded-full outline-none"
                  style={{
                    background: "var(--neu-accent-primary)",
                    color: "var(--text-on-accent, white)",
                    fontSize: 13,
                  }}
                >
                  重试
                </motion.button>
              </div>
            ) : filteredPrivate.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-12 gap-3">
                <MessageSquareDashed
                  style={{
                    width: 36,
                    height: 36,
                    color: "var(--neu-text-muted)",
                    opacity: 0.5,
                  }}
                  strokeWidth={1.25}
                />
                <p style={{ fontSize: 14, color: "var(--neu-text-muted)" }}>
                  没有找到会话
                </p>
              </div>
            ) : (
              filteredPrivate.map((chat) => (
                <div key={chat.other_user.user_id} className="px-2 py-1">
                  <PrivateChatListItem
                    chat={chat}
                    onClick={() =>
                      navigate(`/messages/${chat.other_user.user_id}`)
                    }
                  />
                </div>
              ))
            )
          ) : loadingGroup ? (
            <ChatSkeleton />
          ) : filteredGroups.length === 0 ? (
            <EmptyState icon={Users} title="暂无群聊" />
          ) : (
            filteredGroups.map((group) => (
              <div key={group.group.group_id} className="px-2 py-1">
                <GroupChatListItem
                  group={group}
                  onClick={() =>
                    navigate(`/messages/group-${group.group.group_id}`)
                  }
                />
              </div>
            ))
          )}
        </div>
      </motion.div>

      {/* New conversation bottom sheet */}
      <AnimatePresence>
        {showNewChat && (
          <>
            <motion.div
              key="backdrop"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              transition={{ duration: 0.2 }}
              className="fixed inset-0 z-40"
              style={{
                background: "rgba(0,0,0,0.35)",
                backdropFilter: "blur(3px)",
              }}
              onClick={() => setShowNewChat(false)}
            />
            <motion.div
              key="sheet"
              ref={newChatTrapRef}
              role="dialog"
              aria-modal="true"
              aria-label="发起对话"
              initial={{ y: "100%" }}
              animate={{ y: 0 }}
              exit={{ y: "100%" }}
              transition={{
                type: "spring",
                damping: 30,
                stiffness: 260,
                mass: 0.9,
              }}
              className="fixed bottom-0 left-0 right-0 z-50 overflow-hidden"
              style={{
                height: "85%",
                borderRadius: "28px 28px 0 0",
                backgroundColor: "var(--neu-bg)",
                boxShadow: "0 -8px 40px rgba(0,0,0,0.18)",
              }}
            >
              <NewConversationSheet
                defaultTab={newChatTab}
                onClose={() => setShowNewChat(false)}
                onNewChat={(id) => navigate(`/messages/${id}`)}
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
            <motion.div
              key="backdrop2"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              transition={{ duration: 0.2 }}
              className="fixed inset-0 z-50"
              style={{
                background: "rgba(0,0,0,0.35)",
                backdropFilter: "blur(3px)",
              }}
              onClick={() => setShowCreateGroup(false)}
            />
            <motion.div
              key="sheet2"
              ref={groupTrapRef}
              role="dialog"
              aria-modal="true"
              aria-label="创建群组"
              initial={{ y: "100%" }}
              animate={{ y: 0 }}
              exit={{ y: "100%" }}
              transition={{
                type: "spring",
                damping: 30,
                stiffness: 260,
                mass: 0.9,
              }}
              className="fixed bottom-0 left-0 right-0 z-[60] overflow-hidden"
              style={{
                height: "92%",
                borderRadius: "28px 28px 0 0",
                backgroundColor: "var(--neu-bg)",
                boxShadow: "0 -8px 40px rgba(0,0,0,0.18)",
              }}
            >
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
  const {
    privateChats,
    groupChats,
    loadingPrivate,
    loadingGroup,
    totalUnread,
    loadGroups,
  } = useMessagesData();

  const [searchVal, setSearchVal] = useState("");
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [currentTab, setCurrentTab] = useState<"private" | "group">("private");
  const [showNewChat, setShowNewChat] = useState(false);
  const [newChatTab, setNewChatTab] = useState<"add" | "chat">("chat");
  const [showCreateGroup, setShowCreateGroup] = useState(false);
  const hover = useHoverConfig();
  const newChatTrapRef = useFocusTrap(showNewChat, () => setShowNewChat(false));
  const groupTrapRef = useFocusTrap(showCreateGroup, () =>
    setShowCreateGroup(false),
  );

  const filteredPrivate = useMemo(
    () =>
      searchVal.trim()
        ? privateChats.filter((c) =>
            c.other_user.nickname
              .toLowerCase()
              .includes(searchVal.toLowerCase()),
          )
        : privateChats,
    [searchVal, privateChats],
  );

  const filteredGroups = useMemo(
    () =>
      searchVal.trim()
        ? groupChats.filter((g) =>
            g.group.name.toLowerCase().includes(searchVal.toLowerCase()),
          )
        : groupChats,
    [searchVal, groupChats],
  );

  const onlineChats = useMemo(
    () => privateChats.filter((c) => c.presence_status === "online"),
    [privateChats],
  );

  function handleTabChange(tab: "private" | "group") {
    setCurrentTab(tab);
    if (tab === "group") loadGroups();
  }

  return (
    <div className="flex h-full">
      {/* Left: chat list */}
      <div
        className="flex flex-col flex-shrink-0 overflow-hidden"
        style={{ width: 300, borderRight: "1px solid var(--neu-border)" }}
      >
        {/* List header */}
        <div
          className="flex-shrink-0 px-4 pt-4 pb-3"
          style={{ borderBottom: "1px solid var(--neu-border)" }}
        >
          <div className="flex items-center justify-between mb-3">
            <h1 style={{ fontSize: 18, color: "var(--neu-text)" }}>消息</h1>
            <motion.button
              whileTap={hover.tap}
              whileHover={hover.btn}
              onClick={() => {
                setNewChatTab("chat");
                setShowNewChat(true);
              }}
              aria-label="新建会话"
              className="flex items-center justify-center rounded-full outline-none neu-surface-flat"
              style={{ width: 38, height: 38 }}
            >
              <Plus
                style={{
                  width: 14,
                  height: 14,
                  color: "var(--neu-text-muted)",
                }}
                strokeWidth={1.5}
              />
            </motion.button>
          </div>
          <div className="flex items-center gap-2 px-3 py-2 rounded-[14px] neu-input-surface">
            <Search
              style={{ width: 14, height: 14, color: "var(--neu-text-muted)" }}
              strokeWidth={1.5}
            />
            <input
              value={searchVal}
              onChange={(e) => setSearchVal(e.target.value)}
              placeholder="搜索…"
              className="flex-1 bg-transparent outline-none"
              style={{ fontSize: 12, color: "var(--neu-text)" }}
            />
          </div>
        </div>

        {/* Online strip */}
        {onlineChats.length > 0 && (
          <div
            className="flex-shrink-0 px-4 py-2.5"
            style={{ borderBottom: "1px solid var(--neu-border)" }}
          >
            <p
              className="uppercase mb-2"
              style={{
                fontSize: 9,
                letterSpacing: "0.12em",
                color: "var(--neu-text-muted)",
              }}
            >
              在线
            </p>
            <div className="flex gap-3 overflow-x-auto no-scrollbar">
              {onlineChats.map((c) => (
                <motion.div
                  key={c.other_user.user_id}
                  whileTap={{ scale: 0.93 }}
                  onClick={() => setSelectedId(c.other_user.user_id)}
                  className="flex flex-col items-center gap-1 flex-shrink-0 cursor-pointer"
                >
                  <div className="relative">
                    <div
                      className="rounded-full overflow-hidden"
                      style={{
                        width: 36,
                        height: 36,
                        border:
                          selectedId === c.other_user.user_id
                            ? "2px solid var(--neu-accent-primary)"
                            : "1.5px solid var(--neu-border)",
                      }}
                    >
                      <img
                        src={normalizeAvatar(c.other_user.avatar_url)}
                        alt={c.other_user.nickname}
                        className="w-full h-full object-cover"
                      />
                    </div>
                    <span
                      className="absolute bottom-0 right-0 rounded-full"
                      style={{
                        width: 8,
                        height: 8,
                        background: "var(--neu-success)",
                        border: "1.5px solid var(--neu-bg)",
                      }}
                    />
                  </div>
                  <span
                    style={{ fontSize: 10, color: "var(--neu-text-muted)" }}
                  >
                    {c.other_user.nickname}
                  </span>
                </motion.div>
              ))}
            </div>
          </div>
        )}

        {/* Tab pills */}
        <div className="flex-shrink-0 flex items-center gap-2 px-4 pt-2 pb-1">
          {(["private", "group"] as const).map((tab) => (
            <motion.button
              whileTap={{ scale: 0.94 }}
              key={tab}
              className="rounded-full px-2 py-0.5 outline-none transition-all"
              style={
                currentTab === tab
                  ? {
                      background: "var(--neu-accent-primary)",
                      color: "var(--text-on-accent, white)",
                      fontSize: 11,
                      boxShadow: "0 2px 8px var(--neu-glow)",
                    }
                  : {
                      background: "var(--neu-bg)",
                      color: "var(--neu-text-muted)",
                      fontSize: 11,
                      border: "1px solid var(--neu-border)",
                    }
              }
              onClick={() => handleTabChange(tab)}
            >
              {tab === "private" ? "私聊" : "群聊"}
            </motion.button>
          ))}
        </div>

        {/* Chat list */}
        <div className="flex-1 overflow-y-auto no-scrollbar px-2 py-2">
          {currentTab === "private" ? (
            loadingPrivate ? (
              <ChatSkeleton />
            ) : (
              <>
                {totalUnread > 0 && (
                  <p
                    className="uppercase px-2 mb-1"
                    style={{
                      fontSize: 9,
                      letterSpacing: "0.12em",
                      color: "var(--neu-text-muted)",
                    }}
                  >
                    未读
                  </p>
                )}
                {filteredPrivate.map((chat) => (
                  <PrivateChatListItem
                    key={chat.other_user.user_id}
                    chat={chat}
                    isSelected={selectedId === chat.other_user.user_id}
                    onClick={() => setSelectedId(chat.other_user.user_id)}
                    compact
                  />
                ))}
              </>
            )
          ) : loadingGroup ? (
            <ChatSkeleton />
          ) : (
            filteredGroups.map((group) => (
              <GroupChatListItem
                key={group.group.group_id}
                group={group}
                isSelected={selectedId === `group-${group.group.group_id}`}
                onClick={() => setSelectedId(`group-${group.group.group_id}`)}
              />
            ))
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
              <div
                className="flex items-center justify-center rounded-full neu-surface-flat"
                style={{ width: 72, height: 72 }}
              >
                <MessageSquareDashed
                  style={{
                    width: 28,
                    height: 28,
                    color: "var(--neu-text-muted)",
                    opacity: 0.5,
                  }}
                  strokeWidth={1.25}
                />
              </div>
              <div className="text-center">
                <p style={{ fontSize: 16, color: "var(--neu-text)" }}>
                  选择一个会话
                </p>
                <p
                  style={{
                    fontSize: 13,
                    color: "var(--neu-text-muted)",
                    marginTop: 4,
                  }}
                >
                  从左侧选择联系人开始聊天
                </p>
              </div>
            </motion.div>
          )}
        </AnimatePresence>
      </div>

      {/* New conversation modal (desktop center dialog) */}
      <AnimatePresence>
        {showNewChat && (
          <>
            <motion.div
              key="backdrop"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              transition={{ duration: 0.18 }}
              className="fixed inset-0 z-50"
              style={{
                background: "rgba(0,0,0,0.42)",
                backdropFilter: "blur(5px)",
              }}
              onClick={() => setShowNewChat(false)}
            />
            <div className="fixed inset-0 z-[60] flex items-center justify-center pointer-events-none">
              <motion.div
                key="modal"
                ref={newChatTrapRef}
                role="dialog"
                aria-modal="true"
                aria-label="发起对话"
                initial={{ opacity: 0, scale: 0.93, y: 14 }}
                animate={{ opacity: 1, scale: 1, y: 0 }}
                exit={{ opacity: 0, scale: 0.93, y: 14 }}
                transition={{ type: "spring", damping: 26, stiffness: 300 }}
                className="overflow-hidden pointer-events-auto"
                style={{
                  width: 440,
                  height: "68vh",
                  maxHeight: 600,
                  borderRadius: 24,
                  background: "var(--neu-bg)",
                  boxShadow:
                    "0 24px 80px rgba(0,0,0,0.24), 0 0 0 1px var(--neu-border)",
                }}
              >
                <NewConversationSheet
                  defaultTab={newChatTab}
                  onClose={() => setShowNewChat(false)}
                  onNewChat={(id) => {
                    setSelectedId(String(id));
                    setShowNewChat(false);
                  }}
                  onCreateGroup={() => {
                    setShowNewChat(false);
                    setShowCreateGroup(true);
                  }}
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
            <motion.div
              key="backdrop2"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              transition={{ duration: 0.18 }}
              className="fixed inset-0 z-50"
              style={{
                background: "rgba(0,0,0,0.42)",
                backdropFilter: "blur(5px)",
              }}
              onClick={() => setShowCreateGroup(false)}
            />
            <div className="fixed inset-0 z-[60] flex items-center justify-center pointer-events-none">
              <motion.div
                key="modal2"
                ref={groupTrapRef}
                role="dialog"
                aria-modal="true"
                aria-label="创建群组"
                initial={{ opacity: 0, scale: 0.93, y: 14 }}
                animate={{ opacity: 1, scale: 1, y: 0 }}
                exit={{ opacity: 0, scale: 0.93, y: 14 }}
                transition={{ type: "spring", damping: 26, stiffness: 300 }}
                className="overflow-hidden pointer-events-auto"
                style={{
                  width: 480,
                  height: "78vh",
                  maxHeight: 700,
                  borderRadius: 24,
                  background: "var(--neu-bg)",
                  boxShadow:
                    "0 24px 80px rgba(0,0,0,0.24), 0 0 0 1px var(--neu-border)",
                }}
              >
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
  useDocumentTitle("消息");
  if (platform === "desktop") return <MessagesDesktop />;
  return <MessagesMobile />;
}
