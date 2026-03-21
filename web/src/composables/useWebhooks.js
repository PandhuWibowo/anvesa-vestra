import { ref } from 'vue'
import { useAuth } from './useAuth.js'

export function useWebhooks() {
  const { authHeaders } = useAuth()
  const webhooks = ref([])
  const loading = ref(false)
  const error = ref('')

  async function fetchWebhooks() {
    loading.value = true
    error.value = ''
    try {
      const res = await fetch('/api/webhooks', { headers: authHeaders() })
      if (!res.ok) throw new Error(await res.text())
      webhooks.value = (await res.json()) ?? []
    } catch (err) {
      error.value = err.message
    } finally {
      loading.value = false
    }
  }

  async function createWebhook(url, events, secret) {
    const res = await fetch('/api/webhooks', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body: JSON.stringify({ url, events, secret }),
    })
    const data = await res.json().catch(() => ({}))
    if (!res.ok) throw new Error(data.error || 'Failed to create webhook')
    await fetchWebhooks()
    return data
  }

  async function deleteWebhook(id) {
    const res = await fetch(`/api/webhooks/${id}`, {
      method: 'DELETE',
      headers: authHeaders(),
    })
    if (!res.ok) throw new Error('Failed to delete webhook')
    await fetchWebhooks()
  }

  return { webhooks, loading, error, fetchWebhooks, createWebhook, deleteWebhook }
}
