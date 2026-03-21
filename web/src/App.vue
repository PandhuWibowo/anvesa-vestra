<template>
  <AuthScreen v-if="authChecked && authEnabled && !isAuthenticated" />
  <div v-else-if="authChecked" class="shell">
    <!-- Sidebar -->
    <AppSidebar
      :connections="connections"
      :loading="loading"
      :activeConn="activeConn"
      :activePrefix="activePrefix"
      :docsActive="mode === 'docs'"
      :activityActive="activityOpen"
      :splitActive="splitPane"
      :activeView="mode"
      :username="user?.username || ''"
      :authEnabled="authEnabled"
      @new-connection="startNew"
      @select="handleSelect"
      @edit="handleEdit"
      @delete="handleDelete"
      @docs="mode = 'docs'"
      @activity="activityOpen = !activityOpen"
      @split="toggleSplit"
      @bookmark-navigate="handleBookmarkNavigate"
      @logout="logout"
      @navigate="handleNavigate"
      @export-connections="handleExport"
      @import-connections="handleImport"
    />

    <!-- Main area -->
    <main class="main">

      <!-- Welcome -->
      <div v-if="mode === 'welcome'" class="welcome">
        <div class="welcome__icon">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M3 15a4 4 0 0 0 4 4h9a5 5 0 0 0 1.8-9.7 6 6 0 0 0-11.8-1A4 4 0 0 0 3 15z"/>
          </svg>
        </div>
        <p class="welcome__title">
          {{ connections.length ? 'Select a connection' : 'No connections yet' }}
        </p>
        <p class="welcome__sub">
          {{ connections.length
            ? 'Choose a connection from the sidebar to browse its files.'
            : 'Add your first cloud bucket to start browsing files.' }}
        </p>
        <BaseButton v-if="!connections.length" variant="primary" @click="startNew" style="margin-top:8px">
          New Connection
        </BaseButton>
      </div>

      <!-- New connection form -->
      <AddConnectionForm
        v-else-if="mode === 'form'"
        :testing="testing"
        :saving="saving"
        :error="error"
        :notice="notice"
        @test="testConnection"
        @save="handleSave"
      />

      <!-- Edit connection form -->
      <AddConnectionForm
        v-else-if="mode === 'edit' && editingConn"
        :testing="testing"
        :saving="saving"
        :error="error"
        :notice="notice"
        :editConn="editingConn"
        @test="testConnection"
        @save="handleUpdate"
      />

      <!-- Bucket browser (single or split) -->
      <template v-else-if="mode === 'browse' && activeConn">
        <div :class="splitPane ? 'split-pane' : 'solo-pane'">
          <!-- Left / solo pane -->
          <BucketBrowser
            :conn="activeConn"
            :connections="connections"
            :startPrefix="activePrefix"
            :paneId="splitPane ? 'left' : 'solo'"
            @delete="handleDelete(activeConn.provider, activeConn.id)"
          />

          <!-- Right pane (split mode) -->
          <template v-if="splitPane">
            <div class="split-divider" />
            <div v-if="!rightConn" class="split-placeholder">
              <div class="split-placeholder__inner">
                <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" style="opacity:.3;margin-bottom:8px">
                  <rect x="3" y="3" width="18" height="18" rx="2"/><line x1="12" y1="3" x2="12" y2="21"/>
                </svg>
                <p>Select a connection from the sidebar</p>
                <p style="font-size:11px;color:var(--muted);margin-top:4px">Click any connection to open it in this pane</p>
              </div>
            </div>
            <BucketBrowser
              v-else
              :conn="rightConn"
              :connections="connections"
              :startPrefix="rightPrefix"
              paneId="right"
              @delete="handleDelete(rightConn.provider, rightConn.id)"
            />
          </template>
        </div>
      </template>

      <!-- Documentation -->
      <DocsViewer v-else-if="mode === 'docs'" />

      <!-- Dashboard / Analytics -->
      <AnalyticsDashboard v-else-if="mode === 'dashboard'" />

      <!-- Search -->
      <SearchView v-else-if="mode === 'search'" :connections="connections" @navigate="handleSearchNavigate" />

      <!-- Shared Links -->
      <SharedLinksView v-else-if="mode === 'shared'" />

      <!-- Audit Log -->
      <AuditLogView v-else-if="mode === 'audit'" />

      <!-- Jobs -->
      <JobsView v-else-if="mode === 'jobs'" />

      <!-- Webhooks -->
      <WebhooksView v-else-if="mode === 'webhooks'" />

      <!-- Notifications -->
      <NotificationsView v-else-if="mode === 'notifications'" />

      <!-- Sync -->
      <SyncView v-else-if="mode === 'sync'" />

      <!-- Users -->
      <UsersView v-else-if="mode === 'users'" />

    </main>

    <!-- Activity panel (slides in over right edge) -->
    <ActivityPanel :open="activityOpen" @close="activityOpen = false" />

    <!-- Global overlays -->
    <ToastContainer />
    <ConfirmModal />
    <ShortcutModal :open="shortcutOpen" @close="shortcutOpen = false" />
  </div>
