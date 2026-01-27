import { useAuthStore } from '@/stores/auth';
import { useAdminStore } from '@/stores/admin';
import { AuthService, LoginRequest, RegisterRequest } from '@/lib/api/services/auth';
import { UserService } from '@/lib/api/services/user';
import { AdminRbacService } from '@/lib/api/services/admin/rbac';
import { UserInfo } from '@/types/api';

export function useAuth() {
  const store = useAuthStore();
  const adminStore = useAdminStore();

  const login = async (data: Omit<LoginRequest, 'client_type'>) => {
    const response = await AuthService.login(data);
    if (response.code === 0 && response.data) {
      const { access_token, user_id, nickname } = response.data;
      
      // Update store with minimal info first to establish session
      const fallbackUser: UserInfo = {
        user_id,
        nickname,
        user_type: 'normal',
      };
      store.setAuth(access_token, fallbackUser);

      // Fetch full profile to get avatar and other details
      try {
        const profileRes = await UserService.getProfile();
        if (profileRes.code === 0 && profileRes.data) {
            store.setAuth(access_token, {
              ...profileRes.data,
              user_type: 'normal',
            });
        }
      } catch (e) {
        // Ignore profile fetch error, session is established
        console.error('Failed to fetch profile after login', e);
      }
      try {
        const adminRes = await AdminRbacService.me();
        if (adminRes.code === 0 && adminRes.data) {
          adminStore.setPermissions(adminRes.data.permissions, adminRes.data.roles);
        } else {
          adminStore.clearPermissions();
        }
      } catch {
        adminStore.clearPermissions();
      }
    }
    return response;
  };

  const register = async (data: Omit<RegisterRequest, 'client_type'>) => {
    const response = await AuthService.register(data);
    if (response.code === 0 && response.data) {
      const { access_token, user_id, nickname } = response.data;
      const fallbackUser: UserInfo = { user_id, nickname, user_type: 'normal' };
      store.setAuth(access_token, fallbackUser);

      try {
        const profileRes = await UserService.getProfile();
        if (profileRes.code === 0 && profileRes.data) {
          store.setAuth(access_token, {
            ...profileRes.data,
            user_type: 'normal',
          });
        }
      } catch (e) {
        console.error('Failed to fetch profile after register', e);
      }
    }
    return response;
  };

  const guest = async (nickname: string) => {
    const response = await AuthService.guest({ nickname });
    if (response.code === 0 && response.data) {
      const { access_token, user_id } = response.data;
      store.setAuth(access_token, { user_id, nickname, user_type: 'temp' });
    }
    return response;
  };
  
  const logout = async () => {
    try {
      await AuthService.logout();
    } finally {
      store.clearAuth();
      adminStore.clearPermissions();
    }
  };

  const bootstrap = async () => {
    if (store.hasBootstrapped) {
      return;
    }
    store.setBootstrapped(true);
    const response = await AuthService.refresh();
    if (response.code === 0 && response.data) {
      const { access_token, user_id, nickname } = response.data;
      const fallbackUser: UserInfo = { user_id, nickname, user_type: 'normal' };
      store.setAuth(access_token, fallbackUser);

      try {
        const profileRes = await UserService.getProfile();
        if (profileRes.code === 0 && profileRes.data) {
          store.setAuth(access_token, {
            ...profileRes.data,
            user_type: 'normal',
          });
        }
      } catch (e) {
        console.error('Failed to fetch profile after refresh', e);
      }

      try {
        const adminRes = await AdminRbacService.me();
        if (adminRes.code === 0 && adminRes.data) {
          adminStore.setPermissions(adminRes.data.permissions, adminRes.data.roles);
        } else {
          adminStore.clearPermissions();
        }
      } catch {
        adminStore.clearPermissions();
      }
    }
  };

  const refreshAdminPermissions = async () => {
    const response = await AdminRbacService.me();
    if (response.code === 0 && response.data) {
      adminStore.setPermissions(response.data.permissions, response.data.roles);
    } else {
      adminStore.clearPermissions();
    }
    return response;
  };

  return {
    ...store,
    login,
    register,
    guest,
    logout,
    bootstrap,
    refreshAdminPermissions,
  };
}
