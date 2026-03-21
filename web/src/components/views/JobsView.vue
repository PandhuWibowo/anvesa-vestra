<template>
  <div class="view-panel">
    <div class="view-panel__hd">
      <div>
        <h2 class="view-panel__title">Background Jobs</h2>
        <p class="view-panel__sub">Transfer, sync, and bulk operations</p>
      </div>
      <div style="display:flex;gap:6px;align-items:center">
        <span v-if="autoRefresh" class="file-type" style="font-size:10px">Auto-refreshing</span>
        <button class="icon-btn" @click="toggleAutoRefresh" :title="autoRefresh ? 'Stop auto-refresh' : 'Start auto-refresh'" :style="autoRefresh ? 'color:var(--accent)' : ''">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/>
            <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
          </svg>
        </button>
      </div>
    </div>

    <!-- Status filter -->
    <div class="view-panel__toolbar">
      <div class="status-tabs">
        <button
          v-for="tab in statusTabs"
          :key="tab.value"
          class="status-tab"
          :class="{ 'status-tab--active': statusFilter === tab.value }"
          @click="filterBy(tab.value)"
        >
          {{ tab.label }}
          <span v-if="tab.count != null" class="status-tab__count">{{ tab.count }}</span>
        </button>
      </div>
    </div>

    <div v-if="loading && !jobs.length" class="view-panel__loading">
      <div class="base-btn__spinner" style="width:16px;height:16px"></div>
      Loading jobs...
    </div>

    <div v-else-if="error" class="status-notice status-notice--error" style="margin:16px 24px">
      {{ error }}
    </div>

    <div v-else class="view-panel__body">
      <div v-if="!jobs.length" class="empty-state" style="padding:40px 16px">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="opacity:.4">
          <rect x="2" y="7" width="20" height="14" rx="2" ry="2"/><path d="M16 21V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v16"/>
        </svg>
        No jobs{{ statusFilter ? ` with status "${statusFilter}"` : '' }}
      </div>

      <div v-else class="jobs-list">
        <div class="job-item" v-for="job in jobs" :key="job.id" @click="selectJob(job)">
          <div class="job-item__top">
            <span class="job-type-badge">{{ job.type }}</span>
            <span class="job-status" :class="`job-status--${job.status}`">{{ job.status }}</span>
            <span class="file-date" style="margin-left:auto">{{ formatDate(job.created_at) }}</span>
          </div>
          <div v-if="job.status === 'running' || job.progress > 0" class="job-progress">
            <div class="progress-bar" style="max-width:none">
              <div class="progress-fill" :style="{ width: (job.progress * 100) + '%' }"></div>
            </div>
            <span class="file-type">{{ Math.round(job.progress * 100) }}%</span>
          </div>
          <div v-if="job.error" class="job-error">{{ job.error }}</div>
        </div>
      </div>
    </div>

    <!-- Job detail modal -->
    <Teleport to="body">
      <transition name="modal-fade">
        <div v-if="selectedJob" class="modal-backdrop" @click.self="selectedJob = null">
          <div class="modal" style="max-width:520px">
            <div class="modal-hd">
              <span class="modal-title">Job #{{ selectedJob.id }}</span>
              <button class="modal-close" @click="selectedJob = null">&times;</button>
            </div>
            <div class="modal-bd" style="gap:10px">
              <div class="meta-field">
                <span class="meta-label">Type</span>
                <span class="job-type-badge">{{ selectedJob.type }}</span>
              </div>
              <div class="meta-field">
                <span class="meta-label">Status</span>
                <span class="job-status" :class="`job-status--${selectedJob.status}`">{{ selectedJob.status }}</span>
              </div>
              <div class="meta-field" v-if="selectedJob.progress > 0">
                <span class="meta-label">Progress</span>
                <div style="display:flex;align-items:center;gap:8px">
                  <div class="progress-bar" style="max-width:none;flex:1">
                    <div class="progress-fill" :style="{ width: (selectedJob.progress * 100) + '%' }"></div>
                  </div>
                  <span class="file-type">{{ Math.round(selectedJob.progress * 100) }}%</span>
                </div>
              </div>
              <div class="meta-field" v-if="selectedJob.error">
                <span class="meta-label">Error</span>
                <span class="file-type" style="color:var(--danger)">{{ selectedJob.error }}</span>
              </div>
              <div class="meta-field" v-if="selectedJob.result">
                <span class="meta-label">Result</span>
                <code class="preview-text" style="font-size:11px;padding:8px;background:var(--surface-2);border-radius:var(--r-sm)">{{ selectedJob.result }}</code>
              </div>
              <div class="meta-field" v-if="selectedJob.payload">
                <span class="meta-label">Payload</span>
                <code class="preview-text" style="font-size:11px;padding:8px;background:var(--surface-2);border-radius:var(--r-sm);max-height:150px;overflow:auto">{{ formatPayload(selectedJob.payload) }}</code>
              </div>
              <div class="meta-field">
                <span class="meta-label">Created</span>
                <span class="file-date">{{ formatDate(selectedJob.created_at) }}</span>
              </div>
              <div class="meta-field">
                <span class="meta-label">Updated</span>
                <span class="file-date">{{ formatDate(selectedJob.updated_at) }}</span>
              </div>
            </div>
          </div>
        </div>
      </transition>
    </Teleport>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useJobs } from '../../composables/useJobs.js'

const { jobs, loading, error, fetchJobs, getJob } = useJobs()

const statusFilter = ref('')
const autoRefresh = ref(false)
const selectedJob = ref(null)
let refreshInterval = null

const statusTabs = computed(() => [
  { value: '', label: 'All' },
  { value: 'pending', label: 'Pending', count: jobs.value.filter(j => j.status === 'pending').length || null },
  { value: 'running', label: 'Running', count: jobs.value.filter(j => j.status === 'running').length || null },
  { value: 'completed', label: 'Completed' },
  { value: 'failed', label: 'Failed' },
])

function filterBy(status) {
  statusFilter.value = status
  fetchJobs(status || undefined)
}

function toggleAutoRefresh() {
  autoRefresh.value = !autoRefresh.value
  if (autoRefresh.value) {
    refreshInterval = setInterval(() => fetchJobs(statusFilter.value || undefined), 5000)
  } else {
    clearInterval(refreshInterval)
    refreshInterval = null
  }
}

async function selectJob(job) {
  const detail = await getJob(job.id)
  if (detail) selectedJob.value = detail
}

function formatDate(d) {
  if (!d) return '—'
  return new Date(d).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
}

function formatPayload(p) {
  try { return JSON.stringify(JSON.parse(p), null, 2) } catch { return p }
}

onMounted(() => fetchJobs())
onUnmounted(() => { if (refreshInterval) clearInterval(refreshInterval) })
</script>
