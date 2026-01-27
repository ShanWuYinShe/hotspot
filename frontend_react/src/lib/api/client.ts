import { ApiResponse, ClientType, AuthTokenPairWeb, AuthTokenPairApp, UserInfo } from '@/types/api';
import { useAuthStore } from '@/stores/auth';
import { ENDPOINTS } from '@/lib/api/endpoints';

const CLIENT_TYPE: ClientType = 'web';

type RequestConfig = RequestInit & {
  skipAuth?: boolean;
};

let isRefreshing = false;
let refreshSubscribers: ((token: string) => void)[] = [];

const subscribeTokenRefresh = (cb: (token: string) => void) => {
  refreshSubscribers.push(cb);
};

const onRefreshed = (token: string) => {
  refreshSubscribers.forEach((cb) => cb(token));
  refreshSubscribers = [];
};

async function client<T>(endpoint: string, { skipAuth, ...customConfig }: RequestConfig = {}): Promise<ApiResponse<T>> {
  const store = useAuthStore.getState();
  const headers = {
    'Content-Type': 'application/json',
    ...(customConfig.headers || {}),
  } as Record<string, string>;

  if (store.accessToken && !skipAuth) {
    headers.Authorization = `Bearer ${store.accessToken}`;
  }

  const config: RequestInit = {
    method: 'POST',
    credentials: 'include',
    ...customConfig,
    headers,
  };

  const response = await fetch(endpoint, config);
  
  let data: ApiResponse<T>;
  try {
    data = await response.json();
  } catch {
    throw new Error('Invalid JSON response');
  }

  // Handle Access Token Invalid/Expired (10101)
  if (data.code === 10101) {
      if (!isRefreshing) {
          isRefreshing = true;
      try {
        const refreshRes = await fetch(ENDPOINTS.AUTH.REFRESH_TOKEN, {
          method: 'POST',
          credentials: 'include',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ client_type: CLIENT_TYPE }),
        });

        const refreshData: ApiResponse<AuthTokenPairWeb | AuthTokenPairApp> = await refreshRes.json();

        if (refreshData.code === 0 && refreshData.data?.access_token) {
          const newToken = refreshData.data.access_token;
          const fallbackUser: UserInfo = {
            user_id: refreshData.data.user_id,
            nickname: refreshData.data.nickname,
            user_type: 'normal',
          };
          store.setAuth(newToken, store.user ?? fallbackUser);

          onRefreshed(newToken);
          isRefreshing = false;

          const newHeaders = { ...headers, Authorization: `Bearer ${newToken}` };
          return client(endpoint, { ...customConfig, headers: newHeaders, skipAuth: false });
        }

        isRefreshing = false;
        store.clearAuth();
        return data;
      } catch {
        isRefreshing = false;
        store.clearAuth();
        return data;
      }
    } else {
          return new Promise((resolve) => {
              subscribeTokenRefresh((newToken) => {
                  const newHeaders = { ...headers, Authorization: `Bearer ${newToken}` };
                  resolve(client(endpoint, { ...customConfig, headers: newHeaders, skipAuth: false }));
              });
          });
    }
  }

  return data;
}


export const apiClient = {
  post: <T>(endpoint: string, body?: unknown, config?: RequestConfig) =>
    client<T>(endpoint, { ...config, method: 'POST', body: JSON.stringify(body) }),
};
