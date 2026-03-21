<template>
  <div class="view-panel">
    <div class="view-panel__hd">
      <div>
        <h2 class="view-panel__title">Sync Jobs</h2>
        <p class="view-panel__sub">Schedule automatic file sync between connections</p>
      </div>
      <button class="icon-btn" @click="fetchSyncJobs" title="Refresh">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/>
          <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
        </svg>
      </button>
    </div>

    <div v-if="loading && !syncJobs.length" class="view-panel__loading">
      <div class="base-btn__spinner" style="width:16px;height:16px"></div>
      Loading sync jobs...
    </div>

    <div v-else-if="error" class="status-notice status-notice--error" style="margin:16px 24px">
      {{ error }}
    </div>

    <div v-else class="view-panel__body">
      <!-- Create form -->
      <div class="dash-section">
        <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:12px">
          <h3 class="dash-section__title" style="margin:0">Create Sync Job</h3>
          <button class="icon-btn" @click="showForm = !showForm" :title="showForm ? 'Collapse' : 'Expand'">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
              <polyline v-if="showForm" points="18 15 12 9 6 15"/>
              <polyline v-else points="6 9 12 15 18 9"/>
            </svg>
          </button>
        </div>

        <form v-if="showForm" @submit.prevent="handleCreate" style="display:flex;flex-direction:column;gap:12px">
          <div style="display:flex;flex-direction:column;gap:4px">
            <label style="font-size:12px;font-weight:600;color:var(--text-2)">Job Name</label>
            <input v-model="form.name" class="auth-input" placeholder="e.g. Nightly backup" required />
          </div>

          <!-- Source -->
          <fieldset style="border:1px solid var(--border);border-radius:var(--r);padding:12px;display:flex;flex-direction:column;gap:8px">
            <legend style="font-size:11px;font-weight:600;color:var(--muted);text-transform:uppercase;letter-spacing:.4px;padding:0 4px">Source</legend>
            <div style="display:grid;grid-template-columns:1fr 1fr;gap:8px">
              <div style="display:flex;flex-direction:column;gap:4px">
                <label style="font-size:12px;font-weight:600;color:var(--text-2)">Provider</label>
                <select v-model="form.source_provider" class="auth-input" required>
                  <option value="" disabled>Select provider</option>
                  <option v-for="p in providers" :key="p" :value="p">{{ p.toUpperCase() }}</option>
                </select>
              </div>
              <div style="display:flex;flex-direction:column;gap:4px">
                <label style="font-size:12px;font-weight:600;color:var(--text-2)">Connection ID</label>
                <input v-model="form.source_connection_id" class="auth-input" type="number" placeholder="1" required />
              </div>
            </div>
            <div style="display:flex;flex-direction:column;gap:4px">
              <label style="font-size:12px;font-weight:600;color:var(--text-2)">Prefix</label>
              <input v-model="form.source_prefix" class="auth-input" placeholder="e.g. backups/" />
            </div>
          </fieldset>

          <!-- Destination -->
          <fieldset style="border:1px solid var(--border);border-radius:var(--r);padding:12px;display:flex;flex-direction:column;gap:8px">
            <legend style="font-size:11px;font-weight:600;color:var(--muted);text-transform:uppercase;letter-spacing:.4px;padding:0 4px">Destination</legend>
            <div style="display:grid;grid-template-columns:1fr 1fr;gap:8px">
              <div style="display:flex;flex-direction:column;gap:4px">
                <label style="font-size:12px;font-weight:600;color:var(--text-2)">Provider</label>
                <select v-model="form.dest_provider" class="auth-input" required>
                  <option value="" disabled>Select provider</option>
                  <option v-for="p in providers" :key="p" :value="p">{{ p.toUpperCase() }}</option>
                </select>
              </div>
              <div style="display:flex;flex-direction:column;gap:4px">
                <label style="font-size:12px;font-weight:600;color:var(--text-2)">Connection ID</label>
                <input v-model="form.dest_connection_id" class="auth-input" type="number" placeholder="2" required />
              </div>
            </div>
            <div style="display:flex;flex-direction:column;gap:4px">
              <label style="font-size:12px;font-weight:600;color:var(--text-2)">Prefix</label>
              <input v-model="form.dest_prefix" class="auth-input" placeholder="e.g. mirror/" />
            </div>
          </fieldset>

          <div style="display:flex;flex-direction:column;gap:4px">
            <label style="font-size:12px;font-weight:600;color:var(--text-2)">Schedule</label>
            <select v-model="form.schedule" class="auth-input" required>
              <option v-for="s in schedules" :key="s.value" :value="s.value">{{ s.label }}</option>
            </select>
          </div>

          <button
            type="submit" class="base-btn base-btn--primary"
            :disabled="!form.name || !form.source_provider || !form.source_connection_id || !form.dest_provider || !form.dest_connection_id"
          >
            Create Sync Job
          </button>
        </form>
      </div>

      <!-- Job list -->
      <div class="dash-section" style="margin-top:20px">
        <h3 class="dash-section__title">Sync Jobs</h3>
        <div v-if="!syncJobs.length" class="sidebar-empty" style="padding:16px 0">
          No sync jobs configured yet.
        </div>
        <div v-else class="jobs-list">
          <div v-for="job in syncJobs" :key="job.id" class="job-item" style="cursor:default">
            <div class="job-item__top">
              <span class="webhook-item__url" style="font-size:13px;font-weight:600">{{ job.name }}</span>
              <span class="job-status" :class="statusClass(job.status)">{{ job.status || 'idle' }}</span>
              <span class="file-type" style="margin-left:auto">{{ job.schedule }}</span>
            </div>
            <div style="display:flex;align-items:center;gap:6px;font-size:12px;color:var(--text-2)">
              <span class="base-badge" :class="`base-badge--${job.source_provider}`" v-if="job.source_provider">{{ job.source_provider.toUpperCase() }}</span>
              <span style="font-family:var(--mono);font-size:11px">{{ job.source_prefix || '/' }}</span>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--muted)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <line x1="5" y1="12" x2="19" y2="12"/><polyline points="12 5 19 12 12 19"/>
              </svg>
              <span class="base-badge" :class="`base-badge--${job.dest_provider}`" v-if="job.dest_provider">{{ job.dest_provider.toUpperCase() }}</span>
              <span style="font-family:var(--mono);font-size:11px">{{ job.dest_prefix || '/' }}</span>
            </div>
            <div v-if="job.last_run || job.next_run" style="display:flex;gap:16px;font-size:11px;color:var(--muted)">
              <span v-if="job.last_run">Last run: {{ formatDate(job.last_run) }}</span>
              <span v-if="job.next_run">Next run: {{ formatDate(job.next_run) }}</span>
            </div>
            <div style="display:flex;gap:6px;margin-top:4px">
              <button class="base-btn base-btn--ghost" style="padding:4px 10px;font-size:12px" @click="handleRun(job.id)" :disabled="job.status === 'running'">
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <polygon points="5 3 19 12 5 21 5 3"/>
                </svg>
                Run Now
              </button>
              <button class="base-btn base-btn--danger" style="padding:4px 10px;font-size:12px" @click="handleDelete(job.id)">
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                  <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
                </svg>
                Delete
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useSync } from '../../composables/useSync.js'
import { useToast } from '../../composables/useToast.js'