</template>

<script setup>
import { ref, watch, onMounted, onUnmounted, nextTick } from 'vue'
import AppSidebar        from './components/layout/AppHeader.vue'
import AddConnectionForm from './components/connections/AddConnectionForm.vue'
import BucketBrowser     from './components/connections/BucketBrowser.vue'
import DocsViewer        from './components/docs/DocsViewer.vue'
import ActivityPanel     from './components/ui/ActivityPanel.vue'
import BaseButton        from './components/ui/BaseButton.vue'
import ToastContainer    from './components/ui/ToastContainer.vue'
import ConfirmModal      from './components/ui/ConfirmModal.vue'
import AuthScreen        from './components/auth/AuthScreen.vue'
import AnalyticsDashboard from './components/views/AnalyticsDashboard.vue'
import SearchView        from './components/views/SearchView.vue'
import SharedLinksView   from './components/views/SharedLinksView.vue'
import AuditLogView      from './components/views/AuditLogView.vue'
import JobsView          from './components/views/JobsView.vue'
import WebhooksView      from './components/views/WebhooksView.vue'
import NotificationsView from './components/views/NotificationsView.vue'
import SyncView          from './components/views/SyncView.vue'
import UsersView         from './components/views/UsersView.vue'
import ShortcutModal     from './components/ui/ShortcutModal.vue'
import { useConnections } from './composables/useConnections.js'
import { useAuth }        from './composables/useAuth.js'
import { useToast }       from './composables/useToast.js'
import { useConfirm }     from './composables/useConfirm.js'
import { useConnectionBackup } from './composables/useConnectionBackup.js'

const {
  connections, loading, testing, saving, error, notice,
  fetchConnections, testConnection, saveConnection, updateConnection,
  removeConnection, clearMessages,
} = useConnections()

const { token, isAuthenticated, authChecked, authEnabled, checkSetup, fetchMe, logout, user } = useAuth()
const toast = useToast()
const confirm = useConfirm()
const { exportConnections, importConnections } = useConnectionBackup()

const mode        = ref('welcome') // 'welcome' | 'form' | 'edit' | 'browse' | 'docs' | 'dashboard' | 'search' | 'shared' | 'audit' | 'jobs' | 'webhooks'
const shortcutOpen = ref(false)
const activeConn  = ref(null)
const activePrefix = ref('')
const editingConn = ref(null)

function saveNavState() {
  const state = { mode: mode.value, prefix: activePrefix.value }
  if (activeConn.value) {
    state.connProvider = activeConn.value.provider
    state.connId = activeConn.value.id
  }
  localStorage.setItem('anveesa-nav', JSON.stringify(state))
}

function restoreNavState() {
  try {
    const raw = localStorage.getItem('anveesa-nav')
    if (!raw) return
    const state = JSON.parse(raw)
    const restorable = ['browse', 'docs', 'dashboard', 'search', 'shared', 'audit', 'jobs', 'webhooks', 'notifications', 'sync', 'users']
    if (!restorable.includes(state.mode)) return
    if (state.mode === 'browse' && state.connProvider != null && state.connId != null) {
      const conn = connections.value.find(
        c => c.provider === state.connProvider && c.id === state.connId
      )
      if (conn) {
        activeConn.value = conn
        activePrefix.value = state.prefix || ''
        mode.value = 'browse'
        return
      }
    }
    if (state.mode !== 'browse') {
      mode.value = state.mode
    }
  } catch { /* ignore corrupt data */ }
}

// ── Activity panel ─────────────────────────────────────────────
const activityOpen = ref(false)

// ── Split (dual-pane) ──────────────────────────────────────────
const splitPane   = ref(false)
const rightConn   = ref(null)
const rightPrefix = ref('')
// Track which pane receives the next sidebar click in split mode
const nextPane    = ref('left') // 'left' | 'right'

function toggleSplit() {
  splitPane.value = !splitPane.value
  if (!splitPane.value) {
    rightConn.value   = null
    rightPrefix.value = ''
    nextPane.value    = 'left'
  }
}

onMounted(async () => {
  // Handle OAuth redirect token
  const urlParams = new URLSearchParams(window.location.search)
  const oauthToken = urlParams.get('oauth_token')
  if (oauthToken) {
    localStorage.setItem('auth_token', oauthToken)
    token.value = oauthToken
    window.history.replaceState({}, '', window.location.pathname)
  }

  await checkSetup()
  if (authEnabled.value && token.value) {
    await fetchMe()
  }
  if (!authEnabled.value || isAuthenticated.value) {
    await fetchConnections()
    restoreNavState()
  }
  window.addEventListener('keydown', onAppKeyDown)
})
onUnmounted(() => window.removeEventListener('keydown', onAppKeyDown))

