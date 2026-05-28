import { create } from "zustand";
import type { AuthData, MyProfile } from "@/lib/types/api";

const STORAGE_KEY = "hotspot_auth";

interface AccountInfo {
  user_id: string;
  nickname: string;
  avatar_url?: string;
  access_token_expires_at: string;
  is_locked?: boolean;
}

interface AuthState {
  accounts: AccountInfo[];
  currentUserId: string | null;
  profile: MyProfile | null;
  sessionValid: boolean;
  isRestoring: boolean;

  currentAccount: AccountInfo | undefined;
  isAuthenticated: boolean;
  availableAccounts: AccountInfo[];
  lockedAccounts: AccountInfo[];

  // Actions
  login: (authData: AuthData) => void;
  logout: () => void;
  logoutAll: () => void;
  setProfile: (profile: MyProfile | null) => void;
  invalidateSession: () => void;
  updateExpiresAt: (expiresAt: string) => void;
  startRestoring: () => void;
  finishRestoring: () => void;
  switchAccount: (userId: string) => void;
  lockAccount: (userId: string) => void;
  unlockAccount: (userId: string) => void;
  removeAccount: (userId: string) => void;
  isTokenExpiringSoon: (thresholdMinutes?: number) => boolean;
}

function loadFromStorage(): {
  accounts: AccountInfo[];
  currentUserId: string | null;
} {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (!stored) return { accounts: [], currentUserId: null };
    const parsed = JSON.parse(stored);
    if (!parsed || !Array.isArray(parsed.accounts))
      return { accounts: [], currentUserId: null };
    return {
      accounts: parsed.accounts,
      currentUserId: parsed.currentUserId ?? null,
    };
  } catch {
    return { accounts: [], currentUserId: null };
  }
}

function saveToStorage(accounts: AccountInfo[], currentUserId: string | null) {
  try {
    localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({ accounts, currentUserId }),
    );
  } catch {
    // ignore
  }
}

const stored = loadFromStorage();

function deriveState(state: {
  accounts: AccountInfo[];
  currentUserId: string | null;
  sessionValid: boolean;
}) {
  const currentAccount = state.accounts.find(
    (a) => a.user_id === state.currentUserId && !a.is_locked,
  );
  return {
    currentAccount,
    isAuthenticated: !!currentAccount && state.sessionValid,
    availableAccounts: state.accounts.filter((a) => !a.is_locked),
    lockedAccounts: state.accounts.filter((a) => a.is_locked),
  };
}

const initialDerived = deriveState({
  accounts: stored.accounts,
  currentUserId: stored.currentUserId,
  sessionValid: false,
});

function syncSet(
  set: (
    partial: Partial<AuthState> | ((s: AuthState) => Partial<AuthState>),
  ) => void,
) {
  return (
    partial: Partial<AuthState> | ((s: AuthState) => Partial<AuthState>),
  ) => {
    set((state) => {
      const next = typeof partial === "function" ? partial(state) : partial;
      return { ...next, ...deriveState({ ...state, ...next }) };
    });
  };
}