const { syncJobs, loading, error, fetchSyncJobs, createSyncJob, deleteSyncJob, runSyncJob } = useSync()
const toast = useToast()

const showForm = ref(true)
const providers = ['gcp', 'aws', 'azure', 'alibaba', 'huawei', 'gdrive']
const schedules = [
  { value: 'manual', label: 'Manual' },
  { value: 'hourly', label: 'Hourly' },
  { value: 'daily', label: 'Daily' },
  { value: 'weekly', label: 'Weekly' },
]

const form = ref({
  name: '',
  source_provider: '',
  source_connection_id: '',
  source_prefix: '',
  dest_provider: '',
  dest_connection_id: '',
  dest_prefix: '',
  schedule: 'manual',
})

function statusClass(status) {
  if (status === 'running') return 'job-status--running'
  if (status === 'error') return 'job-status--failed'
  return 'job-status--pending'
}

async function handleCreate() {
  try {
    await createSyncJob({
      name: form.value.name,
      source_provider: form.value.source_provider,
      source_connection_id: Number(form.value.source_connection_id),
      source_prefix: form.value.source_prefix,
      dest_provider: form.value.dest_provider,
      dest_connection_id: Number(form.value.dest_connection_id),
      dest_prefix: form.value.dest_prefix,
      schedule: form.value.schedule,
    })
    toast.success('Sync job created')
    form.value = { name: '', source_provider: '', source_connection_id: '', source_prefix: '', dest_provider: '', dest_connection_id: '', dest_prefix: '', schedule: 'manual' }
  } catch (err) {
    toast.error(err.message)
  }
}

async function handleRun(id) {
  try {
    await runSyncJob(id)
    toast.success('Sync job started')
  } catch (err) {
    toast.error(err.message)
  }
}

async function handleDelete(id) {
  try {
    await deleteSyncJob(id)
    toast.success('Sync job deleted')
  } catch (err) {
    toast.error(err.message)
  }
}

function formatDate(d) {
  if (!d) return '—'
  return new Date(d).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
}

onMounted(fetchSyncJobs)
</script>
