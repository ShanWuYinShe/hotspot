/** Centralized pagination page sizes to eliminate ~40 scattered magic numbers. */
export const PAGE_SIZE = {
  default: 20,
  chat: 50,
  relations: 24,
  profileGrid: 12,
  admin: 50,
} as const;
