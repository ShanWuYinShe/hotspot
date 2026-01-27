import { create } from 'zustand';

import { AdminRoleBasic } from '@/types/api';

interface AdminState {
  permissions: string[];
  roles: AdminRoleBasic[];
  setPermissions: (permissions: string[], roles: AdminRoleBasic[]) => void;
  clearPermissions: () => void;
}

export const useAdminStore = create<AdminState>((set) => ({
  permissions: [],
  roles: [],
  setPermissions: (permissions, roles) => set({ permissions, roles }),
  clearPermissions: () => set({ permissions: [], roles: [] }),
}));
