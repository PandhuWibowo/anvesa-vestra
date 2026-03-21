import { ref } from 'vue'
import { useAuth } from './useAuth.js'

export function useSearch() {
  const { authHeaders } = useAuth()
  const results = ref([])
  const searching = ref(false)
  const error = ref('')

  async function search(query, provider, connectionId) {
    searching.value = true
    error.value = ''
    results.value = []
    try {
      const body = { query, provider }
      if (connectionId) body.connection_id = connectionId
      const res = await fetch('/api/search', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...authHeaders() },
        body: JSON.stringify(body),
      })
      if (!res.ok) throw new Error(await res.text())
      const data = await res.json()
      results.value = data.results || []
    } catch (err) {
      error.value = err.message
    } finally {
      searching.value = false
    }
  }

  function clearResults() {
    results.value = []
    error.value = ''
  }

  return { results, searching, error, search, clearResults }
}
