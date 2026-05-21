import { useState, useCallback } from 'react';
import { usersService } from '@/lib/api/services/users';
import { isSuccess } from '@/lib/api/client';
import { useAuthStore } from '@/lib/stores/auth';
import type { GeoPoint } from '@/lib/types/api';

interface EditProfileResult {
  saving: boolean;
  saved: boolean;
  saveProfile: (params: {
    nickname?: string;
    bio?: string;
    tags?: string[];
    location_name?: string;
  }) => Promise<boolean>;
}

export function useEditProfile(): EditProfileResult {
  const setProfile = useAuthStore((s) => s.setProfile);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);

  const saveProfile = useCallback(async (params: {
    nickname?: string;
    bio?: string;
    tags?: string[];
    location_name?: string;
  }): Promise<boolean> => {
    if (saving) return false;
    setSaving(true);
    try {
      const res = await usersService.updateMyProfile(params);
      if (isSuccess(res)) {
        if (params.location_name) {
          const profile = useAuthStore.getState().profile;
          const point: GeoPoint = profile?.location?.point ?? { latitude: 0, longitude: 0 };
          await usersService.updateLocation({
            location: { point, location_name: params.location_name },
          });
        }
        const profileRes = await usersService.getMyProfile();
        if (isSuccess(profileRes) && profileRes.data) {
          setProfile(profileRes.data);
        }
        setSaved(true);
        return true;
      }
      return false;
    } catch {
      return false;
    } finally {
      setSaving(false);
    }
  }, [saving, setProfile]);

  return { saving, saved, saveProfile };
}
