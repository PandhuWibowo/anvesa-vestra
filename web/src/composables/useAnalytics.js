import { ref } from 'vue'
import { useAuth } from './useAuth.js'

export function useAnalytics() {
  const { authHeaders } = useAuth()
  const data = ref(null)
  const loading = ref(false)
  const error = ref('')

  async function fetchAnalytics() {
    loading.value = true
    error.value = ''
    try {
      const res = await fetch('/api/analytics', { headers: authHeaders() })
      if (!res.ok) throw new Error(await res.text())
      data.value = await res.json()
    } catch (err) {
      error.value = err.message
    } finally {
      loading.value = false
    }
  }

  return { data, loading, error, fetchAnalytics }
}
