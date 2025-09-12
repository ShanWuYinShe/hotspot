import { useUserService } from "~/services/user.service";
import { DEFAULTS } from "~/config/defaults";
import type { AuthResponse, UserInfo, User } from "~/types";

// 安全且简洁的账户信息
interface SecureAccount {
  user_id: string; // 修正：使用 string 避免大整数精度问题
  nickname: string;
  avatar_url?: string;
  last_login: string;
  is_temp?: boolean; // 🔥 修复：统一使用 is_temp 字段，与 AuthResponse 保持一致
  // 安全存储：只保存加密后的refresh_token
  encrypted_token: string;
  // 设备密钥（用于解密）
  device_key: string;
}

// PIN 输入模态框状态
interface PinModalState {
  isOpen: boolean;
  isSetup: boolean;
  resolve?: (pin: string) => void;
  reject?: (error: Error) => void;
}

/**
 * 安全且简洁的认证Store
 * 核心理念：安全第一，简洁体验
 */
export const useAuthStore = defineStore(
  "auth",
  () => {
    // 当前用户状态
    const currentUser = ref<SecureAccount | null>(null);
    const token = ref<string | null>(null); // access_token
    const refreshToken = ref<string | null>(null); // 当前用户的refresh_token（解密后的）
    const expiresAt = ref<string | null>(null);
    const sessionActive = ref(false); // 本次会话是否已验证
    
    // 所有保存的账户
    const accounts = ref<SecureAccount[]>([]);
    
    // PIN 模态框状态
    const pinModalState = ref<PinModalState>({
      isOpen: false,
      isSetup: false,
    });
    
    // 计算属性
    const isAuthenticated = computed(() => !!token.value);
    const user = computed(() => currentUser.value ? {
      user_id: currentUser.value.user_id,
      nickname: currentUser.value.nickname,
      avatar_url: currentUser.value.avatar_url,
      is_temp: currentUser.value.is_temp, // 🔥 修复：统一使用 is_temp 字段
    } as User : null);
    
    // 设备认证状态
    const needsDeviceAuth = computed(() => !sessionActive.value && accounts.value.length > 1);
    
    // 兼容性属性
    const needsUnlock = computed(() => false); // 新系统不需要解锁概念
    const requiresPassphrase = computed(() => false); // 不再需要口令
    const pendingCredential = computed(() => null);

    const userService = useUserService();
    
    // 导入改进的服务
    const deviceKey = useDeviceKey();
    const pinSecurity = usePinSecurity();
    const webAuthn = useWebAuthn();

    // 显示 PIN 输入模态框
    const showPinModal = (isSetup: boolean = false): Promise<string> => {
      return new Promise((resolve, reject) => {
        pinModalState.value = {
          isOpen: true,
          isSetup,
          resolve,
          reject,
        };
      });
    };

    // 处理 PIN 模态框确认
    const handlePinConfirm = async (pin: string) => {
      try {
        if (pinModalState.value.isSetup) {
          await pinSecurity.setPin(pin);
        } else {
          const isValid = await pinSecurity.verifyPin(pin);
          if (!isValid) {
            throw new Error('PIN验证失败');
          }
        }
        
        pinModalState.value.resolve?.(pin);
        pinModalState.value.isOpen = false;
      } catch (error) {
        const errorMessage = error instanceof Error ? error.message : 'PIN处理失败';
        pinModalState.value.reject?.(new Error(errorMessage));
      }
    };

    // 处理 PIN 模态框取消
    const handlePinCancel = () => {
      pinModalState.value.reject?.(new Error('用户取消了PIN输入'));
      pinModalState.value.isOpen = false;
    };

    // 设备验证（支持生物识别和PIN）
    async function authenticateDevice(): Promise<boolean> {
      try {
        // 优先使用 WebAuthn 生物识别
        if (webAuthn.isSupported.value) {
          const platformSupported = await webAuthn.isPlatformAuthenticatorSupported();
          
          if (platformSupported && webAuthn.hasRegisteredCredentials()) {
            try {
              const success = await webAuthn.authenticateWithCredential();
              if (success) {
                sessionActive.value = true;
                return true;
              }
            } catch (error) {
              if (import.meta.dev) {
                console.log('WebAuthn authentication failed, falling back to PIN:', error);
              }
              // 继续降级到 PIN
            }
          } else if (platformSupported && !webAuthn.hasRegisteredCredentials()) {
            // 尝试注册生物识别作为备选方案
            try {
              const userIdentifier = currentUser.value?.user_id || 'device_auth';
              await webAuthn.registerCredential(userIdentifier);
              const success = await webAuthn.authenticateWithCredential();
              if (success) {
                sessionActive.value = true;
                return true;
              }
            } catch (error) {
              if (import.meta.dev) {
                console.log('WebAuthn registration failed, falling back to PIN:', error);
              }
              // 继续降级到 PIN
            }
          }
        }
      } catch (error) {
        if (import.meta.dev) {
          console.log('WebAuthn not available, using PIN:', error);
        }
      }
      
      // 降级方案：PIN验证
      return await authenticateWithPin();
    }

    // PIN验证
    async function authenticateWithPin(): Promise<boolean> {
      try {
        if (!pinSecurity.hasPin()) {
          // 首次设置PIN
          const pin = await showPinModal(true);
          sessionActive.value = true;
          return true;
        } else {
          // 验证现有PIN
          const pin = await showPinModal(false);
          sessionActive.value = true;
          return true;
        }
      } catch (error) {
        if (import.meta.dev) {
          console.log('PIN authentication failed:', error);
        }
        return false;
      }
    }

    // 加密token
    async function encryptToken(refreshToken: string, deviceKeyString: string): Promise<string> {
      const encoder = new TextEncoder();
      const data = encoder.encode(refreshToken);
      const keyMaterial = await crypto.subtle.importKey(
        'raw',
        new TextEncoder().encode(deviceKeyString.slice(0, 32)),
        { name: 'AES-GCM' },
        false,
        ['encrypt']
      );
      
      const iv = crypto.getRandomValues(new Uint8Array(12));
      const encrypted = await crypto.subtle.encrypt(
        { name: 'AES-GCM', iv },
        keyMaterial,
        data
      );
      
      const combined = new Uint8Array(iv.length + encrypted.byteLength);
      combined.set(iv);
      combined.set(new Uint8Array(encrypted), iv.length);
      return btoa(String.fromCharCode(...combined));
    }

    // 解密token
    async function decryptToken(encryptedToken: string, deviceKeyString: string): Promise<string> {
      try {
        const combined = Uint8Array.from(atob(encryptedToken), c => c.charCodeAt(0));
        const iv = combined.slice(0, 12);
        const encrypted = combined.slice(12);
        
        const keyMaterial = await crypto.subtle.importKey(
          'raw',
          new TextEncoder().encode(deviceKeyString.slice(0, 32)),
          { name: 'AES-GCM' },
          false,
          ['decrypt']
        );
        
        const decrypted = await crypto.subtle.decrypt(
          { name: 'AES-GCM', iv },
          keyMaterial,
          encrypted
        );
        
        return new TextDecoder().decode(decrypted);
      } catch (error) {
        console.error('Token decryption failed:', error);
        throw new Error('解密失败，可能是设备或数据损坏');
      }
    }

    // 保存认证数据（内部方法）
    async function saveAuthData(auth: AuthResponse) {
      try {
        const deviceKeyString = await deviceKey.getEncryptionKey('auth_token');
        const encryptedToken = await encryptToken(auth.refresh_token, deviceKeyString);
        
        const account: SecureAccount = {
          user_id: auth.user_id,
          nickname: auth.nickname,
          avatar_url: undefined, // AuthResponse中不包含头像
          last_login: new Date().toISOString(),
          encrypted_token: encryptedToken,
          device_key: deviceKeyString,
          is_temp: auth.is_temp || false, // 🔥 修复：统一使用 is_temp 字段
        };

        // 更新当前状态
        currentUser.value = account;
        token.value = auth.access_token;
        refreshToken.value = auth.refresh_token;
        expiresAt.value = auth.expires_at || null;
        
        // 保存到账户历史（去重、限制数量）
        accounts.value = accounts.value.filter(a => a.user_id !== auth.user_id);
        accounts.value.unshift(account);
        if (accounts.value.length > 5) {
          accounts.value = accounts.value.slice(0, 5);
        }
        
        sessionActive.value = true; // 刚登录无需重新验证
        
        if (import.meta.dev) {
          console.log('认证数据已保存:', { 
            user_id: auth.user_id, 
            nickname: auth.nickname, 
            is_temp: auth.is_temp // 🔥 修复：统一使用 is_temp 字段 
          });
        }
      } catch (error) {
        console.error('Failed to save auth data:', error);
        throw new Error('保存认证数据失败');
      }
    }

    // 登录
    async function login(loginId: string, password: string) {
      const auth = await userService.login(loginId, password);
      await saveAuthData(auth);
      return auth;
    }

    // 注册
    async function register(loginId: string, nickname: string, password: string) {
      const auth = await userService.register(loginId, nickname, password);
      await saveAuthData(auth);
      return auth;
    }

    // 创建临时用户
    async function createTempUser(nickname?: string) {
      const auth = await userService.createTempUser(nickname);
      await saveAuthData(auth);
      return auth;
    }

    // 升级账户
    async function upgradeAccount(loginId: string, password: string) {
      const auth = await userService.upgradeAccount(loginId, password);
      await saveAuthData(auth);
      // 更新当前用户的临时状态
      if (currentUser.value) {
        currentUser.value.is_temp = false; // 🔥 修复：统一使用 is_temp 字段
      }
      return auth;
    }

    // 🚀 安全的账户切换（新的核心功能）
    async function switchToAccount(userId: string) {
      // 如果有多个账户且本次会话未验证，需要设备验证
      if (needsDeviceAuth.value) {
        const authenticated = await authenticateDevice();
        if (!authenticated) {
          throw new Error("设备验证失败，无法切换账户");
        }
      }

      const account = accounts.value.find(a => a.user_id === userId);
      if (!account) throw new Error("账户不存在");

      try {
        // 解密并使用refresh_token
        const decryptedRefreshToken = await decryptToken(account.encrypted_token, account.device_key);
        const auth = await userService.refreshToken(decryptedRefreshToken);
        
        // 更新当前状态
        currentUser.value = account;
        token.value = auth.access_token;
        refreshToken.value = auth.refresh_token;
        expiresAt.value = auth.expires_at || null;
        
        // 更新账户的最后登录时间并重新加密新的refresh_token
        account.last_login = new Date().toISOString();
        account.encrypted_token = await encryptToken(auth.refresh_token, account.device_key);
        
        // 移动到最前面
        const index = accounts.value.findIndex(a => a.user_id === userId);
        if (index > 0) {
          const [movedAccount] = accounts.value.splice(index, 1);
          if (movedAccount) {
            accounts.value.unshift(movedAccount);
          }
        }
        
        return auth;
      } catch (error) {
        // token过期或解密失败，移除该账户
        removeAccount(userId);
        throw new Error("账户已过期或损坏，已自动移除");
      }
    }

    // 刷新令牌
    async function refreshAuth() {
      if (!refreshToken.value) {
        throw new Error("没有刷新令牌");
      }

      try {
        const auth = await userService.refreshToken(refreshToken.value);
        token.value = auth.access_token;
        refreshToken.value = auth.refresh_token;
        expiresAt.value = auth.expires_at || null;

        // 更新当前账户的加密token
        if (currentUser.value) {
          currentUser.value.encrypted_token = await encryptToken(
            auth.refresh_token, 
            currentUser.value.device_key
          );
        }

        return auth;
      } catch (error) {
        // 刷新失败：不立即清空当前账户列表，避免误删；仅清除会话token并保留可切换的账户
        token.value = null;
        refreshToken.value = null;
        expiresAt.value = null;
        sessionActive.value = false;
        throw error;
      }
    }

    // 检查认证状态
    async function checkAuth() {
      try {
        // 优先通过后端验证 token 有效性
        if (token.value) {
          try {
            const authStatus = await userService.checkAuthStatus();
            return authStatus;
          } catch (error) {
            // token 无效，尝试刷新
            const status = (error as { status?: number; data?: { code?: number } })?.status ?? 
                          (error as { status?: number; data?: { code?: number } })?.data?.code;
            
            if (status === 401 || status === 1002) {
              // 尝试刷新 token
              if (refreshToken.value) {
                await refreshAuth();
                return await userService.checkAuthStatus();
              } else {
                throw new Error("认证已过期，需要重新登录");
              }
            } else {
              throw error;
            }
          }
        }
        
        // 没有 access_token，尝试使用 refresh_token
        if (refreshToken.value) {
          await refreshAuth();
          return await userService.checkAuthStatus();
        }
        
        throw new Error("需要登录");
      } catch (error) {
        // 仅对明确未授权的错误清理会话；其他错误不清理，避免误登出
        const status = (error as { status?: number; data?: { code?: number } })?.status ?? 
                      (error as { status?: number; data?: { code?: number } })?.data?.code;
        
        if (status === 401 || status === 1002) {
          if (import.meta.dev) {
            console.log('Authentication expired, clearing auth state');
          }
          clearAuth();
        }
        
        throw error;
      }
    }

    // 清理认证状态
    function clearAuth() {
      currentUser.value = null;
      token.value = null;
      refreshToken.value = null;
      expiresAt.value = null;
      sessionActive.value = false;
    }

    // 登出
    async function logout() {
      clearAuth();
    }

    // 移除账户
    function removeAccount(userId: string) {
      accounts.value = accounts.value.filter(a => a.user_id !== userId);
      
      // 如果移除的是当前账户，清空状态
      if (currentUser.value?.user_id === userId) {
        clearAuth();
      }
    }

    // 删除账户
    function deleteAccount(userId: string) {
      removeAccount(userId);
    }

    // 设置会话
    function setSession(authData: AuthResponse, persistAccount: boolean = true) {
      if (persistAccount) {
        saveAuthData(authData);
      } else {
        token.value = authData.access_token;
        refreshToken.value = authData.refresh_token;
        expiresAt.value = authData.expires_at || null;
      }
    }

    async function setAuthData(authData: AuthResponse) {
      await saveAuthData(authData);
    }

    function updateUser(userData: Partial<User>) {
      if (currentUser.value) {
        Object.assign(currentUser.value, userData);
      }
    }

    const switchUser = (userId: string) => switchToAccount(userId);
    
    // 兼容性方法
    const getAccountByUserId = (userId: string) => 
      accounts.value.find(a => a.user_id === userId) || null;

    return {
      // 状态
      user,
      token,
      refreshToken,
      expiresAt,
      isAuthenticated,
      requiresPassphrase,
      needsUnlock,
      pendingCredential,

      // 新增状态
      needsDeviceAuth,
      sessionActive: computed(() => sessionActive.value),
      
      // PIN 模态框状态
      pinModalState: computed(() => pinModalState.value),
      
      // 账户管理
      accounts: computed(() => accounts.value),

      // 认证方法
      login,
      register,
      createTempUser,
      upgradeAccount,
      refreshAuth,
      checkAuth,
      logout,
      setSession,
      setAuthData,
      clearAuth,
      updateUser,
      
      // 新的核心功能
      switchToAccount,
      authenticateDevice,
      
      // PIN 模态框方法
      showPinModal,
      handlePinConfirm,
      handlePinCancel,
      
      // 账户管理方法
      switchUser,
      deleteAccount,
      removeAccount,
      getAccountByUserId,
    };
  },
  {
    persist: {
      // 持久化当前会话必要字段，避免刷新丢失登录态
      pick: ["currentUser", "accounts", "token", "refreshToken", "expiresAt", "sessionActive"],
    },
  }
);