import { ref } from 'vue'
import { useAuth } from './useAuth.js'

const connections = ref([])
const loading     = ref(false)
const testing     = ref(false)
const saving      = ref(false)
const error       = ref('')
const notice      = ref('')

export function useConnections() {
  const { authHeaders } = useAuth()

  function clearMessages() { error.value = ''; notice.value = '' }

  const BASE = {
    gcp:     '/api/gcp',
    aws:     '/api/aws',
    huawei:  '/api/huawei',
    alibaba: '/api/alibaba',
    azure:   '/api/azure',
    gdrive:  '/api/gdrive',
  }

  // ── connection list ──────────────────────────────────────────

  async function fetchConnections() {
    loading.value = true
    clearMessages()
    try {
      const [gcpRes, awsRes, huaweiRes, alibabaRes, azureRes, gdriveRes] = await Promise.all([
        fetch('/api/gcp/connections', { headers: authHeaders() }).then(r => r.ok ? r.json() : []),
        fetch('/api/aws/connections', { headers: authHeaders() }).then(r => r.ok ? r.json() : []),
        fetch('/api/huawei/connections', { headers: authHeaders() }).then(r => r.ok ? r.json() : []),
        fetch('/api/alibaba/connections', { headers: authHeaders() }).then(r => r.ok ? r.json() : []),
        fetch('/api/azure/connections', { headers: authHeaders() }).then(r => r.ok ? r.json() : []),
        fetch('/api/gdrive/connections', { headers: authHeaders() }).then(r => r.ok ? r.json() : []),
      ])
      const gcpList     = (gcpRes     || []).map(c => ({ ...c, provider: 'gcp' }))
      const awsList     = (awsRes     || []).map(c => ({ ...c, provider: 'aws' }))
      const huaweiList  = (huaweiRes  || []).map(c => ({ ...c, provider: 'huawei' }))
      const alibabaList = (alibabaRes || []).map(c => ({ ...c, provider: 'alibaba' }))
      const azureList   = (azureRes   || []).map(c => ({ ...c, provider: 'azure' }))
      const gdriveList  = (gdriveRes  || []).map(c => ({ ...c, provider: 'gdrive' }))
      connections.value = [...gcpList, ...awsList, ...huaweiList, ...alibabaList, ...azureList, ...gdriveList]
    } catch (err) {
      error.value = 'Failed to load connections.'
    } finally {
      loading.value = false
    }
  }

  async function testConnection(provider, bucket, credentials) {
    testing.value = true
    clearMessages()
    try {
      const res = await fetch(BASE[provider] + '/test', {
        method:  'POST',
        headers: { 'Content-Type': 'application/json', ...authHeaders() },
        body:    JSON.stringify({ bucket, credentials }),
      })
      if (!res.ok) error.value = 'Test failed: ' + await res.text()
      else notice.value = 'Connection test succeeded ✓'
    } catch (err) {
      error.value = 'Error: ' + err.message
    } finally {
      testing.value = false
    }
  }

  async function saveConnection(provider, form) {
    saving.value = true
    clearMessages()
    try {
      const res = await fetch(BASE[provider] + '/connection', {
        method:  'POST',
        headers: { 'Content-Type': 'application/json', ...authHeaders() },
        body:    JSON.stringify(form),
      })
      if (!res.ok) { error.value = 'Save failed: ' + await res.text(); return false }
      notice.value = 'Connection saved ✓'
      await fetchConnections()
      return true
    } catch (err) {
      error.value = 'Error: ' + err.message
      return false
    } finally {
      saving.value = false
    }
  }

  async function updateConnection(provider, id, form) {
    saving.value = true
    clearMessages()
    try {
      const res = await fetch(`${BASE[provider]}/connection/${id}`, {
        method:  'PUT',
        headers: { 'Content-Type': 'application/json', ...authHeaders() },
        body:    JSON.stringify(form),
      })
      if (!res.ok) { error.value = 'Update failed: ' + await res.text(); return false }
      notice.value = 'Connection updated ✓'
      await fetchConnections()
      return true
    } catch (err) {
      error.value = 'Error: ' + err.message
      return false
    } finally {
      saving.value = false
    }
  }

  async function removeConnection(provider, id) {
    clearMessages()
    try {
      const res = await fetch(`${BASE[provider]}/connection/${id}`, { method: 'DELETE', headers: authHeaders() })
      if (res.ok) await fetchConnections()
    } catch (err) {
      error.value = 'Delete failed: ' + err.message
    }
  }

  // ── bucket browsing ──────────────────────────────────────────

  async function browseObjects(provider, connectionId, prefix = '', pageToken = '') {
    const res = await fetch(BASE[provider] + '/bucket/browse', {
      method:  'POST',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body:    JSON.stringify({ connection_id: connectionId, prefix, page_token: pageToken }),
    })
    if (!res.ok) throw new Error(await res.text())
    return res.json()
  }

  async function getDownloadURL(provider, connectionId, object, expiresIn) {
    const body = { connection_id: connectionId, object }
    if (expiresIn) body.expires_in = expiresIn
    const res = await fetch(BASE[provider] + '/bucket/download', {
      method:  'POST',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body:    JSON.stringify(body),
    })
    if (!res.ok) throw new Error(await res.text())
    return (await res.json()).url
  }

  async function proxyDownload(provider, connectionId, object) {
    const res = await fetch('/api/proxy/download', {
      method:  'POST',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body:    JSON.stringify({ provider, connection_id: connectionId, object }),
    })
    if (!res.ok) throw new Error(await res.text())
    return res
  }

  async function presignUrl(provider, connectionId, object, expiresIn) {
    const res = await fetch(BASE[provider] + '/bucket/download', {
      method:  'POST',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body:    JSON.stringify({ connection_id: connectionId, object, expires_in: expiresIn }),
    })
    if (!res.ok) throw new Error(await res.text())
    return (await res.json()).url
  }

  async function zipDownload(provider, connectionId, prefix, objects) {
    const res = await fetch('/api/zip', {
      method:  'POST',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body:    JSON.stringify({ provider, connection_id: connectionId, prefix: prefix ?? '', objects: objects ?? [] }),
    })
    if (!res.ok) throw new Error(await res.text())
    return res.blob()
  }

  async function deleteObject(provider, connectionId, object) {
    const res = await fetch(BASE[provider] + '/bucket/delete', {
      method:  'POST',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body:    JSON.stringify({ connection_id: connectionId, object }),
    })
    if (!res.ok) throw new Error(await res.text())
  }

  async function copyObject(provider, connectionId, source, destination, deleteSource = true) {
    const res = await fetch(BASE[provider] + '/bucket/copy', {
      method:  'POST',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body:    JSON.stringify({ connection_id: connectionId, source, destination, delete_source: deleteSource }),
    })
    if (!res.ok) throw new Error(await res.text())
  }

  async function uploadObjects(provider, connectionId, prefix, files) {
    await Promise.all(Array.from(files).map(file => {
      const form = new FormData()
      form.append('connection_id', String(connectionId))
      form.append('prefix',      prefix)
      form.append('file',        file)
      return fetch(BASE[provider] + '/bucket/upload', { method: 'POST', headers: authHeaders(), body: form }).then(r => {
        if (!r.ok) return r.text().then(t => { throw new Error(t) })
      })
    }))
  }

  async function deletePrefix(provider, connectionId, prefix) {
    const res = await fetch(BASE[provider] + '/bucket/delete-prefix', {
      method:  'POST',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body:    JSON.stringify({ connection_id: connectionId, prefix }),
    })
    if (!res.ok) throw new Error(await res.text())
    return res.json()
  }

  async function transferObject(src, dst) {
    const res = await fetch('/api/transfer', {
      method:  'POST',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body:    JSON.stringify({
        src_provider:      src.provider,
        src_connection_id: src.connectionId,
        src_object:        src.object,
        dst_provider:      dst.provider,
        dst_connection_id: dst.connectionId,
        dst_prefix:        dst.prefix,
      }),
    })
    if (!res.ok) throw new Error(await res.text())
    return res.json() // { transfer_id, destination }
  }

  function watchTransferProgress(transferId, onProgress) {
    const headers = authHeaders()
    const token   = headers.Authorization ? headers.Authorization.replace('Bearer ', '') : ''
    const url     = `/api/transfer/progress?id=${transferId}${token ? `&token=${token}` : ''}`
    const es      = new EventSource(url)
    es.onmessage = e => {
      try {
        const data = JSON.parse(e.data)
        onProgress(data)
        if (data.done) es.close()
      } catch {}
    }
    es.onerror = () => es.close()
    return es
  }

  function uploadObjectWithProgress(provider, connectionId, prefix, file, onProgress) {
    return new Promise((resolve, reject) => {
      const form = new FormData()
      form.append('connection_id', String(connectionId))
      form.append('prefix',      prefix)
      form.append('file',        file)
      const xhr = new XMLHttpRequest()
      xhr.upload.addEventListener('progress', e => {
        if (e.lengthComputable) onProgress?.(e.loaded / e.total)
      })
      xhr.addEventListener('load', () => {
        if (xhr.status >= 200 && xhr.status < 300) resolve()
        else reject(new Error(xhr.responseText || `HTTP ${xhr.status}`))
      })
      xhr.addEventListener('error', () => reject(new Error('Network error')))
      xhr.open('POST', BASE[provider] + '/bucket/upload')
      const auth = authHeaders()
      if (auth.Authorization) xhr.setRequestHeader('Authorization', auth.Authorization)
      xhr.send(form)
    })
  }

  async function getBucketStats(provider, connectionId) {
    const res = await fetch(BASE[provider] + '/bucket/stats', {
      method:  'POST',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body:    JSON.stringify({ connection_id: connectionId }),
    })
    if (!res.ok) throw new Error(await res.text())
    return res.json()
  }

  // ── metadata ─────────────────────────────────────────────────

  async function getObjectMetadata(provider, connectionId, object) {
    const res = await fetch(BASE[provider] + '/bucket/metadata', {
      method:  'POST',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body:    JSON.stringify({ connection_id: connectionId, object }),
    })
    if (!res.ok) throw new Error(await res.text())
    return res.json()
  }

  async function updateObjectMetadata(provider, connectionId, object, patch) {
    const res = await fetch(BASE[provider] + '/bucket/metadata/update', {
      method:  'POST',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body:    JSON.stringify({ connection_id: connectionId, object, ...patch }),
    })
    if (!res.ok) throw new Error(await res.text())
  }

  // ── compat (flat listing) ────────────────────────────────────

  async function listObjects(provider, connectionId) {
    const res = await fetch(BASE[provider] + '/bucket/objects', {
      method:  'POST',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body:    JSON.stringify({ connection_id: connectionId }),
    })
    if (!res.ok) throw new Error(await res.text())
    const data = await res.json()
    return { objects: data.objects ?? [], truncated: data.truncated ?? false }
  }

  async function createFolder(provider, connectionId, prefix, name) {
    const res = await fetch(BASE[provider] + '/bucket/create-folder', {
      method:  'POST',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body:    JSON.stringify({ connection_id: connectionId, prefix, name }),
    })
    if (!res.ok) throw new Error(await res.text())
    return res.json()
  }

  return {
    connections, loading, testing, saving, error, notice,
    fetchConnections, testConnection, saveConnection, updateConnection,
    removeConnection, clearMessages,
    browseObjects, getDownloadURL, proxyDownload, presignUrl, zipDownload, deleteObject, copyObject,
    uploadObjects, uploadObjectWithProgress,
    deletePrefix, transferObject,
    getBucketStats, listObjects,
    getObjectMetadata, updateObjectMetadata,
    createFolder, watchTransferProgress,
  }
}
