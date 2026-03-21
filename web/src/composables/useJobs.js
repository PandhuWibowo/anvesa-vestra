import { ref } from 'vue'
import { useAuth } from './useAuth.js'

export function useJobs() {
  const { authHeaders } = useAuth()
  const jobs = ref([])
  const loading = ref(false)
  const error = ref('')

  async function fetchJobs(status) {
    loading.value = true
    error.value = ''
    try {
      let url = '/api/jobs'
      if (status) url += `?status=${status}`
      const res = await fetch(url, { headers: authHeaders() })
      if (!res.ok) throw new Error(await res.text())
      jobs.value = await res.json()
    } catch (err) {
      error.value = err.message
    } finally {
      loading.value = false
    }
  }

  async function createJob(type, payload) {
    error.value = ''
    try {
      const res = await fetch('/api/jobs', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...authHeaders() },
        body: JSON.stringify({ type, payload }),
      })
      if (!res.ok) throw new Error(await res.text())
      const data = await res.json()
      await fetchJobs()
      return data
    } catch (err) {
      error.value = err.message
      return null
    }
  }

  async function getJob(id) {
    try {
      const res = await fetch(`/api/jobs/${id}`, { headers: authHeaders() })
      if (!res.ok) throw new Error(await res.text())
      return await res.json()
    } catch (err) {
      error.value = err.message
      return null
    }
  }

  return { jobs, loading, error, fetchJobs, createJob, getJob }
}
