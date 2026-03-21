<template>
  <div class="view-panel">
    <div class="view-panel__hd">
      <div>
        <h2 class="view-panel__title">Notifications</h2>
        <p class="view-panel__sub">Manage notification channels</p>
      </div>
      <button class="icon-btn" @click="fetchChannels" title="Refresh">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/>
          <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
        </svg>
      </button>
    </div>

    <div v-if="loading && !channels.length" class="view-panel__loading">
      <div class="base-btn__spinner" style="width:16px;height:16px"></div>
      Loading channels...
    </div>

    <div v-else-if="error" class="status-notice status-notice--error" style="margin:16px 24px">
      {{ error }}
    </div>

    <div v-else class="view-panel__body">
      <!-- Create form -->
      <div class="dash-section">
        <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:12px">
          <h3 class="dash-section__title" style="margin:0">Add Channel</h3>
          <button class="icon-btn" @click="showForm = !showForm" :title="showForm ? 'Collapse' : 'Expand'">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
              <polyline v-if="showForm" points="18 15 12 9 6 15"/>
              <polyline v-else points="6 9 12 15 18 9"/>
            </svg>
          </button>
        </div>

        <form v-if="showForm" @submit.prevent="handleCreate" style="display:flex;flex-direction:column;gap:12px">
          <div style="display:flex;flex-direction:column;gap:4px">
            <label style="font-size:12px;font-weight:600;color:var(--text-2)">Channel Name</label>
            <input v-model="form.name" class="auth-input" placeholder="e.g. Dev Alerts" required />
          </div>

          <div style="display:flex;flex-direction:column;gap:4px">
            <label style="font-size:12px;font-weight:600;color:var(--text-2)">Type</label>
            <div style="display:flex;gap:6px;flex-wrap:wrap">
              <button
                v-for="t in channelTypes" :key="t.value" type="button"
                class="status-tab"
                :class="{ 'status-tab--active': form.type === t.value }"
                @click="form.type = t.value"
              >
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" v-html="t.icon"></svg>
                {{ t.label }}
              </button>
            </div>
          </div>

          <!-- Webhook URL config (Slack/Discord/Teams) -->
          <div v-if="form.type && form.type !== 'email'" style="display:flex;flex-direction:column;gap:4px">
            <label style="font-size:12px;font-weight:600;color:var(--text-2)">Webhook URL</label>
            <input v-model="form.config.url" class="auth-input" type="url" placeholder="https://hooks.slack.com/..." required />
          </div>

          <!-- Email config -->
          <template v-if="form.type === 'email'">
            <div style="display:grid;grid-template-columns:1fr 100px;gap:8px">
              <div style="display:flex;flex-direction:column;gap:4px">
                <label style="font-size:12px;font-weight:600;color:var(--text-2)">SMTP Host</label>
                <input v-model="form.config.host" class="auth-input" placeholder="smtp.example.com" required />
              </div>
              <div style="display:flex;flex-direction:column;gap:4px">
                <label style="font-size:12px;font-weight:600;color:var(--text-2)">Port</label>
                <input v-model="form.config.port" class="auth-input" type="number" placeholder="587" required />
              </div>
            </div>
            <div style="display:grid;grid-template-columns:1fr 1fr;gap:8px">
              <div style="display:flex;flex-direction:column;gap:4px">
                <label style="font-size:12px;font-weight:600;color:var(--text-2)">Username</label>
                <input v-model="form.config.username" class="auth-input" placeholder="user@example.com" required />
              </div>
              <div style="display:flex;flex-direction:column;gap:4px">
                <label style="font-size:12px;font-weight:600;color:var(--text-2)">Password</label>
                <input v-model="form.config.password" class="auth-input" type="password" placeholder="••••••••" required />
              </div>
            </div>
            <div style="display:grid;grid-template-columns:1fr 1fr;gap:8px">
              <div style="display:flex;flex-direction:column;gap:4px">
                <label style="font-size:12px;font-weight:600;color:var(--text-2)">From</label>
                <input v-model="form.config.from" class="auth-input" type="email" placeholder="alerts@example.com" required />
              </div>
              <div style="display:flex;flex-direction:column;gap:4px">
                <label style="font-size:12px;font-weight:600;color:var(--text-2)">To</label>
                <input v-model="form.config.to" class="auth-input" type="email" placeholder="team@example.com" required />
              </div>
            </div>
          </template>

          <div style="display:flex;flex-direction:column;gap:4px">
            <label style="font-size:12px;font-weight:600;color:var(--text-2)">Events</label>
            <div class="webhook-events">
              <label v-for="evt in allEvents" :key="evt" class="webhook-event-check">
                <input type="checkbox" :value="evt" v-model="form.events" />
                {{ evt }}
              </label>
            </div>
          </div>

          <div style="display:flex;gap:8px">
            <button
              type="button" class="base-btn base-btn--ghost"
              :disabled="!form.type || testing"
              @click="handleTest"
            >
              <div v-if="testing" class="base-btn__spinner"></div>
              Test
            </button>
            <button
              type="submit" class="base-btn base-btn--primary"
              :disabled="!canCreate"
            >
              Add Channel
            </button>
          </div>
        </form>
      </div>

      <!-- Channel list -->
      <div class="dash-section" style="margin-top:20px">
        <h3 class="dash-section__title">Active Channels</h3>
        <div v-if="!channels.length" class="sidebar-empty" style="padding:16px 0">
          No notification channels configured yet.
        </div>
        <div v-else class="webhook-list">
          <div v-for="ch in channels" :key="ch.id" class="webhook-item">
            <div class="webhook-item__info">
              <div style="display:flex;align-items:center;gap:8px">
                <span class="webhook-item__url">{{ ch.name }}</span>
                <span class="audit-action" :style="typeBadgeStyle(ch.type)">{{ ch.type }}</span>
              </div>
              <div class="webhook-item__events">
                <span v-for="e in parseEvents(ch.events)" :key="e" class="audit-action">{{ e }}</span>
              </div>
            </div>
            <button class="icon-btn icon-btn--danger" @click="handleDelete(ch.id)" title="Delete channel">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
              </svg>
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useNotifications } from '../../composables/useNotifications.js'
import { useToast } from '../../composables/useToast.js'

