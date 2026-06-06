import { useState } from "react";
import { useParams, useNavigate } from "react-router";
import {
  ArrowLeft,
  MapPin,
  MessageCircle,
  UserPlus,
  UserCheck,
  Share2,
  Heart,
  Grid3X3,
  Bookmark,
  MoreHorizontal,
} from "lucide-react";
import { EmptyState } from "../components/ui/EmptyState";
import { motion } from "motion/react";

import { useUserProfileData } from "@/lib/hooks/use-user-profile-data";
import { useDocumentTitle } from "@/lib/hooks/use-document-title";
import { goBack } from "@/lib/utils/navigation";

type Tab = "posts" | "liked" | "saved";

export function UserProfile() {
  const { userId } = useParams<{ userId: string }>();
  const navigate = useNavigate();
  const {
    user,
    loading,
    error,
    isFollowing,
    followLoading,
    toggleFollow,
    refresh,
  } = useUserProfileData(userId);
  const [activeTab, setActiveTab] = useState<Tab>("posts");
  useDocumentTitle(user?.user?.nickname || "用户");

  const fmtNum = (n: number) =>
    n >= 1000 ? `${(n / 1000).toFixed(1)}k` : String(n);

  if (loading || !user) {
    return (
      <div
        className="flex flex-col items-center justify-center gap-3"
        style={{ height: "100%", backgroundColor: "var(--neu-bg)" }}
      >
        <p style={{ fontSize: 14, color: "var(--neu-text-muted)" }}>
          {error || "加载中…"}
        </p>
        {error && (
          <motion.button
            whileTap={{ scale: 0.94 }}
            onClick={refresh}
            className="px-4 py-2 rounded-full outline-none"
            style={{
              background: "var(--neu-accent-primary)",
              color: "var(--text-on-accent, white)",
              fontSize: 13,
            }}
          >
            重试
          </motion.button>
        )}
      </div>
    );
  }

  const { user: userInfo, bio, tags, location, stats } = user;

  return (
    <div
      className="flex flex-col"
      style={{
        height: "100%",
        backgroundColor: "var(--neu-bg)",
        overflowY: "auto",
      }}
    >
      {/* Banner */}
      <div
        className="relative overflow-hidden"
        style={{ height: "clamp(150px, 25vw, 210px)", flexShrink: 0 }}
      >
        <div
          className="absolute inset-0"
          style={{
            background:
              "linear-gradient(135deg, var(--neu-accent-primary), rgba(139,92,246,0.85))",
          }}
        />
        <div
          className="absolute inset-0"
          style={{
            background:
              "linear-gradient(to bottom, rgba(0,0,0,0.15) 0%, rgba(0,0,0,0.55) 100%)",
          }}
        />

        {/* Top buttons */}
        <div className="absolute top-0 left-0 right-0 flex items-center justify-between px-4 pt-4">
          <motion.button
            whileTap={{ scale: 0.88 }}
            onClick={() => goBack(navigate)}
            aria-label="返回"
            className="flex items-center justify-center rounded-full outline-none"
            style={{
              width: 36,
              height: 36,
              background: "rgba(0,0,0,0.35)",
              backdropFilter: "blur(10px)",
              border: "1px solid rgba(255,255,255,0.25)",
            }}
          >
            <ArrowLeft
              style={{ width: 15, height: 15, color: "white" }}
              strokeWidth={1.75}
            />
          </motion.button>
          <div className="flex gap-2">
            <motion.button
              whileTap={{ scale: 0.88 }}
              aria-label="分享主页"
              className="flex items-center justify-center rounded-full outline-none"
              style={{
                width: 36,
                height: 36,
                background: "rgba(0,0,0,0.35)",
                backdropFilter: "blur(10px)",
                border: "1px solid rgba(255,255,255,0.25)",
              }}
            >
              <Share2
                style={{ width: 14, height: 14, color: "white" }}
                strokeWidth={1.75}
              />
            </motion.button>
            <motion.button
              whileTap={{ scale: 0.88 }}
              aria-label="更多操作"
              className="flex items-center justify-center rounded-full outline-none"
              style={{
                width: 36,
                height: 36,
                background: "rgba(0,0,0,0.35)",
                backdropFilter: "blur(10px)",
                border: "1px solid rgba(255,255,255,0.25)",
              }}
            >
              <MoreHorizontal
                style={{ width: 14, height: 14, color: "white" }}
                strokeWidth={1.75}
              />
            </motion.button>
          </div>
        </div>
      </div>

      {/* Avatar + info */}
      <div
        style={{
          maxWidth: 560,
          marginLeft: "auto",
          marginRight: "auto",
          width: "100%",
        }}
      >
        <div className="px-5 -mt-10 pb-4 flex flex-col">
          <div className="flex items-end justify-between mb-3">
            <div className="relative">
              <div
                className="rounded-[22px] overflow-hidden"
                style={{
                  width: 76,
                  height: 76,
                  border: "3px solid var(--neu-bg)",
                  boxShadow: "0 6px 20px rgba(0,0,0,0.22)",
                }}
              >
                {userInfo.avatar_url ? (
                  <img
                    src={userInfo.avatar_url}
                    alt={userInfo.nickname}
                    className="w-full h-full object-cover"
                  />
                ) : (
                  <div
                    className="w-full h-full flex items-center justify-center"
                    style={{ background: "var(--neu-accent-primary)" }}
                  >
                    <span
                      style={{
                        fontSize: 28,
                        color: "var(--text-on-accent, white)",
                      }}
                    >
                      {userInfo.nickname.charAt(0)}
                    </span>
                  </div>
                )}
              </div>
              {user.presence_status === "online" && (
                <span
                  className="absolute bottom-1 right-1 rounded-full"
                  style={{
                    width: 12,
                    height: 12,
                    background: "var(--neu-success)",
                    border: "2px solid var(--neu-bg)",
                  }}
                />
              )}
            </div>

            {/* Action buttons */}
            <div className="flex gap-2 pb-1">
              <motion.button
                whileTap={{ scale: 0.93 }}
                onClick={() => navigate(`/messages/${userInfo.user_id}`)}
                className="flex items-center gap-1.5 px-3.5 py-2 rounded-full outline-none"
                style={{
                  background: "var(--neu-bg)",
                  border: "1px solid var(--neu-border)",
                  boxShadow:
                    "4px 4px 10px var(--neu-shadow-dark), -4px -4px 10px var(--neu-shadow-light)",
                  fontSize: 12,
                  color: "var(--neu-text-muted)",
                }}
              >
                <MessageCircle
                  style={{ width: 14, height: 14 }}
                  strokeWidth={1.5}
                />
                私信
              </motion.button>
              <motion.button
                whileTap={{ scale: 0.93 }}
                onClick={toggleFollow}
                disabled={followLoading}
                className="flex items-center gap-1.5 px-4 py-2 rounded-full outline-none transition-all"
                style={
                  isFollowing
                    ? {
                        background: "var(--neu-bg)",
                        border: "1.5px solid var(--neu-accent-primary)",
                        color: "var(--neu-accent-primary)",
                        fontSize: 12,
                        boxShadow:
                          "4px 4px 10px var(--neu-shadow-dark), -4px -4px 10px var(--neu-shadow-light)",
                      }
                    : {
                        background: "var(--neu-accent-primary)",
                        color: "var(--text-on-accent, white)",
                        fontSize: 12,
                        boxShadow: "0 4px 14px var(--neu-glow)",
                        border: "1px solid transparent",
                      }
                }
              >
                {isFollowing ? (
                  <>
                    <UserCheck
                      style={{ width: 14, height: 14 }}
                      strokeWidth={1.5}
                    />
                    已关注
                  </>
                ) : (
                  <>
                    <UserPlus
                      style={{ width: 14, height: 14 }}
                      strokeWidth={1.5}
                    />
                    关注
                  </>
                )}
              </motion.button>
            </div>
          </div>

          <h1
            className="truncate"
            style={{ fontSize: 20, color: "var(--neu-text)" }}
          >
            {userInfo.nickname}
          </h1>
          <p
            style={{
              fontSize: 13,
              color: "var(--neu-text-muted)",
              marginTop: 1,
            }}
          >
            @{userInfo.user_id.slice(0, 8)}
          </p>
          {bio && (
            <p
              style={{
                fontSize: 13,
                color: "var(--neu-text-muted)",
                lineHeight: 1.65,
                marginTop: 8,
              }}
            >
              {bio}
            </p>
          )}

          {location && (
            <div className="flex items-center gap-1.5 mt-2 min-w-0">
              <MapPin
                style={{
                  width: 11,
                  height: 11,
                  color: "var(--neu-accent-primary)",
                }}
                strokeWidth={1.5}
                className="flex-shrink-0"
              />
              <span
                className="truncate"
                style={{ fontSize: 11, color: "var(--neu-text-muted)" }}
              >
                {location.location_name}
                {location.distance != null &&
                  ` · ${location.distance < 1000 ? `${Math.round(location.distance)}m` : `${(location.distance / 1000).toFixed(1)}km`}`}
              </span>
            </div>
          )}

          {/* Tags */}
          {tags && tags.length > 0 && (
            <div className="flex gap-2 mt-3 flex-wrap">
              {tags.map((t) => (
                <span
                  key={t}
                  className="px-2.5 py-0.5 rounded-full"
                  style={{
                    fontSize: 11,
                    background: "var(--neu-accent-soft)",
                    color: "var(--neu-accent-primary)",
                    border: "1px solid var(--neu-accent-soft-border)",
                  }}
                >
                  #{t}
                </span>
              ))}
            </div>
          )}

          {/* Stats */}
          {stats && (
            <div
              className="flex gap-0 mt-5 rounded-[20px] overflow-hidden"
              style={{
                background: "var(--neu-bg)",
                boxShadow:
                  "6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light)",
                border: "1px solid var(--neu-border)",
              }}
            >
              {[
                { label: "帖子", v: fmtNum(stats.moments_count) },
                { label: "粉丝", v: fmtNum(stats.followers_count) },
                { label: "关注", v: fmtNum(stats.following_count) },
              ].map((s, i) => (
                <div
                  key={s.label}
                  className="flex-1 flex flex-col items-center py-4 relative"
                >
                  {i > 0 && (
                    <div
                      className="absolute left-0 top-1/2 -translate-y-1/2"
                      style={{
                        width: 1,
                        height: 28,
                        background: "var(--neu-border)",
                      }}
                    />
                  )}
                  <span style={{ fontSize: 22, color: "var(--neu-text)" }}>
                    {s.v}
                  </span>
                  <span
                    className="uppercase mt-1"
                    style={{
                      fontSize: 9,
                      letterSpacing: "0.1em",
                      color: "var(--neu-text-muted)",
                    }}
                  >
                    {s.label}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Tabs */}
        <div
          className="flex"
          style={{
            borderBottom: "1px solid var(--neu-border)",
            paddingLeft: 16,
            paddingRight: 16,
            flexShrink: 0,
          }}
        >
          {(
            [
              ["posts", Grid3X3, "帖子"],
              ["liked", Heart, "喜欢"],
              ["saved", Bookmark, "收藏"],
            ] as const
          ).map(([id, Icon, label]) => {
            const active = activeTab === id;
            return (
              <motion.button
                key={id}
                whileTap={{ scale: 0.94 }}
                onClick={() => setActiveTab(id)}
                className="relative flex items-center justify-center gap-1.5 pb-3 pt-1.5 outline-none"
                style={{
                  flex: 1,
                  color: active
                    ? "var(--neu-accent-primary)"
                    : "var(--neu-text-muted)",
                  fontSize: 13,
                  fontWeight: active ? 500 : 400,
                }}
              >
                <Icon
                  style={{ width: 13, height: 13 }}
                  strokeWidth={active ? 2 : 1.5}
                />
                {label}
                {active && (
                  <motion.div
                    layoutId="user-profile-tab"
                    style={{
                      position: "absolute",
                      bottom: -1,
                      left: "18%",
                      right: "18%",
                      height: 2,
                      borderRadius: 1,
                      background: "var(--neu-accent-primary)",
                    }}
                    transition={{ type: "spring", stiffness: 420, damping: 34 }}
                  />
                )}
              </motion.button>
            );
          })}
        </div>

        {/* Content placeholder — moments gallery not yet implemented */}
        <EmptyState
          icon={Grid3X3}
          title={
            activeTab === "posts"
              ? "暂无帖子"
              : activeTab === "liked"
                ? "暂无喜欢"
                : "暂无收藏"
          }
        />
      </div>
    </div>
  );
}
