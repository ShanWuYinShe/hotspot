'use client';

import { useCallback } from 'react';

import { useLocationStore } from '@/stores/location';

export function useLocation() {
  const {
    currentLocation,
    referencePoint,
    isLoading,
    error,
    setLocation,
    setReferencePoint,
    setLoading,
    setError,
  } = useLocationStore();

  const requestLocation = useCallback(() => {
    if (!navigator.geolocation) {
      setError('geolocation_not_supported');
      return;
    }

    setLoading(true);
    navigator.geolocation.getCurrentPosition(
      (position) => {
        const { latitude, longitude } = position.coords;
        setReferencePoint({ latitude, longitude });
        setLocation({
          latitude,
          longitude,
          location_name: '',
        });
        setError(null);
        setLoading(false);
      },
      () => {
        setError('geolocation_denied');
        setLoading(false);
      },
      { enableHighAccuracy: true, timeout: 8000 },
    );
  }, [setError, setLoading, setLocation, setReferencePoint]);

  return {
    currentLocation,
    referencePoint,
    isLoading,
    error,
    requestLocation,
  };
}
