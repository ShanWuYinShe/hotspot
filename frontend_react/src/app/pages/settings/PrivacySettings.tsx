import { useState, useEffect } from "react";
import { useNavigate } from "react-router";
import { ArrowLeft, Loader2 } from "lucide-react";
import { motion } from "motion/react";
import { toast } from "sonner";

import { usersService } from "@/lib/api/services";
import { isSuccess } from "@/lib/api/client";
import { useDocumentTitle } from "@/lib/hooks/use-document-title";
import { withMinimumLoadingTime } from "@/lib/utils/loading";
import type {
  PrivacyLevel,
  PrivateChatPermission,
  UserSettings,
} from "@/lib/types/api";

function NeuToggle({ on, onToggle }: { on: boolean; onToggle: () => void }) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={on}
      onClick={onToggle}
      className="relative flex items-center rounded-full flex-shrink-0 outline-none"
      style={{
        width: 50,
        height: 28,
        background: on ? "var(--neu-accent-primary)" : "var(--neu-bg)",
        boxShadow: on
          ? "inset 0 1px 2px rgba(0,0,0,0.12)"
          : "inset 2px 2px 5px var(--neu-shadow-dark), inset -2px -2px 5px var(--neu-shadow-light)",
        border: on ? "none" : "1px solid var(--neu-border)",
        transition: "background 0.25s cubic-bezier(0.4,0,0.2,1)",
      }}
    >
      <motion.span
        animate={{ x: on ? 23 : 2 }}
        transition={{ type: "spring", stiffness: 420, damping: 30 }}
        className="block rounded-full"
        style={{
          position: "absolute",
          top: 3,
          width: 22,
          height: 22,
          background: on ? "#FFFFFF" : "var(--neu-text-muted)",
          boxShadow: "0 1px 4px rgba(0,0,0,0.26)",
        }}
      />
    </button>
  );
}

function ToggleRow({
  label,
  description,
  on,
  onToggle,
}: {
  label: string;
  description: string;
  on: boolean;
  onToggle: () => void;
}) {
  return (
    <div
      className="flex items-center justify-between rounded-[18px] px-4 py-3.5"
      style={{
        background: "var(--neu-bg)",
        boxShadow:
          "inset 2px 2px 5px var(--neu-shadow-dark), inset -2px -2px 5px var(--neu-shadow-light)",
        border: "1px solid var(--neu-border)",
      }}
    >
      <div>
        <p style={{ fontSize: 14, fontWeight: 500, color: "var(--neu-text)" }}>
          {label}
        </p>
        <p style={{ fontSize: 12, color: "var(--neu-text-muted)" }}>
          {description}
        </p>
      </div>
      <NeuToggle on={on} onToggle={onToggle} />
    </div>
  );
}

