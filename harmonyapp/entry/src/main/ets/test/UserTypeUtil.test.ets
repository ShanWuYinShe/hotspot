/**
 * UserTypeUtil 测试用例
 */

import { describe, it, expect, TestRunner, TestUtils } from './TestFramework';
import { UserTypeUtil, UserType } from '../common/utils/UserTypeUtil';

// 用户类型工具测试套件
export const userTypeUtilTests = describe('UserTypeUtil', [
  it('should correctly identify temp user', async () => {
    const tempUserId = TestUtils.generateTestUserId(1); // 临时用户
    expect(UserTypeUtil.isTempUser(tempUserId)).toBe(true);
    expect(UserTypeUtil.isRegularUser(tempUserId)).toBe(false);
    expect(UserTypeUtil.isSystemUser(tempUserId)).toBe(false);
  }),

  it('should correctly identify regular user', async () => {
    const regularUserId = TestUtils.generateTestUserId(2); // 普通用户
    expect(UserTypeUtil.isTempUser(regularUserId)).toBe(false);
    expect(UserTypeUtil.isRegularUser(regularUserId)).toBe(true);
    expect(UserTypeUtil.isSystemUser(regularUserId)).toBe(false);
  }),

  it('should correctly identify system user', async () => {
    const systemUserId = TestUtils.generateTestUserId(3); // 系统用户
    expect(UserTypeUtil.isTempUser(systemUserId)).toBe(false);
    expect(UserTypeUtil.isRegularUser(systemUserId)).toBe(false);
    expect(UserTypeUtil.isSystemUser(systemUserId)).toBe(true);
  }),

  it('should get correct user type', async () => {
    const tempUserId = TestUtils.generateTestUserId(1);
    const regularUserId = TestUtils.generateTestUserId(2);
    const systemUserId = TestUtils.generateTestUserId(3);

    expect(UserTypeUtil.getUserType(tempUserId)).toBe(UserType.TEMP_USER);
    expect(UserTypeUtil.getUserType(regularUserId)).toBe(UserType.REGULAR_USER);
    expect(UserTypeUtil.getUserType(systemUserId)).toBe(UserType.SYSTEM_USER);
  }),

  it('should get correct user type name', async () => {
    const tempUserId = TestUtils.generateTestUserId(1);
    const regularUserId = TestUtils.generateTestUserId(2);
    const systemUserId = TestUtils.generateTestUserId(3);

    expect(UserTypeUtil.getUserTypeName(tempUserId)).toBe('临时用户');
    expect(UserTypeUtil.getUserTypeName(regularUserId)).toBe('普通用户');
    expect(UserTypeUtil.getUserTypeName(systemUserId)).toBe('系统用户');
  }),

  it('should validate user ID format', async () => {
    const validUserId = TestUtils.generateTestUserId(2);
    expect(UserTypeUtil.validateUserId(validUserId)).toBe(true);
    
    expect(UserTypeUtil.validateUserId('')).toBe(false);
    expect(UserTypeUtil.validateUserId('invalid')).toBe(false);
    expect(UserTypeUtil.validateUserId('0')).toBe(false);
  }),

  it('should generate valid user ID', async () => {
    const userId = UserTypeUtil.generateUserId(UserType.REGULAR_USER, 12345n);
    expect(UserTypeUtil.validateUserId(userId)).toBe(true);
    expect(UserTypeUtil.getUserType(userId)).toBe(UserType.REGULAR_USER);
    expect(UserTypeUtil.getUserSequence(userId)).toBe(12345n);
  }),

  it('should batch get user types', async () => {
    const userIds = [
      TestUtils.generateTestUserId(1),
      TestUtils.generateTestUserId(2),
      TestUtils.generateTestUserId(3)
    ];

    const typeMap = UserTypeUtil.batchGetUserTypes(userIds);
    expect(typeMap.size).toBe(3);
    expect(typeMap.get(userIds[0])).toBe(UserType.TEMP_USER);
    expect(typeMap.get(userIds[1])).toBe(UserType.REGULAR_USER);
    expect(typeMap.get(userIds[2])).toBe(UserType.SYSTEM_USER);
  }),

  it('should filter users by type', async () => {
    const userIds = [
      TestUtils.generateTestUserId(1),
      TestUtils.generateTestUserId(2),
      TestUtils.generateTestUserId(1),
      TestUtils.generateTestUserId(3)
    ];

    const tempUsers = UserTypeUtil.getTempUsers(userIds);
    const regularUsers = UserTypeUtil.getRegularUsers(userIds);
    const systemUsers = UserTypeUtil.getSystemUsers(userIds);

    expect(tempUsers.length).toBe(2);
    expect(regularUsers.length).toBe(1);
    expect(systemUsers.length).toBe(1);
  }),

  it('should check user permissions correctly', async () => {
    const tempUserId = TestUtils.generateTestUserId(1);
    const regularUserId = TestUtils.generateTestUserId(2);
    const systemUserId = TestUtils.generateTestUserId(3);

    // 临时用户权限
    expect(UserTypeUtil.hasPermission(tempUserId, 'view_public_content')).toBe(true);
    expect(UserTypeUtil.hasPermission(tempUserId, 'create_group')).toBe(false);
    expect(UserTypeUtil.hasPermission(tempUserId, 'manage_users')).toBe(false);

    // 普通用户权限
    expect(UserTypeUtil.hasPermission(regularUserId, 'view_public_content')).toBe(true);
    expect(UserTypeUtil.hasPermission(regularUserId, 'create_group')).toBe(true);
    expect(UserTypeUtil.hasPermission(regularUserId, 'manage_users')).toBe(false);

    // 系统用户权限
    expect(UserTypeUtil.hasPermission(systemUserId, 'view_public_content')).toBe(true);
    expect(UserTypeUtil.hasPermission(systemUserId, 'create_group')).toBe(true);
    expect(UserTypeUtil.hasPermission(systemUserId, 'manage_users')).toBe(true);
  })
]);

// 运行测试的函数
export async function runUserTypeUtilTests(): Promise<void> {
  const runner = new TestRunner();
  await runner.runSuite(userTypeUtilTests);
}

