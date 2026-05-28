import { useState, useMemo } from "react";
import { useNavigate } from "react-router";
import { ArrowLeft, MapPin, Search, Map, Radar } from "lucide-react";
import { motion } from "motion/react";
import { usePlatform } from "../contexts/PlatformContext";
import { ListSkeleton } from "../components/ui/skeleton";
import {
  TabletRail,
  NearbyPulseCard,
  MoodCheckinCard,
  SuggestedFollowsCard,
  StreakCard,
  VibeCard,
} from "../components/TabletRail";
import { useHomeData } from "@/lib/hooks/use-home-data";
import { useFollowToggle } from "@/lib/hooks/use-follow-toggle";
import { useDocumentTitle } from "@/lib/hooks/use-document-title";
import { goBack } from "@/lib/utils/navigation";
import { normalizeAvatar, formatDistance } from "@/lib/utils";

const SORT_OPTIONS = ["距离", "在线", "共同好友"];

export function NearbyAll() {
  const navigate = useNavigate();
  const { platform } = usePlatform();
  const { nearbyUsers, loading, error, refresh } = useHomeData();
  const isTablet = platform === "tablet";
  const { toggledIds: followedIds, toggle } = useFollowToggle();
  const [search, setSearch] = useState("");
  const [sort, setSort] = useState("距离");
  useDocumentTitle("附近的人");

  let list = useMemo(() => {
    let items = nearbyUsers.filter(
      (u) =>
        search === "" ||
        u.user.nickname.toLowerCase().includes(search.toLowerCase()),
    );
    if (sort === "在线")
      items = [...items].sort(
        (a, b) =>
          (b.presence_status === "online" ? 1 : 0) -
          (a.presence_status === "online" ? 1 : 0),
      );
    return items;
  }, [nearbyUsers, search, sort]);

  const tags = useMemo(
    () => [
      "全部",
      ...new Set(
        nearbyUsers
          .map((u) => u.tags ?? [])
          .flat()
          .filter(Boolean),
      ),
    ],
    [nearbyUsers],
  );
  const [tagFilt, setTagFilt] = useState("全部");

  const filtered = useMemo(() => {
    if (tagFilt === "全部") return list;
    return list.filter((u) => (u.tags ?? []).includes(tagFilt));
  }, [list, tagFilt]);

  const onlineCount = filtered.filter(
    (u) => u.presence_status === "online",
  ).length;

  return (
    <>
      <div
        className="sticky top-0 z-10 pt-3 pb-3"
        style={{
          backgroundColor: "var(--neu-bg)",
          borderBottom: "1px solid var(--neu-border)",
        }}
      >
        <div
          className={isTablet ? "flex gap-6 mx-auto" : "px-5"}
          style={isTablet ? { maxWidth: 964, width: "100%" } : {}}
        >
          <div
            className={
              isTablet ? "mx-auto w-full lg:mx-0 px-5 min-w-0" : "w-full"
            }
            style={isTablet ? { maxWidth: 640 } : {}}
          >
            <div className="flex items-center gap-3 mb-3">
              <motion.button
                whileTap={{ scale: 0.88 }}
                onClick={() => goBack(navigate)}
                aria-label="返回"
                className="flex items-center justify-center rounded-full outline-none neu-surface-flat"
                style={{ width: 40, height: 40 }}
              >
                <ArrowLeft
                  style={{
                    width: 15,
                    height: 15,
                    color: "var(--neu-text-muted)",
                  }}
                  strokeWidth={1.5}
                />
              </motion.button>
              <h1
                className="flex-1"
                style={{ fontSize: 18, color: "var(--neu-text)" }}
              >
                附近的人
              </h1>
              <motion.button
                whileTap={{ scale: 0.94 }}
                onClick={() => navigate("/location")}
                aria-label="地图视图"
                className="flex items-center justify-center rounded-full outline-none neu-surface-flat"
                style={{ width: 40, height: 40 }}
              >
                <Map
                  style={{
                    width: 15,
                    height: 15,
                    color: "var(--neu-accent-primary)",
                  }}
                  strokeWidth={1.5}
                />
              </motion.button>
            </div>

            <div className="flex items-center gap-3 px-4 py-2.5 rounded-[14px] neu-input-surface">
              <Search
                style={{
                  width: 14,
                  height: 14,
                  color: "var(--neu-text-muted)",
                  flexShrink: 0,
                }}
                strokeWidth={1.5}
              />
              <input
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="搜索用户…"
                className="flex-1 bg-transparent outline-none"
                style={{ fontSize: 13, color: "var(--neu-text)" }}
              />
            </div>
          </div>
          {isTablet && (
            <div
              className="hidden lg:block flex-shrink-0"
              style={{ width: 300 + 20 }}
            />
          )}
        </div>
      </div>

      <div
        className={isTablet ? "flex gap-6 mx-auto pt-5 pb-8" : "px-5 pt-5 pb-8"}
        style={
          isTablet
            ? { maxWidth: 964, width: "100%" }
            : {
                maxWidth: 640,
                marginLeft: "auto",
                marginRight: "auto",
                width: "100%",
              }
        }
      >
        <div
          className={
            isTablet ? "mx-auto w-full lg:mx-0 px-5 min-w-0" : "w-full"
          }
          style={isTablet ? { maxWidth: 640 } : {}}
        >
          <div className="neu-ambient-glow mb-6">
            <div
              className="neu-surface-hero rounded-[22px] flex items-center gap-4"
              style={{ padding: "16px 18px" }}
            >
              <div
                className="flex items-center justify-center rounded-full flex-shrink-0"
                style={{
                  width: 44,
                  height: 44,
                  background: "var(--neu-accent-primary)",
                  boxShadow: "0 4px 14px var(--neu-glow)",
                }}
              >
                <Radar
                  style={{
                    width: 19,
                    height: 19,
                    color: "var(--text-on-accent, white)",
                  }}
                  strokeWidth={2}
                />
              </div>
              <div className="flex-1">
                <p
                  style={{
                    fontSize: 13,
                    color: "var(--neu-text)",
                    fontWeight: 500,
                  }}
                >
                  附近的人
                </p>
                <p
                  style={{
                    fontSize: 11,
                    color: "var(--neu-text-muted)",
                    marginTop: 2,
                  }}
                >
                  <span style={{ color: "#22c55e" }}>{onlineCount} 人在线</span>
                  <span style={{ marginLeft: 8 }}>
                    · 共 {filtered.length} 人
                  </span>
                </p>
              </div>
            </div>
          </div>

          <div className="flex items-center gap-1 mb-3 p-1 rounded-full neu-input-surface w-fit">
            {SORT_OPTIONS.map((s) => (
              <motion.button
                key={s}
                whileTap={{ scale: 0.93 }}
                onClick={() => setSort(s)}
                className="px-3.5 py-1 rounded-full outline-none transition-all"
                style={
                  sort === s
                    ? {
                        background: "var(--neu-accent-primary)",
                        color: "var(--text-on-accent, white)",
                        fontSize: 12,
                        fontWeight: 500,
                      }
                    : { color: "var(--neu-text-muted)", fontSize: 12 }
                }
              >
                {s}
              </motion.button>
            ))}
          </div>

          <div className="flex gap-1.5 overflow-x-auto no-scrollbar mb-5">
            {tags.slice(0, 12).map((t) => {
              const active = tagFilt === t;
              return (
                <motion.button
                  key={t}
                  whileTap={{ scale: 0.94 }}
                  onClick={() => setTagFilt(t)}
                  className="flex-shrink-0 px-3 py-1 rounded-full outline-none transition-all neu-surface-ghost"
                  style={
                    active
                      ? {
                          background: "var(--neu-accent-soft)",
                          color: "var(--neu-accent-primary)",
                          fontSize: 12,
                          fontWeight: 500,
                          borderColor: "var(--neu-accent-soft-border)",
                        }
                      : { fontSize: 12, color: "var(--neu-text-muted)" }
                  }
                >
                  {t}
                </motion.button>
              );
            })}
          </div>

          {loading ? (
            <ListSkeleton rows={5} />
          ) : error && filtered.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 gap-3">
              <p style={{ fontSize: 14, color: "var(--neu-text-muted)" }}>
                {error}
              </p>
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
            </div>
          ) : filtered.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 gap-3">
              <p style={{ fontSize: 14, color: "var(--neu-text-muted)" }}>
                暂无附近用户
              </p>
            </div>
          ) : (
            <div
              className="flex flex-col rounded-[16px] overflow-hidden"
              style={{
                gap: 1,
                background: "var(--neu-border)",
                border: "1px solid var(--neu-border)",
              }}
            >
              {filtered.map((user) => {
                const followed = followedIds.has(user.user.user_id);
                return (
                  <div
                    key={user.user.user_id}
                    onClick={() => navigate(`/user/${user.user.user_id}`)}
                    className="flex items-center gap-3 px-4 py-3 cursor-pointer transition-colors"
                    style={{ background: "var(--neu-bg)" }}
                    onMouseEnter={(e) =>
                      (e.currentTarget.style.background = "var(--neu-border)")
                    }
                    onMouseLeave={(e) =>
                      (e.currentTarget.style.background = "var(--neu-bg)")
                    }
                  >
                    <div className="relative flex-shrink-0">
                      <div
                        className="rounded-full overflow-hidden"
                        style={{ width: 42, height: 42 }}
                      >
                        <img
                          src={normalizeAvatar(user.user.avatar_url)}
                          alt={user.user.nickname}
                          className="w-full h-full object-cover"
                        />
                      </div>
                      {user.presence_status === "online" && (
                        <span
                          className="absolute bottom-0 right-0 rounded-full"
                          style={{
                            width: 10,
                            height: 10,
                            background: "#22c55e",
                            border: "2px solid var(--neu-bg)",
                          }}
                        />
                      )}
                    </div>
                    <div className="flex-1 min-w-0">
                      <p
                        className="truncate"
                        style={{ fontSize: 14, color: "var(--neu-text)" }}
                      >
                        {user.user.nickname}
                      </p>
                      <div className="flex items-center gap-1.5 mt-0.5">
                        {user.tags?.map((tag) => (
                          <span
                            key={tag}
                            style={{
                              fontSize: 11,
                              color: "var(--neu-accent-primary)",
                            }}
                          >
                            #{tag}
                          </span>
                        ))}
                        {user.social_context?.mutual_friends_count ? (
                          <>
                            <span
                              style={{
                                fontSize: 10,
                                color: "var(--neu-text-muted)",
                                opacity: 0.5,
                              }}
                            >
                              ·
                            </span>
                            <span
                              style={{
                                fontSize: 11,
                                color: "var(--neu-text-muted)",
                              }}
                            >
                              {user.social_context.mutual_friends_count}位共同
                            </span>
                          </>
                        ) : null}
                      </div>
                    </div>
                    <div className="flex flex-col items-end gap-1.5">
                      <div className="flex items-center gap-0.5">
                        <MapPin
                          style={{
                            width: 10,
                            height: 10,
                            color: "var(--neu-text-muted)",
                          }}
                          strokeWidth={1.5}
                        />
                        <span
                          style={{
                            fontSize: 11,
                            color: "var(--neu-text-muted)",
                          }}
                        >
                          {formatDistance(user.location?.distance)}
                        </span>
                      </div>
                      <motion.button
                        whileTap={{ scale: 0.93 }}
                        onClick={(e) => {
                          e.stopPropagation();
                          toggle(user.user.user_id);
                        }}
                        className="px-3 py-0.5 rounded-full outline-none transition-all"
                        style={
                          followed
                            ? {
                                background: "transparent",
                                color: "var(--neu-text-muted)",
                                border: "1px solid var(--neu-border-pressed)",
                                fontSize: 11,
                              }
                            : {
                                background: "var(--neu-accent-primary)",
                                color: "var(--text-on-accent, white)",
                                border: "1px solid transparent",
                                fontSize: 11,
                              }
                        }
                      >
                        {followed ? "已关注" : "关注"}
                      </motion.button>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {isTablet && (
          <aside
            className="hidden lg:block flex-shrink-0 pr-5"
            style={{ width: 300 + 20 }}
          >
            <div style={{ width: 300 }}>
              <TabletRail>
                <NearbyPulseCard />
                <MoodCheckinCard />
                <SuggestedFollowsCard />
                <StreakCard />
                <VibeCard />
              </TabletRail>
            </div>
          </aside>
        )}
      </div>
    </>
  );
}
