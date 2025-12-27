import { BaseService } from './base.service'
import { ADMIN_RBAC } from '~/config/api'
import type {
  AdminMeData,
  AdminPermissionsData,
  AdminRolesData,
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
    return this.post<AdminRolesData>(ADMIN_RBAC.ROLES_LIST, {})
  }

  async createRole(request: AdminRbacRoleCreateRequest) {
    return this.post<any>(ADMIN_RBAC.ROLES_CREATE, request)
  }

  async updateRole(request: AdminRbacRoleUpdateRequest) {
    return this.post<void>(ADMIN_RBAC.ROLES_UPDATE, request)
  }

  async deleteRole(request: AdminRbacRoleDeleteRequest) {
    return this.post<void>(ADMIN_RBAC.ROLES_DELETE, request)
  }

  async setRolePermissions(request: AdminRbacRoleSetPermissionsRequest) {
    return this.post<void>(ADMIN_RBAC.ROLES_SET_PERMISSIONS, request)
  }

  async listAdmins() {
    return this.post<AdminAdminsData>(ADMIN_RBAC.ADMINS_LIST, {})
  }

  async grantAdmin(request: AdminRbacGrantAdminRequest) {
    return this.post<void>(ADMIN_RBAC.ADMINS_GRANT, request)
  }

  async revokeAdmin(request: AdminRbacRevokeAdminRequest) {
    return this.post<void>(ADMIN_RBAC.ADMINS_REVOKE, request)
  }
}

export const adminRbacService = new AdminRbacService()
export function useAdminRbacService() {
  return adminRbacService
}

