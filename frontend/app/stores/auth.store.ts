import { defineStore } from "pinia";
import { useUserService } from "~/services/user.service";
import { DEFAULTS } from "~/config/defaults";
import type { AuthResponse, UserInfo } from "~/types";

// 用户基础信息类型（保持向后兼容）
export interface User {
  user_id: string; // 修正：使用 string 避免大整数精度问题
  nickname: string;
  avatar_url?: string;
  email?: string;
  is_temporary?: boolean;
}

// 安全且简洁的账户信息
interface SecureAccount {
  user_id: string; // 修正：使用 string 避免大整数精度问题
  nickname: string;
  avatar_url?: string;
  last_login: string;
  is_temporary?: boolean;
  // 安全存储：只保存加密后的refresh_token
  encrypted_token: string;
  // 设备密钥（用于解密）
  device_key: string;
}

/**
 * 安全且简洁的认证Store
 * 核心理念：安全第一，简洁体验
 * 保持与原有API的向后兼容性
 */
export const useAuthStore = defineStore(
  "auth",
  () => {
    // 当前用户状态
    const currentUser = ref<SecureAccount | null>(null);
    const token = ref<string | null>(null); // access_token，保持向后兼容命名
    const refreshToken = ref<string | null>(null); // 当前用户的refresh_token（解密后的）
    const expiresAt = ref<string | null>(null);
    const sessionActive = ref(false); // 本次会话是否已验证
    
    // 所有保存的账户
    const accounts = ref<SecureAccount[]>([]);
    
    // 向后兼容的计算属性
    const isAuthenticated = computed(() => !!token.value);
    const user = computed(() => currentUser.value ? {
      user_id: currentUser.value.user_id,
      nickname: currentUser.value.nickname,
      avatar_url: currentUser.value.avatar_url,
      is_temporary: currentUser.value.is_temporary,
      email: undefined, // TODO: 从后端获取邮箱信息
    } as User : null);
    
    // 兼容性属性
    const needsUnlock = computed(() => false); // 新系统不需要解锁概念
    const requiresPassphrase = computed(() => false); // 不再需要口令
    const pendingCredential = computed(() => null);
    const needsDeviceAuth = computed(() => !sessionActive.value && accounts.value.length > 1);

    const userService = useUserService();

    // 生成设备密钥（基于浏览器特征）- 使用简单缓存避免重复生成
    let cachedDeviceKey: string | null = null;
    async function generateDeviceKey(): Promise<string> {
      if (cachedDeviceKey) return cachedDeviceKey;
      
      const fingerprint = [
        navigator.userAgent,
        navigator.language,
        screen.width,
        screen.height,
        new Date().getTimezoneOffset(),
      ].join('|');
      
      const encoder = new TextEncoder();
      const data = encoder.encode(fingerprint + Date.now());
      const hashBuffer = await crypto.subtle.digest('SHA-256', data);
      const hashArray = Array.from(new Uint8Array(hashBuffer));
      cachedDeviceKey = hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
      return cachedDeviceKey;
    }

    // 设备验证（支持生物识别和PIN）
    async function authenticateDevice(): Promise<boolean> {
      try {
        // 优先使用 Web Authentication API
        if ('credentials' in navigator && window.PublicKeyCredential) {
          const challenge = crypto.getRandomValues(new Uint8Array(32));
          const credential = await navigator.credentials.create({
            publicKey: {
              challenge,
              rp: { name: "HotSpot" },
              user: {
                id: new Uint8Array(16),
                name: "device-auth",
                displayName: "设备验证"
              },
              pubKeyCredParams: [{ alg: -7, type: "public-key" }],
              authenticatorSelection: {
                userVerification: "required",
                authenticatorAttachment: "platform"
              },
              timeout: 10000,
            }
          }) as PublicKeyCredential;
          
          if (credential) {
            sessionActive.value = true;
            return true;
          }
        }
      } catch (e) {
        // 生物识别失败，降级到PIN
      }
      
      // 降级方案：4位PIN验证
      return await authenticateWithPin();
    }

    // PIN验证
    async function authenticateWithPin(): Promise<boolean> {
      const storedPinHash = localStorage.getItem('hotspot_device_pin');
      
      if (!storedPinHash) {
        // 首次设置PIN
        const newPin = prompt('请设置4位数字PIN以保护账户切换\n(建议使用好记的数字组合)');
        if (newPin && /^\d{4}$/.test(newPin)) {
          const hash = await hashPin(newPin);
          localStorage.setItem('hotspot_device_pin', hash);
          sessionActive.value = true;
          return true;
        }
        return false;
      } else {
        // 验证现有PIN
        const inputPin = prompt('请输入4位数字PIN以解锁账户切换');
        if (inputPin && /^\d{4}$/.test(inputPin)) {
          const inputHash = await hashPin(inputPin);
          if (inputHash === storedPinHash) {
            sessionActive.value = true;
            return true;
          }
        }
        return false;
      }
    }

    // PIN哈希
    async function hashPin(pin: string): Promise<string> {
      const encoder = new TextEncoder();
      const data = encoder.encode(pin + 'hotspot-device-salt-v1');
      const hashBuffer = await crypto.subtle.digest('SHA-256', data);
      const hashArray = Array.from(new Uint8Array(hashBuffer));
      return hashArray.map(b => b.toString(16).padStart(2, '0')).join('').slice(0, 16);
    }

    // 加密token
    async function encryptToken(refreshToken: string, deviceKey: string): Promise<string> {
      const encoder = new TextEncoder();
      const data = encoder.encode(refreshToken);
      const keyMaterial = await crypto.subtle.importKey(
        'raw',
        new TextEncoder().encode(deviceKey.slice(0, 32)),
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
    async function decryptToken(encryptedToken: string, deviceKey: string): Promise<string> {
      try {
        const combined = Uint8Array.from(atob(encryptedToken), c => c.charCodeAt(0));
        const iv = combined.slice(0, 12);
        const encrypted = combined.slice(12);
        
        const keyMaterial = await crypto.subtle.importKey(
          'raw',
          new TextEncoder().encode(deviceKey.slice(0, 32)),
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
        throw new Error('解密失败，可能是设备或数据损坏');
      }
    }

    // 保存认证数据（内部方法）
    async function saveAuthData(auth: AuthResponse) {
      const deviceKey = await generateDeviceKey();
      const encryptedToken = await encryptToken(auth.refresh_token, deviceKey);
      
      const account: SecureAccount = {
        user_id: auth.user_id,
        nickname: auth.nickname,
        avatar_url: undefined, // AuthResponse中不包含头像
        last_login: new Date().toISOString(),
        encrypted_token: encryptedToken,
        device_key: deviceKey,
        is_temporary: auth.is_temp || false, // 添加临时用户标识
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
      
      console.log('认证数据已保存:', { 
        user_id: auth.user_id, 
        nickname: auth.nickname, 
        is_temporary: auth.is_temp 
      });
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
        currentUser.value.is_temporary = false;
      }
      return auth;
    }

    // 🚀 安全的账户切换（新的核心功能）
    async function switchToAccount(userId: number | string) {
      // 如果有多个账户且本次会话未验证，需要设备验证
      if (needsDeviceAuth.value) {
        const authenticated = await authenticateDevice();
        if (!authenticated) {
          throw new Error("设备验证失败，无法切换账户");
        }
      }

      const account = accounts.value.find(a => a.user_id === String(userId));
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
        const index = accounts.value.findIndex(a => a.user_id === String(userId));
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
        if (token.value) {
          return await userService.checkAuthStatus();
        }
        
        if (refreshToken.value) {
          await refreshAuth();
          return await userService.checkAuthStatus();
        }
        
        throw new Error("需要登录");
      } catch (error) {
        // 仅对明确未授权的错误清理会话；其他错误不清理，避免误登出
        const status = (error as any)?.status ?? (error as any)?.data?.code;
        if (status === 401) {
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
    function removeAccount(userId: number | string) {
      const userIdStr = String(userId);
      accounts.value = accounts.value.filter(a => a.user_id !== userIdStr);
      
      // 如果移除的是当前账户，清空状态
      if (currentUser.value?.user_id === userIdStr) {
        clearAuth();
      }
    }

    // 删除账户（向后兼容）
    function deleteAccount(userId: string | number) {
      removeAccount(userId);
    }

    // 向后兼容的方法存根
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

    // 向后兼容方法（简化实现）
    const getAccountByUserId = (userId: string) => 
      accounts.value.find(a => a.user_id === userId) || null;

    const addOrUpdateAccount = (authData: any) => {
      // 新系统中通过saveAuthData自动处理
    };

    const cleanupInvalidAccounts = () => {
      accounts.value = accounts.value.filter(a => a.user_id && a.nickname);
    };

    // 不再需要的复杂方法（返回成功以保持兼容性）
    const savePassphrase = async (passphrase: string) => Promise.resolve();
    const unlockWithPassphrase = async (passphrase: string) => Promise.resolve();
    const switchUser = (userId: string) => switchToAccount(userId);
    const saveEncryptedCredential = async () => Promise.resolve();
    const unlockCredential = async () => Promise.resolve(null);

    return {
      // 状态（保持向后兼容）
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
      
      // 账户管理
      accounts: computed(() => accounts.value),
      getAccountByUserId,

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
      
      // 向后兼容方法
      savePassphrase,
      unlockWithPassphrase,
      switchUser,
      deleteAccount,
      addOrUpdateAccount,
      removeAccount,
      cleanupInvalidAccounts,
      saveEncryptedCredential,
      unlockCredential,
    };
  },
  {
    persist: {
      // 持久化当前会话必要字段，避免刷新丢失登录态
      pick: ["currentUser", "accounts", "token", "refreshToken", "expiresAt", "sessionActive"],
    },
  }
);