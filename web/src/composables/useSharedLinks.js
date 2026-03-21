import { ref } from 'vue'
import { useAuth } from './useAuth.js'

export function useSharedLinks() {
  const { authHeaders } = useAuth()
  const links = ref([])
  const loading = ref(false)
  const error = ref('')

  async function fetchLinks() {
    loading.value = true
    error.value = ''
    try {
      const res = await fetch('/api/shares', { headers: authHeaders() })
      if (!res.ok) throw new Error(await res.text())
      links.value = await res.json()
    } catch (err) {
      error.value = err.message
    } finally {
      loading.value = false
    }
  }

  async function createLink(params) {
    error.value = ''
    try {
      const res = await fetch('/api/share', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...authHeaders() },
        body: JSON.stringify(params),
      })
      if (!res.ok) throw new Error(await res.text())
      const data = await res.json()
      await fetchLinks()
      return data
    } catch (err) {
      error.value = err.message
      return null
    }
  }

  async function revokeLink(id) {
    error.value = ''
    try {
      const res = await fetch(`/api/shares/${id}`, { method: 'DELETE', headers: authHeaders() })
      if (!res.ok) throw new Error(await res.text())
      await fetchLinks()
      return true
    } catch (err) {
      error.value = err.message
      return false
    }
  }

  return { links, loading, error, fetchLinks, createLink, revokeLink }
}
