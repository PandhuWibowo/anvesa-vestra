import { ref, computed } from 'vue'

const token = ref(localStorage.getItem('auth_token') || '')
const user = ref(null)
const authLoading = ref(false)
const authError = ref('')
const setupRequired = ref(false)
const authEnabled = ref(true)
const authChecked = ref(false)

export function useAuth() {
  const isAuthenticated = computed(() => !!token.value)

  function setToken(t) {
    token.value = t
    if (t) localStorage.setItem('auth_token', t)
    else localStorage.removeItem('auth_token')
  }

  function authHeaders() {
    if (!token.value) return {}
    return { 'Authorization': 'Bearer ' + token.value }
  }

  async function checkSetup() {
    try {
      const res = await fetch('/api/auth/setup-status')
      const data = await res.json()
      authEnabled.value = data.auth_enabled
      setupRequired.value = data.setup_required
      authChecked.value = true
    } catch {
      authChecked.value = true
    }
  }

  async function register(username, password) {
    authLoading.value = true
    authError.value = ''
    try {
      const res = await fetch('/api/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      })
      const data = await res.json()
      if (!res.ok) throw new Error(data.error || 'Registration failed')
      setToken(data.token)
      user.value = { id: data.user_id, username: data.username, role: data.role }
      setupRequired.value = false
      return true
    } catch (err) {
      authError.value = err.message
      return false
    } finally {
      authLoading.value = false
    }
  }

  async function login(username, password) {
    authLoading.value = true
    authError.value = ''
    try {
      const res = await fetch('/api/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      })
      const data = await res.json()
      if (!res.ok) throw new Error(data.error || 'Login failed')
      setToken(data.token)
      user.value = { id: data.user_id, username: data.username, role: data.role }
      return true
    } catch (err) {
      authError.value = err.message
      return false
    } finally {
      authLoading.value = false
    }
  }

  async function fetchMe() {
    if (!token.value) return
    try {
      const res = await fetch('/api/auth/me', {
        headers: authHeaders(),
      })
      if (res.ok) {
        const data = await res.json()
        user.value = { id: data.user_id, username: data.username, role: data.role }
      } else {
        setToken('')
        user.value = null
      }
    } catch {
      setToken('')
      user.value = null
    }
  }

  function logout() {
    setToken('')
    user.value = null
  }

  return {
    token, user, authLoading, authError, setupRequired, authEnabled, authChecked,
    isAuthenticated, authHeaders,
    checkSetup, register, login, fetchMe, logout,
  }
}
