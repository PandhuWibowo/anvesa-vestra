<template>
  <div class="view-panel">
    <div class="view-panel__hd">
      <div>
        <h2 class="view-panel__title">Search</h2>
        <p class="view-panel__sub">Search objects across all connections</p>
      </div>
    </div>

    <div class="view-panel__toolbar">
      <div class="search-ctrl">
        <label class="search-ctrl__label">Provider</label>
        <select v-model="provider" class="base-input" style="max-width:180px">
          <option v-for="p in providers" :key="p.id" :value="p.id">{{ p.name }}</option>
        </select>
      </div>

      <div class="search-ctrl" v-if="filteredConns.length">
        <label class="search-ctrl__label">Connection</label>
        <select v-model="connectionId" class="base-input" style="max-width:220px">
          <option :value="0">All connections</option>
          <option v-for="c in filteredConns" :key="c.id" :value="c.id">{{ c.name }}</option>
        </select>
      </div>

      <div class="search-ctrl search-ctrl--grow">
        <label class="search-ctrl__label">Prefix / path</label>
        <form @submit.prevent="doSearch" style="display:flex;gap:6px">
          <input v-model="query" class="base-input" placeholder="e.g. images/2024/" style="flex:1" />
          <button class="base-btn base-btn--primary" :disabled="!query.trim() || searching" type="submit">
            <div v-if="searching" class="base-btn__spinner"></div>
            Search
          </button>
        </form>
      </div>
    </div>

    <div class="view-panel__body">
      <div v-if="error" class="status-notice status-notice--error" style="margin:0 0 12px">
        {{ error }}
      </div>

      <DragTable
        v-if="results.length || (hasSearched && !searching)"
        :columns="columns"
        :rows="results"
        :resizable-columns="true"
        :reorderable-columns="true"
        :column-toggle="true"
        striped
        @row-click="(row) => $emit('navigate', row)"
      >
        <template #cell-connection_name="{ row }">
          <div class="search-conn">
            <div class="conn-badge conn-badge--xs" :class="`conn-badge--${row.provider}`">
              <ProviderIcon :provider="row.provider" :size="9" />
            </div>
            {{ row.connection_name }}
          </div>
        </template>
        <template #cell-key="{ row }">
          <span class="file-name" style="gap:4px"><span class="file-icon">📄</span>{{ row.key }}</span>
        </template>
        <template #cell-size="{ row }">
          <span class="file-size">{{ formatSize(row.size) }}</span>
        </template>
        <template #cell-updated="{ row }">
          <span class="file-date">{{ formatDate(row.updated) }}</span>
        </template>
        <template #empty>
          <div style="display:flex;flex-direction:column;align-items:center;gap:8px">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="opacity:.4">
              <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
            </svg>
            No results found for "{{ lastQuery }}"
          </div>
        </template>
      </DragTable>

      <div v-if="results.length" class="view-panel__footer">
        {{ results.length }} result{{ results.length === 1 ? '' : 's' }}
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import DragTable from '../ui/DragTable.vue'
import { useSearch } from '../../composables/useSearch.js'
import ProviderIcon from '../ui/ProviderIcon.vue'

const props = defineProps({
  connections: { type: Array, default: () => [] },
})

defineEmits(['navigate'])

const providers = [
  { id: 'aws', name: 'Amazon S3' },
  { id: 'gcp', name: 'Google Cloud' },
  { id: 'azure', name: 'Azure Blob' },
  { id: 'alibaba', name: 'Alibaba OSS' },
  { id: 'huawei', name: 'Huawei OBS' },
]

const columns = [
  { key: 'connection_name', label: 'Connection', sortable: true },
  { key: 'key', label: 'Key', sortable: true },
  { key: 'size', label: 'Size', sortable: true, width: 100, align: 'right' },
  { key: 'updated', label: 'Modified', sortable: true, width: 150 },
]

const provider = ref('aws')
const connectionId = ref(0)
const query = ref('')
const hasSearched = ref(false)
const lastQuery = ref('')

const filteredConns = computed(() =>
  props.connections.filter(c => c.provider === provider.value)
)

const { results, searching, error, search } = useSearch()

async function doSearch() {
  if (!query.value.trim()) return
  hasSearched.value = true
  lastQuery.value = query.value
  await search(query.value, provider.value, connectionId.value || undefined)
}

function formatSize(bytes) {
  if (!bytes) return '—'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let size = bytes
  while (size >= 1024 && i < units.length - 1) { size /= 1024; i++ }
  return `${size.toFixed(i > 0 ? 1 : 0)} ${units[i]}`
}

function formatDate(d) {
  if (!d) return '—'
  return new Date(d).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
}
</script>
