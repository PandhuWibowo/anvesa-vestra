import { ref } from 'vue'
import { useAuth } from './useAuth.js'

export function useAudit() {
  const { authHeaders } = useAuth()
  const entries = ref([])
  const loading = ref(false)
  const error = ref('')

  async function fetchAudit(limit = 100, offset = 0) {
    loading.value = true
    error.value = ''
    try {
      const res = await fetch(`/api/audit?limit=${limit}&offset=${offset}`, { headers: authHeaders() })
      if (!res.ok) throw new Error(await res.text())
      entries.value = await res.json()
    } catch (err) {
      error.value = err.message
    } finally {
      loading.value = false
    }
  }

  async function loadMore(offset) {
    try {
      const res = await fetch(`/api/audit?limit=100&offset=${offset}`, { headers: authHeaders() })
      if (!res.ok) throw new Error(await res.text())
      const more = await res.json()
      entries.value = [...entries.value, ...more]
      return more.length
    } catch (err) {
      error.value = err.message
      return 0
    }
  }

  return { entries, loading, error, fetchAudit, loadMore }
}
