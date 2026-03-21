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

  async function browseObjects(provider, bucket, credentials, prefix = '', pageToken = '') {
    const res = await fetch(BASE[provider] + '/bucket/browse', {
      method:  'POST',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body:    JSON.stringify({ bucket, credentials, prefix, page_token: pageToken }),
    })
    if (!res.ok) throw new Error(await res.text())
    return res.json() // { prefix, entries, next_page_token }
  }

  async function getDownloadURL(provider, bucket, credentials, object) {
    const res = await fetch(BASE[provider] + '/bucket/download', {
      method:  'POST',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body:    JSON.stringify({ bucket, credentials, object }),
    })
    if (!res.ok) throw new Error(await res.text())
    return (await res.json()).url
  }

  // proxyDownload: fetch file content through the backend to avoid CORS issues
  async function proxyDownload(provider, bucket, credentials, object) {
    const res = await fetch('/api/proxy/download', {
      method:  'POST',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body:    JSON.stringify({ provider, bucket, credentials, object }),
    })
    if (!res.ok) throw new Error(await res.text())
    return res
  }

  // presignUrl: like getDownloadURL but with a custom expiry (seconds)
  async function presignUrl(provider, bucket, credentials, object, expiresIn) {
    const res = await fetch(BASE[provider] + '/bucket/download', {
      method:  'POST',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body:    JSON.stringify({ bucket, credentials, object, expires_in: expiresIn }),
    })
    if (!res.ok) throw new Error(await res.text())
    return (await res.json()).url
  }

  // zipDownload: request a zip archive of a prefix or explicit object list
  async function zipDownload(provider, bucket, credentials, prefix, objects) {
    const res = await fetch('/api/zip', {
      method:  'POST',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body:    JSON.stringify({ provider, bucket, credentials, prefix: prefix ?? '', objects: objects ?? [] }),
    })
    if (!res.ok) throw new Error(await res.text())
    return res.blob()
  }

  async function deleteObject(provider, bucket, credentials, object) {
    const res = await fetch(BASE[provider] + '/bucket/delete', {
      method:  'POST',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body:    JSON.stringify({ bucket, credentials, object }),
    })
    if (!res.ok) throw new Error(await res.text())
  }

  async function copyObject(provider, bucket, credentials, source, destination, deleteSource = true) {
    const res = await fetch(BASE[provider] + '/bucket/copy', {
      method:  'POST',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body:    JSON.stringify({ bucket, credentials, source, destination, delete_source: deleteSource }),
    })
    if (!res.ok) throw new Error(await res.text())
  }

  async function uploadObjects(provider, bucket, credentials, prefix, files) {
    await Promise.all(Array.from(files).map(file => {
      const form = new FormData()
      form.append('bucket',      bucket)
      form.append('credentials', credentials)
      form.append('prefix',      prefix)
      form.append('file',        file)
      return fetch(BASE[provider] + '/bucket/upload', { method: 'POST', headers: authHeaders(), body: form }).then(r => {
        if (!r.ok) return r.text().then(t => { throw new Error(t) })
      })
    }))
  }

  async function deletePrefix(provider, bucket, credentials, prefix) {
    const res = await fetch(BASE[provider] + '/bucket/delete-prefix', {
      method:  'POST',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body:    JSON.stringify({ bucket, credentials, prefix }),
    })
    if (!res.ok) throw new Error(await res.text())
    return res.json() // { deleted: N }
  }

  async function transferObject(src, dst) {
    // src: { provider, bucket, credentials, object }
    // dst: { provider, bucket, credentials, prefix }
    const res = await fetch('/api/transfer', {
      method:  'POST',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body:    JSON.stringify({
        src_provider:    src.provider,
        src_bucket:      src.bucket,
        src_credentials: src.credentials,
        src_object:      src.object,
        dst_provider:    dst.provider,
        dst_bucket:      dst.bucket,
        dst_credentials: dst.credentials,
        dst_prefix:      dst.prefix,
      }),
    })
    if (!res.ok) throw new Error(await res.text())
    return res.json() // { destination: "path/file.txt" }
  }

  function uploadObjectWithProgress(provider, bucket, credentials, prefix, file, onProgress) {
    return new Promise((resolve, reject) => {
      const form = new FormData()
      form.append('bucket',      bucket)
      form.append('credentials', credentials)
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

  async function getBucketStats(provider, bucket, credentials) {
    const res = await fetch(BASE[provider] + '/bucket/stats', {
      method:  'POST',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body:    JSON.stringify({ bucket, credentials }),
    })
    if (!res.ok) throw new Error(await res.text())
    return res.json() // { object_count, total_size, truncated }
  }

  // ── metadata ─────────────────────────────────────────────────

  async function getObjectMetadata(provider, bucket, credentials, object) {
    const res = await fetch(BASE[provider] + '/bucket/metadata', {
      method:  'POST',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body:    JSON.stringify({ bucket, credentials, object }),
    })
    if (!res.ok) throw new Error(await res.text())
    return res.json() // { content_type, cache_control, metadata, size, updated, etag, md5? }
  }

  async function updateObjectMetadata(provider, bucket, credentials, object, patch) {
    const res = await fetch(BASE[provider] + '/bucket/metadata/update', {
      method:  'POST',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body:    JSON.stringify({ bucket, credentials, object, ...patch }),
    })
    if (!res.ok) throw new Error(await res.text())
  }

  // ── compat (flat listing) ────────────────────────────────────

  async function listObjects(provider, bucket, credentials) {
    const res = await fetch(BASE[provider] + '/bucket/objects', {
      method:  'POST',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body:    JSON.stringify({ bucket, credentials }),
    })
    if (!res.ok) throw new Error(await res.text())
    const data = await res.json()
    return { objects: data.objects ?? [], truncated: data.truncated ?? false }
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
  }
}
