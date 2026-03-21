import { ref } from 'vue'
import { useAuth } from './useAuth.js'

const users = ref([])
const loading = ref(false)
const error = ref('')

export function useUsers() {
  const { authHeaders } = useAuth()

  async function fetchUsers() {
    loading.value = true
    error.value = ''
    try {
      const res = await fetch('/api/users', { headers: authHeaders() })
      if (!res.ok) throw new Error(await res.text())
      users.value = (await res.json()) ?? []
    } catch (err) {
      error.value = err.message
    } finally {
      loading.value = false
    }
  }

  async function createUser(data) {
    const res = await fetch('/api/users', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body: JSON.stringify({ username: data.username, password: data.password, role: data.role }),
    })
    const body = await res.json().catch(() => ({}))
    if (!res.ok) throw new Error(body.error || 'Failed to create user')
    await fetchUsers()
    return body
  }

  async function updateUserRole(id, role) {
    const res = await fetch(`/api/users/${id}/role`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body: JSON.stringify({ role }),
    })
    const body = await res.json().catch(() => ({}))
    if (!res.ok) throw new Error(body.error || 'Failed to update user role')
    await fetchUsers()
    return body
  }

  async function deleteUser(id) {
    const res = await fetch(`/api/users/${id}`, {
      method: 'DELETE',
      headers: authHeaders(),
    })
    if (!res.ok) throw new Error('Failed to delete user')
    await fetchUsers()
  }

  return { users, loading, error, fetchUsers, createUser, updateUserRole, deleteUser }
}