export function PrivacySettings() {
  const navigate = useNavigate();
  useDocumentTitle("隐私设置");

  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);

  const [privacyLevel, setPrivacyLevel] = useState<PrivacyLevel>("public");
  const [allowPrivateChat, setAllowPrivateChat] =
    useState<PrivateChatPermission>("everyone");
  const [allowFollow, setAllowFollow] = useState(true);
  const [searchable, setSearchable] = useState(true);
  const [locationSearchable, setLocationSearchable] = useState(true);
  const [locationVisible, setLocationVisible] = useState(true);
  const [presenceVisible, setPresenceVisible] = useState(true);
  const [socialListsVisible, setSocialListsVisible] = useState(true);

  useEffect(() => {
    (async () => {
      try {
        const response = await withMinimumLoadingTime(() =>
          usersService.getMySettings(),
        );
        if (isSuccess(response) && response.data) {
          const s: UserSettings = response.data;
          setPrivacyLevel(s.privacy_level);
          setAllowPrivateChat(s.allow_private_chat);
          setAllowFollow(s.allow_follow);
          setSearchable(s.searchable);
          setLocationSearchable(s.location_searchable);
          setLocationVisible(s.location_visible);
          setPresenceVisible(s.presence_visible);
          setSocialListsVisible(s.social_lists_visible);
        }
      } catch {
        toast.error("加载隐私设置失败");
      } finally {
        setIsLoading(false);
      }
    })();
  }, []);

  const saveSettings = async () => {
    setIsSaving(true);
    try {
      const response = await usersService.updateMySettings({
        privacy_level: privacyLevel,
        allow_private_chat: allowPrivateChat,
        allow_follow: allowFollow,
        searchable,
        location_searchable: locationSearchable,
        location_visible: locationVisible,
        presence_visible: presenceVisible,
        social_lists_visible: socialListsVisible,
      });
      if (isSuccess(response)) {
        toast.success("隐私设置已保存");
      } else {
        toast.error(response.message || "保存失败");
      }
    } catch {
      toast.error("保存失败");
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <>
      {/* Header */}
      <div
        className="flex-shrink-0 flex items-center gap-3 px-4"
        style={{ height: 56, borderBottom: "1px solid var(--neu-border)" }}
      >
        <motion.button
          whileTap={{ scale: 0.88 }}
          onClick={() => navigate(-1)}
          className="flex items-center justify-center rounded-full outline-none"
          style={{
            width: 36,
            height: 36,
            background: "var(--neu-bg)",
            boxShadow:
              "4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light)",
            border: "1px solid var(--neu-border)",
          }}
        >
          <ArrowLeft
            style={{ width: 15, height: 15, color: "var(--neu-text-muted)" }}
            strokeWidth={1.5}
          />
        </motion.button>
        <p style={{ fontSize: 16, fontWeight: 600, color: "var(--neu-text)" }}>
          隐私设置
        </p>
      </div>

      {/* Content */}
      <motion.div
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.28, ease: [0.22, 1, 0.36, 1] }}
        className="flex flex-col px-5 pt-4 pb-10 gap-5"
        style={{
          maxWidth: 560,
          marginLeft: "auto",
          marginRight: "auto",
          width: "100%",
        }}
      >
        {isLoading ? (
          <div
            className="rounded-[22px] p-5 flex flex-col gap-4"
            style={{
              background: "var(--neu-bg)",
              boxShadow:
                "7px 7px 16px var(--neu-shadow-dark), -7px -7px 16px var(--neu-shadow-light)",
              border: "1px solid var(--neu-border)",
            }}
          >
            {Array.from({ length: 5 }).map((_, i) => (
              <div
                key={i}
                className="h-11 w-full rounded-[18px]"
                style={{
                  background: "var(--neu-bg)",
                  boxShadow:
                    "inset 2px 2px 5px var(--neu-shadow-dark), inset -2px -2px 5px var(--neu-shadow-light)",
                  border: "1px solid var(--neu-border)",
                }}
              />
            ))}
          </div>
        ) : (
          <>
            {/* Basic Visibility */}
            <div
              className="rounded-[22px] p-5"
              style={{
                background: "var(--neu-bg)",
                boxShadow:
                  "7px 7px 16px var(--neu-shadow-dark), -7px -7px 16px var(--neu-shadow-light)",
                border: "1px solid var(--neu-border)",
              }}
            >
              <p
                style={{
                  fontSize: 10,
                  letterSpacing: "0.14em",
                  textTransform: "uppercase",
                  color: "var(--neu-text-muted)",
                  marginBottom: 4,
                }}
              >
                基础可见性
              </p>
              <p
                style={{
                  fontSize: 12,
                  color: "var(--neu-text-muted)",
                  marginBottom: 16,
                }}
              >
                控制大部分信息默认可见范围
              </p>

              <div className="flex flex-col gap-3">
                {/* Privacy level */}
                <div
                  className="rounded-[18px] p-4"
                  style={{
                    background: "var(--neu-bg)",
                    boxShadow:
                      "inset 2px 2px 5px var(--neu-shadow-dark), inset -2px -2px 5px var(--neu-shadow-light)",
                    border: "1px solid var(--neu-border)",
                  }}
                >
                  <label
                    htmlFor="privacy-level"
                    style={{
                      fontSize: 13,
                      color: "var(--neu-text-muted)",
                      display: "block",
                      marginBottom: 8,
                    }}
                  >
                    隐私级别
                  </label>
                  <select
                    id="privacy-level"
                    value={privacyLevel}
                    onChange={(e) =>
                      setPrivacyLevel(e.target.value as PrivacyLevel)
                    }
                    className="w-full rounded-[18px] px-4 py-2.5 outline-none"
                    style={{
                      border: "1px solid var(--neu-border)",
                      background: "var(--neu-bg)",
                      color: "var(--neu-text)",
                    }}
                  >
                    <option value="public">公开</option>
                    <option value="private">仅自己</option>
                  </select>
                </div>

                <ToggleRow
                  label="位置可见"
                  description="允许展示你的地理位置"
                  on={locationVisible}
                  onToggle={() => setLocationVisible((v) => !v)}
                />
                <ToggleRow
                  label="在线状态可见"
                  description="允许显示在线/离线状态"
                  on={presenceVisible}
                  onToggle={() => setPresenceVisible((v) => !v)}
                />
                <ToggleRow
                  label="好友/关注/粉丝列表可见"
                  description="控制你的社交列表是否对他人可见"
                  on={socialListsVisible}
                  onToggle={() => setSocialListsVisible((v) => !v)}
                />
              </div>
            </div>

            {/* Social & Chat */}
            <div
              className="rounded-[22px] p-5"
              style={{
                background: "var(--neu-bg)",
                boxShadow:
                  "7px 7px 16px var(--neu-shadow-dark), -7px -7px 16px var(--neu-shadow-light)",
                border: "1px solid var(--neu-border)",
              }}
            >
              <p
                style={{
                  fontSize: 10,
                  letterSpacing: "0.14em",
                  textTransform: "uppercase",
                  color: "var(--neu-text-muted)",
                  marginBottom: 4,
                }}
              >
                社交与私聊
              </p>
              <p
                style={{
                  fontSize: 12,
                  color: "var(--neu-text-muted)",
                  marginBottom: 16,
                }}
              >
                管理关注与私聊权限
              </p>

              <div className="flex flex-col gap-3">
                <ToggleRow
                  label="允许关注"
                  description="关闭后他人关注需审批"
                  on={allowFollow}
                  onToggle={() => setAllowFollow((v) => !v)}
                />

                <div
                  className="rounded-[18px] p-4"
                  style={{
                    background: "var(--neu-bg)",
                    boxShadow:
                      "inset 2px 2px 5px var(--neu-shadow-dark), inset -2px -2px 5px var(--neu-shadow-light)",
                    border: "1px solid var(--neu-border)",
                  }}
                >
                  <label
                    htmlFor="private-chat"
                    style={{
                      fontSize: 13,
                      color: "var(--neu-text-muted)",
                      display: "block",
                      marginBottom: 8,
                    }}
                  >
                    私聊权限
                  </label>
                  <select
                    id="private-chat"
                    value={allowPrivateChat}
                    onChange={(e) =>
                      setAllowPrivateChat(
                        e.target.value as PrivateChatPermission,
                      )
                    }
                    className="w-full rounded-[18px] px-4 py-2.5 outline-none"
                    style={{
                      boxShadow:
                        "inset 2px 2px 5px var(--neu-shadow-dark), inset -2px -2px 5px var(--neu-shadow-light)",
                      border: "1px solid var(--neu-border)",
                      background: "var(--neu-bg)",
                      color: "var(--neu-text)",
                    }}
                  >
                    <option value="everyone">所有人可私聊</option>
                    <option value="friends">仅好友可私聊</option>
                    <option value="nobody">不允许私聊</option>
                  </select>
                </div>
              </div>
            </div>

            {/* Search & Discovery */}
            <div
              className="rounded-[22px] p-5"
              style={{
                background: "var(--neu-bg)",
                boxShadow:
                  "7px 7px 16px var(--neu-shadow-dark), -7px -7px 16px var(--neu-shadow-light)",
                border: "1px solid var(--neu-border)",
              }}
            >
              <p
                style={{
                  fontSize: 10,
                  letterSpacing: "0.14em",
                  textTransform: "uppercase",
                  color: "var(--neu-text-muted)",
                  marginBottom: 4,
                }}
              >
                搜索与发现
              </p>
              <p
                style={{
                  fontSize: 12,
                  color: "var(--neu-text-muted)",
                  marginBottom: 16,
                }}
              >
                控制你在搜索/附近中的展示
              </p>

              <div className="flex flex-col gap-3">
                <ToggleRow
                  label="允许被搜索"
                  description="关闭后不会出现在搜索结果"
                  on={searchable}
                  onToggle={() => setSearchable((v) => !v)}
                />
                <ToggleRow
                  label="附近可见"
                  description="允许出现在附近的人"
                  on={locationSearchable}
                  onToggle={() => setLocationSearchable((v) => !v)}
                />
              </div>
            </div>

            {/* Save button */}
            <motion.button
              whileTap={{ scale: 0.96 }}
              onClick={saveSettings}
              disabled={isSaving}
              className="flex h-11 w-full items-center justify-center rounded-full text-sm font-medium transition-opacity hover:opacity-90 disabled:opacity-50"
              style={{
                background: "var(--neu-accent-primary)",
                color: "var(--text-on-accent, white)",
                boxShadow: "0 4px 14px var(--neu-glow)",
              }}
            >
              {isSaving && (
                <Loader2
                  style={{ width: 16, height: 16, marginRight: 8 }}
                  className="animate-spin"
                />
              )}
              保存设置
            </motion.button>
          </>
        )}
      </motion.div>
    </>
  );
}