export const useAuthStore = create<AuthState>((origSet, get) => {
  const set = syncSet(origSet);
  return {
    accounts: stored.accounts,
    currentUserId: stored.currentUserId,
    profile: null,
    sessionValid: false,
    isRestoring: !!stored.currentUserId,

    currentAccount: initialDerived.currentAccount,
    isAuthenticated: initialDerived.isAuthenticated,
    availableAccounts: initialDerived.availableAccounts,
    lockedAccounts: initialDerived.lockedAccounts,

    login: (authData: AuthData) => {
      set((state) => {
        const accountInfo: AccountInfo = {
          user_id: authData.user_id,
          nickname: "",
          access_token_expires_at: authData.access_token_expires_at,
          is_locked: false,
        };
        const existingIndex = state.accounts.findIndex(
          (a) => a.user_id === authData.user_id,
        );
        let newAccounts: AccountInfo[];
        if (existingIndex >= 0) {
          newAccounts = [
            ...state.accounts.slice(0, existingIndex),
            accountInfo,
            ...state.accounts.slice(existingIndex + 1),
          ];
        } else {
          newAccounts = [...state.accounts, accountInfo];
        }
        saveToStorage(newAccounts, authData.user_id);
        return {
          accounts: newAccounts,
          currentUserId: authData.user_id,
          sessionValid: true,
        };
      });
    },

    logout: () => {
      set((state) => {
        const newAccounts = state.accounts.filter(
          (a) => a.user_id !== state.currentUserId,
        );
        const newCurrentUserId = newAccounts[0]?.user_id ?? null;
        saveToStorage(newAccounts, newCurrentUserId);
        return {
          accounts: newAccounts,
          currentUserId: newCurrentUserId,
          sessionValid: false,
          profile: null,
        };
      });
    },

    logoutAll: () => {
      saveToStorage([], null);
      set({
        accounts: [],
        currentUserId: null,
        sessionValid: false,
        profile: null,
      });
    },

    setProfile: (profile: MyProfile | null) => {
      set((state) => {
        if (profile && state.currentUserId) {
          const index = state.accounts.findIndex(
            (a) => a.user_id === state.currentUserId,
          );
          if (index >= 0) {
            const newAccounts = [
              ...state.accounts.slice(0, index),
              {
                ...state.accounts[index]!,
                nickname: profile.nickname,
                avatar_url: profile.avatar_url,
              },
              ...state.accounts.slice(index + 1),
            ];
            saveToStorage(newAccounts, state.currentUserId);
            return { profile, accounts: newAccounts };
          }
        }
        return { profile };
      });
    },

    invalidateSession: () => {
      set({ sessionValid: false, profile: null });
    },

    updateExpiresAt: (expiresAt: string) => {
      set((state) => {
        if (!state.currentUserId) return state;
        const index = state.accounts.findIndex(
          (a) => a.user_id === state.currentUserId,
        );
        if (index >= 0) {
          const newAccounts = [
            ...state.accounts.slice(0, index),
            { ...state.accounts[index]!, access_token_expires_at: expiresAt },
            ...state.accounts.slice(index + 1),
          ];
          saveToStorage(newAccounts, state.currentUserId);
          return { accounts: newAccounts };
        }
        return state;
      });
    },

    startRestoring: () => set({ isRestoring: true }),
    finishRestoring: () => set({ isRestoring: false }),

    switchAccount: (userId: string) => {
      const { accounts } = get();
      const account = accounts.find(
        (a) => a.user_id === userId && !a.is_locked,
      );
      if (account) {
        saveToStorage(accounts, userId);
        set({ currentUserId: userId, sessionValid: false, profile: null });
      }
    },

    lockAccount: (userId: string) => {
      set((state) => {
        const index = state.accounts.findIndex((a) => a.user_id === userId);
        if (index < 0) return state;
        const newAccounts = [
          ...state.accounts.slice(0, index),
          { ...state.accounts[index]!, is_locked: true },
          ...state.accounts.slice(index + 1),
        ];
        let newCurrentUserId = state.currentUserId;
        if (state.currentUserId === userId) {
          newCurrentUserId =
            newAccounts.find((a) => !a.is_locked)?.user_id ?? null;
        }
        saveToStorage(newAccounts, newCurrentUserId);
        return {
          accounts: newAccounts,
          currentUserId: newCurrentUserId,
          sessionValid:
            state.currentUserId === userId ? false : state.sessionValid,
          profile: state.currentUserId === userId ? null : state.profile,
        };
      });
    },

    unlockAccount: (userId: string) => {
      set((state) => {
        const index = state.accounts.findIndex((a) => a.user_id === userId);
        if (index < 0) return state;
        const newAccounts = [
          ...state.accounts.slice(0, index),
          { ...state.accounts[index]!, is_locked: false },
          ...state.accounts.slice(index + 1),
        ];
        saveToStorage(newAccounts, state.currentUserId);
        return { accounts: newAccounts };
      });
    },

    removeAccount: (userId: string) => {
      set((state) => {
        const newAccounts = state.accounts.filter((a) => a.user_id !== userId);
        const newCurrentUserId =
          state.currentUserId === userId
            ? (newAccounts[0]?.user_id ?? null)
            : state.currentUserId;
        saveToStorage(newAccounts, newCurrentUserId);
        return {
          accounts: newAccounts,
          currentUserId: newCurrentUserId,
          sessionValid:
            state.currentUserId === userId ? false : state.sessionValid,
          profile: state.currentUserId === userId ? null : state.profile,
        };
      });
    },

    isTokenExpiringSoon: (thresholdMinutes = 5) => {
      const { accounts, currentUserId } = get();
      const account = accounts.find(
        (a) => a.user_id === currentUserId && !a.is_locked,
      );
      if (!account?.access_token_expires_at) return true;
      const expiresAt = new Date(account.access_token_expires_at).getTime();
      return expiresAt - Date.now() < thresholdMinutes * 60 * 1000;
    },
  };
});