watch(isAuthenticated, async (authed) => {
  if (authed) {
    await fetchConnections()
    restoreNavState()
  }
})

watch([mode, activeConn, activePrefix], () => saveNavState())

function onAppKeyDown(e) {
  const inInput = ['INPUT', 'TEXTAREA'].includes(document.activeElement?.tagName)
  if (e.key === '?' && !inInput && !e.metaKey && !e.ctrlKey) {
    shortcutOpen.value = !shortcutOpen.value
    return
  }
  if ((e.key === 'n' || e.key === 'N') && !inInput && !e.metaKey && !e.ctrlKey) {
    startNew()
  }
}

// ── Navigation ────────────────────────────────────────────────

function handleSelect(conn) {
  if (splitPane.value) {
    if (nextPane.value === 'right') {
      rightConn.value   = conn
      rightPrefix.value = ''
      nextPane.value    = 'left'
    } else {
      activeConn.value   = conn
      activePrefix.value = ''
      nextPane.value     = 'right'
      mode.value         = 'browse'
    }
  } else {
    activeConn.value   = conn
    activePrefix.value = ''
    editingConn.value  = null
    mode.value         = 'browse'
  }
  clearMessages()
}

function startNew() {
  editingConn.value = null
  mode.value        = 'form'
  clearMessages()
}

function handleEdit(conn) {
  editingConn.value = conn
  mode.value        = 'edit'
  clearMessages()
}

// ── Bookmark navigation ───────────────────────────────────────

function handleBookmarkNavigate(bm) {
  const conn = connections.value.find(
    c => c.provider === bm.provider && String(c.id) === String(bm.id)
  )
  if (!conn) return
  if (splitPane.value && nextPane.value === 'right') {
    rightConn.value   = conn
    rightPrefix.value = bm.prefix ?? ''
    nextPane.value    = 'left'
  } else {
    activeConn.value   = conn
    activePrefix.value = bm.prefix ?? ''
    mode.value         = 'browse'
    nextPane.value     = 'right'
  }
  clearMessages()
}

// ── View navigation ──────────────────────────────────────────

function handleNavigate(view) {
  mode.value = view
  clearMessages()
}

function handleSearchNavigate(result) {
  const conn = connections.value.find(
    c => c.provider === result.provider && c.id === result.connection_id
  )
  if (!conn) return
  const parts = result.key.split('/')
  const prefix = parts.length > 1 ? parts.slice(0, -1).join('/') + '/' : ''
  activeConn.value   = conn
  activePrefix.value = prefix
  mode.value         = 'browse'
}

// ── Connection backup ─────────────────────────────────────────

async function handleExport() {
  try {
    await exportConnections()
    toast.success('Connections exported.')
  } catch {
    toast.error('Export failed.')
  }
}

function handleImport() {
  const input = document.createElement('input')
  input.type = 'file'
  input.accept = '.json'
  input.onchange = async (e) => {
    const file = e.target.files[0]
    if (!file) return
    try {
      const count = await importConnections(file)
      toast.success(`Imported ${count} connection(s).`)
      fetchConnections()
    } catch (err) {
      toast.error('Import failed: ' + err.message)
    }
  }
  input.click()
}

// ── Save / Update ─────────────────────────────────────────────

async function handleSave(provider, form, resolve) {
  const success = await saveConnection(provider, form)
  if (success) {
    const saved = connections.value.find(
      c => c.provider === provider && c.name === form.name
    )
    if (saved) {
      activeConn.value   = saved
      activePrefix.value = ''
      editingConn.value  = null
      mode.value         = 'browse'
    }
    toast.success('Connection saved.')
  }
  resolve?.(success)
}

async function handleUpdate(provider, form, resolve, id) {
  const success = await updateConnection(provider, id, form)
  if (success) {
    if (activeConn.value?.id === id && activeConn.value?.provider === provider) {
      const updated = connections.value.find(
        c => c.provider === provider && c.id === id
      )
      if (updated) activeConn.value = updated
    }
    editingConn.value = null
    mode.value        = activeConn.value ? 'browse' : 'welcome'
    toast.success('Connection updated.')
  }
  resolve?.(success)
}

// ── Delete ────────────────────────────────────────────────────

async function handleDelete(provider, id) {
  const ok = await confirm.confirm(
    'Are you sure you want to permanently delete this connection? This cannot be undone.',
    'Delete Connection'
  )
  if (!ok) return
  if (activeConn.value?.id === id && activeConn.value?.provider === provider) {
    activeConn.value   = null
    activePrefix.value = ''
    mode.value         = 'welcome'
  }
  if (rightConn.value?.id === id && rightConn.value?.provider === provider) {
    rightConn.value   = null
    rightPrefix.value = ''
  }
  if (editingConn.value?.id === id && editingConn.value?.provider === provider) {
    editingConn.value = null
    mode.value        = 'welcome'
  }
  removeConnection(provider, id)
}
</script>
