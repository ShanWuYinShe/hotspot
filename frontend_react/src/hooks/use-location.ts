'use client';

import * as React from 'react';
import type { GeoPoint } from '@/types/api';

interface LocationState {
  location: GeoPoint | null;
  error: string | null;
  isLoading: boolean;
}

interface UseLocationOptions {
  enableHighAccuracy?: boolean;
  timeout?: number;
  maximumAge?: number;
}

/**
 * 获取用户位置的 Hook
 */
export function useLocation(options: UseLocationOptions = {}) {
  const {
    enableHighAccuracy = true,
    timeout = 10000,
    maximumAge = 60000,
  } = options;

  const [state, setState] = React.useState<LocationState>({
    location: null,
    error: null,
    isLoading: true,
  });

  const getLocation = React.useCallback(() => {
    if (!navigator.geolocation) {
      setState({
        location: null,
        error: '您的浏览器不支持定位功能',
        isLoading: false,
      });
      return;
    }

    setState((prev) => ({ ...prev, isLoading: true, error: null }));

    navigator.geolocation.getCurrentPosition(
      (position) => {
        setState({
          location: {
            latitude: position.coords.latitude,
            longitude: position.coords.longitude,
          },
          error: null,
          isLoading: false,
        });
      },
      (err) => {
        let errorMessage: string;
        switch (err.code) {
          case err.PERMISSION_DENIED:
            errorMessage = '请允许获取位置信息';
            break;
          case err.POSITION_UNAVAILABLE:
            errorMessage = '无法获取位置信息';
            break;
          case err.TIMEOUT:
            errorMessage = '获取位置超时';
            break;
          default:
            errorMessage = '获取位置失败';
        }
        setState({
          location: null,
          error: errorMessage,
          isLoading: false,
        });
      },
      {
        enableHighAccuracy,
        timeout,
        maximumAge,
      }
    );
  }, [enableHighAccuracy, timeout, maximumAge]);

  // 初始化时获取位置
  React.useEffect(() => {
    getLocation();
  }, [getLocation]);

  return {
    ...state,
    refresh: getLocation,
  };
}
