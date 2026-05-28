import { useState, useCallback } from "react";
import { authService } from "@/lib/api/services/auth";
import { isSuccess } from "@/lib/api/client";

interface AuthActions {
  loading: boolean;
  login: (loginId: string, password: string) => Promise<boolean>;
  register: (
    loginId: string,
    nickname: string,
    password: string,
  ) => Promise<boolean>;
  logout: () => Promise<void>;
}

export function useAuthActions(): AuthActions {
  const [loading, setLoading] = useState(false);

  const login = useCallback(
    async (loginId: string, password: string): Promise<boolean> => {
      setLoading(true);
      try {
        const res = await authService.login({
          login_id: loginId,
          password,
          client_type: "web",
        });
        return isSuccess(res);
      } catch {
        return false;
      } finally {
        setLoading(false);
      }
    },
    [],
  );

  const register = useCallback(
    async (
      loginId: string,
      nickname: string,
      password: string,
    ): Promise<boolean> => {
      setLoading(true);
      try {
        const res = await authService.register({
          login_id: loginId,
          nickname,
          password,
          client_type: "web",
        });
        return isSuccess(res);
      } catch {
        return false;
      } finally {
        setLoading(false);
      }
    },
    [],
  );

  const logout = useCallback(async (): Promise<void> => {
    try {
      await authService.logout();
    } catch {}
  }, []);

  return { loading, login, register, logout };
}
