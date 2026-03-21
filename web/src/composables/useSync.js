import { ref } from 'vue'
import { useAuth } from './useAuth.js'

const syncJobs = ref([])
const loading = ref(false)
const error = ref('')

export function useSync() {
  const { authHeaders } = useAuth()

  async function fetchSyncJobs() {
    loading.value = true
    error.value = ''
    try {
      const res = await fetch('/api/sync', { headers: authHeaders() })
      if (!res.ok) throw new Error(await res.text())
      syncJobs.value = (await res.json()) ?? []
    } catch (err) {
      error.value = err.message
    } finally {
      loading.value = false
    }
  }

  async function createSyncJob(data) {
    const res = await fetch('/api/sync', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body: JSON.stringify(data),
    })
    const body = await res.json().catch(() => ({}))
    if (!res.ok) throw new Error(body.error || 'Failed to create sync job')
    await fetchSyncJobs()
    return body
  }

  async function updateSyncJob(id, data) {
    const res = await fetch(`/api/sync/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body: JSON.stringify(data),
    })
    const body = await res.json().catch(() => ({}))
    if (!res.ok) throw new Error(body.error || 'Failed to update sync job')
    await fetchSyncJobs()
    return body
  }

  async function deleteSyncJob(id) {
    const res = await fetch(`/api/sync/${id}`, {
      method: 'DELETE',
      headers: authHeaders(),
    })
    if (!res.ok) throw new Error('Failed to delete sync job')
    await fetchSyncJobs()
  }

  async function runSyncJob(id) {
    const res = await fetch(`/api/sync/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body: JSON.stringify({ status: 'running' }),
    })
    const body = await res.json().catch(() => ({}))
    if (!res.ok) throw new Error(body.error || 'Failed to run sync job')
    await fetchSyncJobs()
    return body
  }

  return { syncJobs, loading, error, fetchSyncJobs, createSyncJob, updateSyncJob, deleteSyncJob, runSyncJob }
}
