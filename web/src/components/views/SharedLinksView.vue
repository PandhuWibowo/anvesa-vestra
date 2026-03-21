<template>
  <div class="view-panel">
    <div class="view-panel__hd">
      <div>
        <h2 class="view-panel__title">Shared Links</h2>
        <p class="view-panel__sub">Manage public download links</p>
      </div>
      <button class="icon-btn" @click="fetchLinks" title="Refresh">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/>
          <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
        </svg>
      </button>
    </div>

    <div v-if="loading" class="view-panel__loading">
      <div class="base-btn__spinner" style="width:16px;height:16px"></div>
      Loading shared links...
    </div>

    <div v-else-if="error" class="status-notice status-notice--error" style="margin:16px 24px">
      {{ error }}
    </div>

    <div v-else class="view-panel__body">
      <DragTable
        :columns="columns"
        :rows="links"
        row-key="id"
        :draggable-rows="true"
        :resizable-columns="true"
        :reorderable-columns="true"
        :column-toggle="true"
        striped
        @row-reorder="onReorder"
      >
        <template #cell-object="{ row }">
          <span class="file-name" style="gap:4px">
            <span class="file-icon">📄</span>
            {{ row.object }}
          </span>
        </template>
        <template #cell-provider="{ row }">
          <span class="base-badge" :class="`base-badge--${row.provider}`">{{ row.provider.toUpperCase() }}</span>
        </template>
        <template #cell-download_count="{ row }">
          {{ row.download_count }}
          <span v-if="row.max_downloads > 0" class="file-type"> / {{ row.max_downloads }}</span>
        </template>
        <template #cell-expires_at="{ row }">
          <span class="file-date">
            <span v-if="row.expires_at" :style="isExpired(row) ? 'color:var(--danger)' : ''">{{ formatDate(row.expires_at) }}</span>
            <span v-else class="file-type">Never</span>
          </span>
        </template>
        <template #cell-created_at="{ row }">
          <span class="file-date">{{ formatDate(row.created_at) }}</span>
        </template>
        <template #cell-actions="{ row }">
          <div style="display:flex;gap:6px;white-space:nowrap">
            <button class="row-btn" style="opacity:1" @click.stop="copyLink(row.token)" title="Copy link">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
              </svg>
            </button>
            <button class="row-btn danger" style="opacity:1" @click.stop="handleRevoke(row.id)" title="Revoke link">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
              </svg>
            </button>
          </div>
        </template>
        <template #empty>
          <div style="display:flex;flex-direction:column;align-items:center;gap:8px">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="opacity:.4">
              <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/>
              <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>
            </svg>
            No shared links yet
          </div>
        </template>
      </DragTable>
    </div>
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import DragTable from '../ui/DragTable.vue'
import { useSharedLinks } from '../../composables/useSharedLinks.js'
import { useToast } from '../../composables/useToast.js'
import { useConfirm } from '../../composables/useConfirm.js'

const { links, loading, error, fetchLinks, revokeLink } = useSharedLinks()
const toast = useToast()
const confirm = useConfirm()

const columns = [
  { key: 'object', label: 'Object', sortable: true },
  { key: 'provider', label: 'Provider', sortable: true, width: 100 },
  { key: 'download_count', label: 'Downloads', sortable: true, width: 110 },
  { key: 'expires_at', label: 'Expires', sortable: true, width: 170 },
  { key: 'created_at', label: 'Created', sortable: true, width: 170 },
  { key: 'actions', label: 'Actions', width: 90 },
]

onMounted(fetchLinks)

function onReorder({ rows }) { links.value = rows }

function isExpired(link) {
  if (!link.expires_at) return false
  return new Date(link.expires_at) < new Date()
}

function copyLink(token) {
  const url = `${window.location.origin}/api/share/${token}`
  navigator.clipboard.writeText(url).then(() => toast.success('Link copied')).catch(() => toast.error('Failed to copy'))
}

async function handleRevoke(id) {
  const ok = await confirm.confirm('Revoke this shared link? It will no longer be accessible.', 'Revoke Link')
  if (!ok) return
  const success = await revokeLink(id)
  if (success) toast.success('Link revoked')
  else toast.error('Failed to revoke link')
}

function formatDate(d) {
  if (!d) return '—'
  return new Date(d).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
}
</script>
