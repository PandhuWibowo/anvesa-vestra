import { ref } from 'vue'
import { useAuth } from './useAuth.js'

const channels = ref([])
const loading = ref(false)
const error = ref('')

export function useNotifications() {
  const { authHeaders } = useAuth()

  async function fetchChannels() {
    loading.value = true
    error.value = ''
    try {
      const res = await fetch('/api/notifications', { headers: authHeaders() })
      if (!res.ok) throw new Error(await res.text())
      channels.value = (await res.json()) ?? []
    } catch (err) {
      error.value = err.message
    } finally {
      loading.value = false
    }
  }

  async function createChannel(data) {
    const res = await fetch('/api/notifications', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body: JSON.stringify({
        name: data.name,
        type: data.type,
        config: data.config,
        events: Array.isArray(data.events) ? data.events.join(',') : data.events,
      }),
    })
    const body = await res.json().catch(() => ({}))
    if (!res.ok) throw new Error(body.error || 'Failed to create channel')
    await fetchChannels()
    return true
  }

  async function deleteChannel(id) {
    const res = await fetch(`/api/notifications/${id}`, {
      method: 'DELETE',
      headers: authHeaders(),
    })
    if (!res.ok) throw new Error('Failed to delete channel')
    await fetchChannels()
  }

  async function testChannel(type, config) {
    const res = await fetch('/api/notifications/test', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body: JSON.stringify({ type, config }),
    })
    const body = await res.json().catch(() => ({}))
    if (!res.ok) throw new Error(body.error || 'Test failed')
    return { ok: true }
  }

  return { channels, loading, error, fetchChannels, createChannel, deleteChannel, testChannel }
}
