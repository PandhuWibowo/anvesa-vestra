<template>
  <div class="view-panel">
    <div class="view-panel__hd">
      <div>
        <h2 class="view-panel__title">Users</h2>
        <p class="view-panel__sub">Manage user accounts and roles</p>
      </div>
      <button class="icon-btn" @click="fetchUsers" title="Refresh">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/>
          <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
        </svg>
      </button>
    </div>

    <div v-if="loading && !users.length" class="view-panel__loading">
      <div class="base-btn__spinner" style="width:16px;height:16px"></div>
      Loading users...
    </div>

    <div v-else-if="error" class="status-notice status-notice--error" style="margin:16px 24px">
      {{ error }}
    </div>

    <div v-else class="view-panel__body">
      <!-- Create form -->
      <div class="dash-section">
        <h3 class="dash-section__title">Add User</h3>
        <form @submit.prevent="handleCreate" style="display:flex;flex-direction:column;gap:12px">
          <div style="display:grid;grid-template-columns:1fr 1fr;gap:8px">
            <div style="display:flex;flex-direction:column;gap:4px">
              <label style="font-size:12px;font-weight:600;color:var(--text-2)">Username</label>
              <input v-model="form.username" class="auth-input" placeholder="johndoe" required />
            </div>
            <div style="display:flex;flex-direction:column;gap:4px">
              <label style="font-size:12px;font-weight:600;color:var(--text-2)">Password</label>
              <input v-model="form.password" class="auth-input" type="password" placeholder="••••••••" required />
            </div>
          </div>
          <div style="display:flex;flex-direction:column;gap:4px">
            <label style="font-size:12px;font-weight:600;color:var(--text-2)">Role</label>
            <div style="display:flex;gap:6px">
              <button
                v-for="r in roles" :key="r.value" type="button"
                class="status-tab"
                :class="{ 'status-tab--active': form.role === r.value }"
                :style="form.role === r.value ? r.activeStyle : {}"
                @click="form.role = r.value"
              >
                {{ r.label }}
              </button>
            </div>
          </div>
          <button
            type="submit" class="base-btn base-btn--primary"
            :disabled="!form.username || !form.password || !form.role"
            style="align-self:flex-start"
          >
            Create User
          </button>
        </form>
      </div>

      <!-- User table -->
      <div class="dash-section" style="margin-top:20px">
        <h3 class="dash-section__title">All Users</h3>
        <DragTable
          :columns="columns"
          :rows="users"
          row-key="id"
          :draggable-rows="true"
          :resizable-columns="true"
          :reorderable-columns="true"
          :column-toggle="true"
          striped
          @row-reorder="onReorder"
        >
          <template #cell-username="{ row }">
            <span style="font-weight:500;font-size:13px">{{ row.username }}</span>
            <span v-if="isCurrentUser(row)" class="file-type" style="margin-left:6px">(you)</span>
          </template>
          <template #cell-role="{ row }">
            <span class="audit-action" :style="roleBadgeStyle(row.role)">{{ row.role }}</span>
          </template>
          <template #cell-created_at="{ row }">
            <span class="file-date">{{ formatDate(row.created_at) }}</span>
          </template>
          <template #cell-actions="{ row }">
            <div style="display:flex;align-items:center;gap:6px">
              <select
                :value="row.role"
                class="auth-input"
                style="padding:4px 8px;font-size:11px;width:auto"
                @change="handleRoleChange(row.id, $event.target.value)"
                :disabled="isCurrentUser(row)"
              >
                <option v-for="r in roles" :key="r.value" :value="r.value">{{ r.label }}</option>
              </select>
              <button
                class="icon-btn icon-btn--danger"
                @click.stop="handleDelete(row.id)"
                :disabled="isCurrentUser(row) || isLastAdmin(row)"
                :title="isCurrentUser(row) ? 'Cannot delete yourself' : isLastAdmin(row) ? 'Cannot delete last admin' : 'Delete user'"
              >
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                  <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
                </svg>
              </button>
            </div>
          </template>
          <template #empty>No users found.</template>
        </DragTable>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import DragTable from '../ui/DragTable.vue'
import { useUsers } from '../../composables/useUsers.js'
import { useAuth } from '../../composables/useAuth.js'
import { useToast } from '../../composables/useToast.js'

const { users, loading, error, fetchUsers, createUser, updateUserRole, deleteUser } = useUsers()
const { user: currentUser } = useAuth()
const toast = useToast()

const columns = [
  { key: 'username', label: 'Username', sortable: true, width: 180 },
  { key: 'role', label: 'Role', sortable: true, width: 100 },
  { key: 'created_at', label: 'Created', sortable: true, width: 180 },
  { key: 'actions', label: 'Actions', width: 160 },
]

const roles = [
  { value: 'admin', label: 'Admin', activeStyle: { background: 'var(--accent-bg)', borderColor: 'var(--accent)', color: 'var(--accent)' } },
  { value: 'editor', label: 'Editor', activeStyle: { background: 'rgba(66,133,244,.1)', borderColor: '#4285f4', color: '#4285f4' } },
  { value: 'viewer', label: 'Viewer', activeStyle: { background: 'var(--surface-2)', borderColor: 'var(--border-2)', color: 'var(--text-2)' } },
]

const form = ref({ username: '', password: '', role: 'viewer' })
const adminCount = computed(() => users.value.filter(u => u.role === 'admin').length)

function isCurrentUser(u) { return currentUser.value && currentUser.value.id === u.id }
function isLastAdmin(u) { return u.role === 'admin' && adminCount.value <= 1 }

function roleBadgeStyle(role) {
  if (role === 'admin') return { background: 'var(--accent-bg)', color: 'var(--accent)' }
  if (role === 'editor') return { background: 'rgba(66,133,244,.1)', color: '#4285f4' }
  return { background: 'var(--surface-2)', color: 'var(--muted)' }
}

function onReorder({ rows }) { users.value = rows }

async function handleCreate() {
  try {
    await createUser({ username: form.value.username, password: form.value.password, role: form.value.role })
    toast.success('User created')
    form.value = { username: '', password: '', role: 'viewer' }
  } catch (err) { toast.error(err.message) }
}

async function handleRoleChange(id, role) {
  try { await updateUserRole(id, role); toast.success('Role updated') }
  catch (err) { toast.error(err.message) }
}

async function handleDelete(id) {
  try { await deleteUser(id); toast.success('User deleted') }
  catch (err) { toast.error(err.message) }
}

function formatDate(d) {
  if (!d) return '—'
  return new Date(d).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
}

onMounted(fetchUsers)
</script>
