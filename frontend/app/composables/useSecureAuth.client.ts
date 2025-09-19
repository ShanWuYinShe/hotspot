/**
 * 安全认证服务 - 实现密码安全机制
 * 根据需求文档实现 SHA256 + nonce + timestamp 的安全认证流程
 */

import { computePasswordHash, isWebCryptoSupported } from './useCrypto.client';
import { AUTH_ENDPOINTS, USER_ENDPOINTS, buildAuthUrl, buildUserUrl } from '~/config/api';
import type { 
  GetNonceRequest, 
  NonceResponse, 
  LoginRequest, 
  RegisterRequest, 
  SetPasswordRequest,
  UpgradeAccountRequest,
  AuthResponse,
  BasicSuccess
} from '~/types';

export const useSecureAuth = () => {
  const api = useApi();

  /**
   * 验证浏览器支持
   */
  const checkCryptoSupport = (): void => {
    if (!isWebCryptoSupported()) {
      throw new Error('您的浏览器不支持 Web Crypto API，无法进行安全认证');
    }
  };

  /**
   * 获取认证随机数
   * @param loginId 登录ID（邮箱/手机号）
   * @returns nonce 信息
   */
  const getNonce = async (loginId: string): Promise<NonceResponse> => {
    checkCryptoSupport();
    
    const request: GetNonceRequest = { login_id: loginId };
    const response = await api.postPublic(buildAuthUrl(AUTH_ENDPOINTS.PUBLIC.GET_NONCE), request);
    
    return extractApiData(response);
  };

  /**
   * 安全登录
   * @param loginId 登录ID
   * @param password 原始密码
   * @returns 认证响应
   */
  const secureLogin = async (loginId: string, password: string): Promise<AuthResponse> => {
    checkCryptoSupport();
    
    // 1. 获取 nonce
    const nonceData = await getNonce(loginId);
    
    // 2. 计算密码哈希
    const passwordHash = await computePasswordHash(
      password,
      nonceData.nonce,
      nonceData.timestamp
    );
    
    // 3. 发送安全登录请求
    const request: LoginRequest = {
      login_id: loginId,
      password_hash: passwordHash,
      nonce_id: nonceData.nonce_id,
      timestamp: nonceData.timestamp
    };
    
    const response = await api.postPublic(buildAuthUrl(AUTH_ENDPOINTS.PUBLIC.LOGIN), request);
    return extractApiData(response);
  };

  /**
   * 安全注册
   * @param loginId 登录ID
   * @param nickname 昵称
   * @param password 原始密码
   * @returns 认证响应
   */
  const secureRegister = async (
    loginId: string, 
    nickname: string, 
    password: string
  ): Promise<AuthResponse> => {
    checkCryptoSupport();
    
    // 1. 获取 nonce
    const nonceData = await getNonce(loginId);
    
    // 2. 计算密码哈希
    const passwordHash = await computePasswordHash(
      password,
      nonceData.nonce,
      nonceData.timestamp
    );
    
    // 3. 发送安全注册请求
    const request: RegisterRequest = {
      login_id: loginId,
      nickname: nickname,
      password_hash: passwordHash,
      nonce_id: nonceData.nonce_id,
      timestamp: nonceData.timestamp
    };
    
    const response = await api.postPublic(buildAuthUrl(AUTH_ENDPOINTS.PUBLIC.REGISTER), request);
    return extractApiData(response);
  };

  /**
   * 安全设置密码
   * @param newPassword 新密码
   * @param oldPassword 旧密码（可选）
   * @param userId 用户ID（用于哈希计算）
   * @returns 操作结果
   */
  const secureSetPassword = async (
    newPassword: string,
    userId: string,
    oldPassword?: string
  ): Promise<BasicSuccess> => {
    checkCryptoSupport();
    
    // 1. 获取 nonce（使用 user_id 作为标识）
    const nonceData = await getNonce(userId);
    
    // 2. 计算新密码哈希
    const newPasswordHash = await computePasswordHash(
      newPassword,
      nonceData.nonce,
      nonceData.timestamp
    );
    
    // 3. 计算旧密码哈希（如果提供）
    let oldPasswordHash: string | undefined;
    if (oldPassword) {
      oldPasswordHash = await computePasswordHash(
        oldPassword,
        nonceData.nonce,
        nonceData.timestamp
      );
    }
    
    // 4. 发送安全设置密码请求
    const request: SetPasswordRequest = {
      new_password_hash: newPasswordHash,
      old_password_hash: oldPasswordHash,
      nonce_id: nonceData.nonce_id,
      timestamp: nonceData.timestamp
    };
    
    const response = await api.post(buildAuthUrl(AUTH_ENDPOINTS.PROTECTED.SET_PASSWORD, true), request, { auth: true });
    return extractApiData(response);
  };

  /**
   * 安全账户升级
   * @param loginId 登录ID
   * @param password 密码
   * @returns 认证响应
   */
  const secureUpgradeAccount = async (
    loginId: string, 
    password: string
  ): Promise<AuthResponse> => {
    checkCryptoSupport();
    
    // 1. 获取 nonce
    const nonceData = await getNonce(loginId);
    
    // 2. 计算密码哈希
    const passwordHash = await computePasswordHash(
      password,
      nonceData.nonce,
      nonceData.timestamp
    );
    
    // 3. 发送安全升级请求
    const request: UpgradeAccountRequest = {
      login_id: loginId,
      password_hash: passwordHash,
      nonce_id: nonceData.nonce_id,
      timestamp: nonceData.timestamp
    };
    
    const response = await api.post(buildUserUrl(USER_ENDPOINTS.PROTECTED.UPGRADE_ACCOUNT), request, { auth: true });
    return extractApiData(response);
  };

  return {
    // 核心安全认证方法
    secureLogin,
    secureRegister,
    secureSetPassword,
    secureUpgradeAccount,
    
    // 工具方法
    getNonce,
    checkCryptoSupport,
  };
};
