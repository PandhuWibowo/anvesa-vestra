import { describe, it, expect, beforeEach } from 'vitest'
import { useAuth } from '../useAuth.js'

describe('useAuth', () => {
  beforeEach(() => {
    localStorage.clear()
    // Reset module state
    const { token, user } = useAuth()
    token.value = ''
    user.value = null
  })

  it('starts unauthenticated with no token', () => {
    const { isAuthenticated } = useAuth()
    expect(isAuthenticated.value).toBe(false)
  })

  it('returns auth headers when token is set', () => {
    const { token, authHeaders } = useAuth()
    token.value = 'test-token'
    localStorage.setItem('auth_token', 'test-token')
    expect(authHeaders()).toEqual({ 'Authorization': 'Bearer test-token' })
  })

  it('returns empty headers when no token', () => {
    const { authHeaders } = useAuth()
    expect(authHeaders()).toEqual({})
  })

  it('logout clears token and user', () => {
    const { token, user, logout, isAuthenticated } = useAuth()
    token.value = 'test-token'
    user.value = { username: 'test' }
    expect(isAuthenticated.value).toBe(true)

    logout()
    expect(isAuthenticated.value).toBe(false)
    expect(token.value).toBe('')
    expect(user.value).toBeNull()
  })
})
