import { useState, useEffect, useCallback } from "react";
import { feedService } from "@/lib/api/services/feed";
import { useLocationStore } from "@/lib/stores/location";
import { useAuthStore } from "@/lib/stores/auth";
import { isSuccess } from "@/lib/api/client";
import type { UserItem, GroupItem, TrendingBundle } from "@/lib/types/api";

interface HomeData {
  nearbyUsers: UserItem[];
  nearbyGroups: GroupItem[];
  trending: TrendingBundle | null;
  loading: boolean;
  error: string | null;
  refresh: () => void;
}

export function useHomeData(): HomeData {
  const [nearbyUsers, setNearbyUsers] = useState<UserItem[]>([]);
  const [nearbyGroups, setNearbyGroups] = useState<GroupItem[]>([]);
  const [trending, setTrending] = useState<TrendingBundle | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const geoPoint = useLocationStore((s) => s.geoPoint);
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);

    if (!isAuthenticated) {
      setLoading(false);
      return;
    }

    try {
      const point = geoPoint ?? { latitude: 34.26, longitude: 117.18 }; // fallback to Xuzhou
      const reference_point = point;

      const [usersRes, groupsRes, trendingRes] = await Promise.allSettled([
        feedService.getUsers({
          reference_point,
          radius: 5000,
          pagination: { limit: 20 },
        }),
        feedService.getGroups({
          reference_point,
          radius: 5000,
          pagination: { limit: 10 },
        }),
        feedService.getTrending(),
      ]);

      if (
        usersRes.status === "fulfilled" &&
        isSuccess(usersRes.value) &&
        usersRes.value.data
      ) {
        setNearbyUsers(usersRes.value.data.items);
      }
      if (
        groupsRes.status === "fulfilled" &&
        isSuccess(groupsRes.value) &&
        groupsRes.value.data
      ) {
        setNearbyGroups(groupsRes.value.data.items);
      }
      if (
        trendingRes.status === "fulfilled" &&
        isSuccess(trendingRes.value) &&
        trendingRes.value.data
      ) {
        setTrending(trendingRes.value.data);
      }
    } catch (e) {
      setError("网络异常，请重试");
    } finally {
      setLoading(false);
    }
  }, [geoPoint, isAuthenticated]);

  useEffect(() => {
    load();
  }, [load]);

  return { nearbyUsers, nearbyGroups, trending, loading, error, refresh: load };
}
