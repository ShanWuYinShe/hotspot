import { defineStore } from 'pinia'
import type { UserSearchItem, UserStatsData } from '~/types/api'

export interface User extends UserSearchItem {
  stats?: UserStatsData
}

export const useUsersStore = defineStore('users', () => {
  const _users = ref<Record<number, User>>({})

  // Getter
  const getUser = (userId: number) => _users.value[userId]

  // Setter
  const setUser = (user: UserSearchItem) => {
    const existing = _users.value[user.user_id]
    _users.value[user.user_id] = { ...existing, ...user }
  }

  const setUsers = (users: UserSearchItem[]) => {
    if (!users || !Array.isArray(users)) return
    users.forEach(user => setUser(user))
  }

  const setUserStats = (userId: number, stats: UserStatsData) => {
    const user = _users.value[userId]
    if (user) {
      _users.value[userId] = { ...user, stats }
    }
  }

  return {
    users: _users,
    getUser,
    setUser,
    setUsers,
    setUserStats
  }
})
