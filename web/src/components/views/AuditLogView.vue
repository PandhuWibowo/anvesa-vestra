<template>
  <div class="view-panel">
    <div class="view-panel__hd">
      <div>
        <h2 class="view-panel__title">Audit Log</h2>
        <p class="view-panel__sub">Track all platform activity</p>
      </div>
      <button class="icon-btn" @click="refresh" title="Refresh">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/>
          <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
        </svg>
      </button>
    </div>

    <div v-if="loading && !entries.length" class="view-panel__loading">
      <div class="base-btn__spinner" style="width:16px;height:16px"></div>
      Loading audit log...
    </div>

    <div v-else-if="error" class="status-notice status-notice--error" style="margin:16px 24px">
      {{ error }}
    </div>

    <div v-else class="view-panel__body">
      <DragTable
        :columns="columns"
        :rows="entries"
        row-key="id"
        :resizable-columns="true"
        :reorderable-columns="true"
        :column-toggle="true"
        striped
      >
        <template #cell-action="{ row }">
          <span class="audit-action" :class="`audit-action--${row.action}`">{{ row.action }}</span>
        </template>
        <template #cell-provider="{ row }">
          <span v-if="row.provider" class="base-badge" :class="`base-badge--${row.provider}`">{{ row.provider.toUpperCase() }}</span>
          <span v-else class="file-type">—</span>
        </template>
        <template #cell-object="{ row }">
          <span class="file-name" style="font-size:11px" v-if="row.object">{{ row.object }}</span>
          <span v-else class="file-type">—</span>
        </template>
        <template #cell-details="{ row }">
          <span class="file-type" style="max-width:200px;overflow:hidden;text-overflow:ellipsis;display:block">{{ row.details || '—' }}</span>
        </template>
        <template #cell-ip="{ row }">
          <span class="file-type" style="font-family:var(--mono);font-size:11px">{{ row.ip || '—' }}</span>
        </template>
        <template #cell-created_at="{ row }">
          <span class="file-date">{{ formatDate(row.created_at) }}</span>
        </template>
        <template #empty>
          <div style="display:flex;flex-direction:column;align-items:center;gap:8px">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="opacity:.4">
              <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>
            </svg>
            No audit entries yet
          </div>
        </template>
      </DragTable>

      <div v-if="entries.length >= 100" class="view-panel__footer">
        <button class="base-btn base-btn--ghost" @click="handleLoadMore" :disabled="loadingMore">
          <div v-if="loadingMore" class="base-btn__spinner"></div>
          Load more
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import DragTable from '../ui/DragTable.vue'
import { useAudit } from '../../composables/useAudit.js'

const { entries, loading, error, fetchAudit, loadMore } = useAudit()
const loadingMore = ref(false)

const columns = [
  { key: 'action', label: 'Action', sortable: true, width: 90 },
  { key: 'provider', label: 'Provider', sortable: true, width: 90 },
  { key: 'object', label: 'Object', sortable: true },
  { key: 'details', label: 'Details', width: 200 },
  { key: 'ip', label: 'IP', width: 120 },
  { key: 'created_at', label: 'Time', sortable: true, width: 170 },
]

onMounted(() => fetchAudit())
function refresh() { fetchAudit() }

async function handleLoadMore() {
  loadingMore.value = true
  await loadMore(entries.value.length)
  loadingMore.value = false
}

function formatDate(d) {
  if (!d) return '—'
  return new Date(d).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
}
</script>
