import { ref } from 'vue'
import { useAuth } from './useAuth.js'

export function useConnectionBackup() {
  const { authHeaders } = useAuth()
  const importing = ref(false)
  const importError = ref('')

  async function exportConnections() {
    const res = await fetch('/api/connections/export', { headers: authHeaders() })
    if (!res.ok) throw new Error('Export failed')
    const blob = await res.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'anveesa-connections.json'
    a.click()
    URL.revokeObjectURL(url)
  }

  async function importConnections(file) {
    importing.value = true
    importError.value = ''
    try {
      const text = await file.text()
      const payload = JSON.parse(text)
      const res = await fetch('/api/connections/import', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...authHeaders() },
        body: JSON.stringify(payload),
      })
      const data = await res.json()
      if (!res.ok) throw new Error(data.error || 'Import failed')
      return data.imported
    } catch (err) {
      importError.value = err.message
      throw err
    } finally {
      importing.value = false
    }
  }

  return { importing, importError, exportConnections, importConnections }
}
