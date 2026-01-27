const CLIENT_ID_KEY = 'hotspot_client_id';

const generateClientId = () => {
  if (typeof crypto?.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`;
};

export const getClientId = () => {
  if (typeof window === 'undefined') {
    return 'server';
  }
  const existing = window.localStorage.getItem(CLIENT_ID_KEY);
  if (existing) {
    return existing;
  }
  const id = generateClientId();
  window.localStorage.setItem(CLIENT_ID_KEY, id);
  return id;
};

export const createIdempotencyKey = () => {
  const clientId = getClientId();
  const timestamp = Date.now();
  const random = generateClientId();
  return `${clientId}-${timestamp}-${random}`;
};
