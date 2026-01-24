'use client';

import { useCallback } from 'react';
import { useLocationStore } from '@/stores/location';
import { useAuthStore } from '@/stores/auth';
import { useFeedbackStore } from '@/stores/feedback';
import { userService } from '@/lib/api/services/user';
import { isSuccess } from '@/lib/api/client';
import type { Location } from '@/types/api';

export function useLocation() {
  const {
    latitude,
    longitude,
    location_name,
    isLoading,
    error,
    lastUpdated,
    setLocation,
    setLoading,
    setError,
    getReferencePoint,
  } = useLocationStore();
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const { error: showError } = useFeedbackStore();

  // 获取当前位置（使用浏览器 Geolocation API）
  const getCurrentPosition = useCallback(async (): Promise<GeolocationPosition | null> => {
    if (!navigator.geolocation) {
      setError('浏览器不支持地理位置');
      return null;
    }

    return new Promise((resolve) => {
      navigator.geolocation.getCurrentPosition(
        (position) => resolve(position),
        (err) => {
          setError(err.message);
          resolve(null);
        },
        {
          enableHighAccuracy: true,
          timeout: 10000,
          maximumAge: 60000,
        }
      );
    });
  }, [setError]);

  // 更新位置到服务器
  const updateServerLocation = useCallback(
    async (location: Location) => {
      if (!isAuthenticated) return;
      try {
        const response = await userService.updateLocation(location);
        if (!isSuccess(response)) {
          console.error('Failed to update location on server:', response.message);
        }
      } catch (err) {
        console.error('Failed to update location on server:', err);
      }
    },
    [isAuthenticated]
  );

  // 刷新位置
  const refreshLocation = useCallback(
    async (updateServer = true) => {
      setLoading(true);
      setError(null);

      const position = await getCurrentPosition();
      
      if (position) {
        const location: Location = {
          latitude: position.coords.latitude,
          longitude: position.coords.longitude,
          location_name: '', // 可以通过逆地理编码获取
        };

        setLocation(location);

        if (updateServer) {
          await updateServerLocation(location);
        }
      } else {
        showError('获取位置失败', '请检查位置权限设置');
      }

      setLoading(false);
    },
    [getCurrentPosition, setLocation, setLoading, setError, updateServerLocation, showError]
  );

  // 手动设置位置
  const setManualLocation = useCallback(
    async (location: Location, updateServer = true) => {
      setLocation(location);
      
      if (updateServer) {
        await updateServerLocation(location);
      }
    },
    [setLocation, updateServerLocation]
  );

  return {
    // State
    latitude,
    longitude,
    location_name,
    isLoading,
    error,
    lastUpdated,
    hasLocation: latitude !== null && longitude !== null,
    currentLocation:
      latitude !== null && longitude !== null
        ? {
            latitude,
            longitude,
            location_name: location_name ?? '',
          }
        : null,
    referencePoint: getReferencePoint(),

    // Actions
    refreshLocation,
    setManualLocation,
    getReferencePoint,
  };
}
