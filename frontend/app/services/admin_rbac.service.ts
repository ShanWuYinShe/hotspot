import { BaseService } from './base.service'
import { ADMIN_RBAC } from '~/config/api'
import type {
  AdminMeData,
  AdminPermissionsData,
  AdminRolesData,
  AdminRoleItem,
  AdminAdminsData,
  AdminRbacGrantAdminRequest,
  AdminRbacRevokeAdminRequest,
  AdminRbacRoleCreateRequest,
  AdminRbacRoleUpdateRequest,
  AdminRbacRoleDeleteRequest,
  AdminRbacRoleSetPermissionsRequest
} from '~/types/api'

class AdminRbacService extends BaseService {
  async me() {
    return this.post<AdminMeData>(ADMIN_RBAC.ME, {})
  }

  async listPermissions() {
    return this.post<AdminPermissionsData>(ADMIN_RBAC.PERMISSIONS, {})
  }

  async listRoles() {
    return this.post<AdminRolesData>(ADMIN_RBAC.ROLES.LIST, {})
  }

  async createRole(request: AdminRbacRoleCreateRequest) {
    return this.post<AdminRoleItem>(ADMIN_RBAC.ROLES.CREATE, request)
  }

  async updateRole(request: AdminRbacRoleUpdateRequest) {
    return this.post<void>(ADMIN_RBAC.ROLES.UPDATE, request)
  }

  async deleteRole(request: AdminRbacRoleDeleteRequest) {
    return this.post<void>(ADMIN_RBAC.ROLES.DELETE, request)
  }

  async setRolePermissions(request: AdminRbacRoleSetPermissionsRequest) {
    return this.post<void>(ADMIN_RBAC.ROLES.SET_PERMISSIONS, request)
  }

  async listAdmins() {
    return this.post<AdminAdminsData>(ADMIN_RBAC.ADMINS.LIST, {})
  }

  async grantAdmin(request: AdminRbacGrantAdminRequest) {
    return this.post<void>(ADMIN_RBAC.ADMINS.GRANT, request)
  }

  async revokeAdmin(request: AdminRbacRevokeAdminRequest) {
    return this.post<void>(ADMIN_RBAC.ADMINS.REVOKE, request)
  }
}

export const adminRbacService = new AdminRbacService()
export function useAdminRbacService() {
  return adminRbacService
}