const { channels, loading, error, fetchChannels, createChannel, deleteChannel, testChannel } = useNotifications()
const toast = useToast()

const showForm = ref(true)
const testing = ref(false)
const allEvents = ['upload', 'download', 'delete', 'transfer', 'share']

const channelTypes = [
  { value: 'slack', label: 'Slack', icon: '<path d="M14.5 10c-.83 0-1.5-.67-1.5-1.5v-5c0-.83.67-1.5 1.5-1.5s1.5.67 1.5 1.5v5c0 .83-.67 1.5-1.5 1.5z"/><path d="M20.5 10H19V8.5c0-.83.67-1.5 1.5-1.5s1.5.67 1.5 1.5-.67 1.5-1.5 1.5z"/><path d="M9.5 14c.83 0 1.5.67 1.5 1.5v5c0 .83-.67 1.5-1.5 1.5S8 21.33 8 20.5v-5c0-.83.67-1.5 1.5-1.5z"/><path d="M3.5 14H5v1.5c0 .83-.67 1.5-1.5 1.5S2 16.33 2 15.5 2.67 14 3.5 14z"/>' },
  { value: 'discord', label: 'Discord', icon: '<circle cx="9" cy="12" r="1"/><circle cx="15" cy="12" r="1"/><path d="M7.5 7.5c3-1 6-1 9 0"/><path d="M7 16.5c3 1 7 1 10 0"/><path d="M15.5 17c0 1 1.5 3 2 3 1.5 0 2.833-1.667 3.5-3 .667-1.667.5-5.833-1.5-11.5-1.457-.864-3-.5-4-.5l-1 2"/><path d="M8.5 17c0 1-1.356 3-1.832 3-1.429 0-2.698-1.667-3.333-3-.635-1.667-.476-5.833 1.428-11.5C6.151 4.636 7.5 5 8.5 5l1 2"/>' },
  { value: 'teams', label: 'Teams', icon: '<rect x="2" y="3" width="20" height="14" rx="2" ry="2"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/>' },
  { value: 'email', label: 'Email', icon: '<path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"/><polyline points="22,6 12,13 2,6"/>' },
]

const defaultConfig = () => ({ url: '', host: '', port: '', username: '', password: '', from: '', to: '' })

const form = ref({
  name: '',
  type: '',
  config: defaultConfig(),
  events: [],
})

const canCreate = computed(() => {
  if (!form.value.name || !form.value.type || form.value.events.length === 0) return false
  if (form.value.type === 'email') {
    return form.value.config.host && form.value.config.port && form.value.config.from && form.value.config.to
  }
  return !!form.value.config.url
})

function buildConfig() {
  if (form.value.type === 'email') {
    return JSON.stringify({
      host: form.value.config.host,
      port: form.value.config.port,
      username: form.value.config.username,
      password: form.value.config.password,
      from: form.value.config.from,
      to: form.value.config.to,
    })
  }
  return JSON.stringify({ url: form.value.config.url })
}

async function handleCreate() {
  try {
    await createChannel({
      name: form.value.name,
      type: form.value.type,
      config: buildConfig(),
      events: form.value.events,
    })
    toast.success('Channel created')
    form.value = { name: '', type: '', config: defaultConfig(), events: [] }
  } catch (err) {
    toast.error(err.message)
  }
}

async function handleTest() {
  testing.value = true
  try {
    await testChannel(form.value.type, buildConfig())
    toast.success('Test notification sent')
  } catch (err) {
    toast.error(err.message)
  } finally {
    testing.value = false
  }
}

async function handleDelete(id) {
  try {
    await deleteChannel(id)
    toast.success('Channel deleted')
  } catch (err) {
    toast.error(err.message)
  }
}

function parseEvents(events) {
  if (Array.isArray(events)) return events
  if (typeof events === 'string') return events.split(',').map(e => e.trim()).filter(Boolean)
  return []
}

function typeBadgeStyle(type) {
  const colors = {
    slack: { background: 'rgba(74,21,75,.1)', color: '#611f69' },
    discord: { background: 'rgba(88,101,242,.1)', color: '#5865F2' },
    teams: { background: 'rgba(70,71,174,.1)', color: '#4648AE' },
    email: { background: 'var(--accent-bg)', color: 'var(--accent)' },
  }
  return colors[type] || {}
}

onMounted(fetchChannels)
</script>
