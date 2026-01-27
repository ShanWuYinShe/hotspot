import { apiClient } from '@/lib/api/client';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { AdminAdminItem, AdminPermissionItem, AdminPermissions, AdminRoleItem } from '@/types/api';

export const AdminRbacService = {
  me: () => apiClient.post<AdminPermissions>(ENDPOINTS.ADMIN.RBAC.ME),

  permissions: () =>
    apiClient.post<{ items: AdminPermissionItem[] }>(ENDPOINTS.ADMIN.RBAC.PERMISSIONS),

  roles: () => apiClient.post<{ items: AdminRoleItem[] }>(ENDPOINTS.ADMIN.RBAC.ROLES_LIST),

  createRole: (data: { name: string; description?: string; permissions: string[] }) =>
    apiClient.post<AdminRoleItem>(ENDPOINTS.ADMIN.RBAC.ROLES_CREATE, data),

  updateRole: (data: { role_id: string; name?: string; description?: string }) =>
    apiClient.post(ENDPOINTS.ADMIN.RBAC.ROLES_UPDATE, data),

  deleteRole: (data: { role_id: string }) =>
    apiClient.post(ENDPOINTS.ADMIN.RBAC.ROLES_DELETE, data),

  setRolePermissions: (data: { role_id: string; permissions: string[] }) =>
    apiClient.post(ENDPOINTS.ADMIN.RBAC.ROLES_SET_PERMISSIONS, data),

  admins: () => apiClient.post<{ items: AdminAdminItem[] }>(ENDPOINTS.ADMIN.RBAC.ADMINS_LIST),

  grantAdmin: (data: { user_id: number; role_ids?: string[]; permissions?: string[] }) =>
    apiClient.post(ENDPOINTS.ADMIN.RBAC.ADMINS_GRANT, data),

  revokeAdmin: (data: { user_id: number }) =>
    apiClient.post(ENDPOINTS.ADMIN.RBAC.ADMINS_REVOKE, data),
};
