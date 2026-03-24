<template>
  <div class="view-panel">
    <div class="view-panel__hd">
      <div>
        <h2 class="view-panel__title">Bucket Management</h2>
        <p class="view-panel__sub">List, create, and delete buckets / containers across your connections</p>
      </div>
    </div>

    <div class="view-panel__body">

      <!-- Connection selector -->
      <div class="dash-section">
        <h3 class="dash-section__title">Select Connection</h3>
        <div class="bm-conn-grid">
          <div
            v-for="c in supportedConnections"
            :key="`${c.provider}-${c.id}`"
            class="bm-conn-card"
            :class="{ active: selected?.id === c.id && selected?.provider === c.provider }"
            @click="selectConn(c)"
          >
            <ProviderIcon :provider="c.provider" :size="14" />
            <span class="bm-conn-name">{{ c.name }}</span>
            <span class="bm-conn-bucket">{{ c.bucket }}</span>
          </div>
        </div>
        <p v-if="!supportedConnections.length" style="font-size:13px;color:var(--muted)">
          No supported connections. Add an AWS S3, Alibaba OSS, GCS, or Azure connection first.
        </p>
      </div>

      <!-- Bucket list -->
      <div v-if="selected" class="dash-section">
        <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:14px">
          <h3 class="dash-section__title" style="margin-bottom:0">
            Buckets — {{ selected.name }}
          </h3>
          <div style="display:flex;gap:8px">
            <button class="icon-btn" @click="loadBuckets" :disabled="bucketsLoading" title="Refresh">
              <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/>
                <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
              </svg>
            </button>
            <button class="base-btn base-btn--primary" @click="showCreate = true" style="font-size:12px;padding:6px 12px">
              + New Bucket
            </button>
          </div>
        </div>

        <div v-if="bucketsLoading" style="display:flex;align-items:center;gap:8px;color:var(--muted);font-size:13px;padding:16px 0">
          <div class="base-btn__spinner" style="width:14px;height:14px"></div>
          Loading buckets…
        </div>

        <div v-else-if="bucketsError" class="status-notice status-notice--error">
          {{ bucketsError }}
        </div>

        <table v-else-if="buckets.length" class="file-table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Location</th>
              <th>Created</th>
              <th style="width:100px"></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="b in buckets" :key="b.name" class="file-row">
              <td>
                <div style="display:flex;align-items:center;gap:8px">
                  <svg width="13" height="13" viewBox="0 0 24 24" fill="currentColor" stroke="none" style="color:var(--aws);opacity:.7;flex-shrink:0">
                    <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
                  </svg>
                  <strong style="font-size:13px;color:var(--text)">{{ b.name }}</strong>
                </div>
              </td>
              <td style="font-size:12px;color:var(--text-2)">{{ b.location || '—' }}</td>
              <td style="font-size:12px;color:var(--muted)">{{ formatDate(b.created_at) }}</td>
              <td>
                <button class="row-btn danger" @click="confirmDelete(b)" title="Delete bucket (must be empty)">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/>
                    <path d="M10 11v6"/><path d="M14 11v6"/><path d="M9 6V4h6v2"/>
                  </svg>
                </button>
              </td>
            </tr>
          </tbody>
        </table>

        <div v-else class="empty-state" style="padding:32px 0">
          <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" style="opacity:.3;margin-bottom:6px">
            <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
          </svg>
          <p style="font-size:13px;color:var(--text-2)">No buckets found.</p>
        </div>
      </div>

    </div>
  </div>

  <!-- Create bucket modal -->
  <BaseModal :open="showCreate" title="Create Bucket" @update:open="showCreate = false">
    <div style="display:flex;flex-direction:column;gap:12px">
      <div>
        <label class="form-label">Bucket Name</label>
        <input
          ref="bucketNameInput"
          v-model="newBucketName"
          class="base-input"
          :placeholder="selected?.provider === 'azure' ? 'my-container' : 'my-bucket-name'"
          @keydown.enter="doCreate"
          @keydown.escape.stop="showCreate = false"
        />
        <p class="form-hint">Must be globally unique. Use lowercase letters, numbers, and hyphens.</p>
      </div>
      <div v-if="selected?.provider === 'aws' || selected?.provider === 'alibaba'">
        <label class="form-label">Region <span style="color:var(--muted);font-weight:400">(optional)</span></label>
        <input v-model="newBucketRegion" class="base-input" placeholder="us-east-1" />
      </div>
      <div v-if="selected?.provider === 'gcp'">
        <label class="form-label">Location <span style="color:var(--muted);font-weight:400">(optional)</span></label>
        <input v-model="newBucketRegion" class="base-input" placeholder="US" />
        <p class="form-hint">e.g. US, EU, asia-east1</p>
      </div>
      <p v-if="createError" style="font-size:12px;color:var(--danger);background:rgba(220,50,50,.08);border-radius:6px;padding:6px 10px;margin:0">{{ createError }}</p>
    </div>
    <template #footer>
      <button class="base-btn base-btn--ghost" @click="showCreate = false">Cancel</button>
      <button class="base-btn base-btn--primary" @click="doCreate" :disabled="!newBucketName.trim() || creating">
        {{ creating ? 'Creating…' : 'Create' }}
      </button>
    </template>
  </BaseModal>
</template>

<script setup>
import { ref, computed, watch, nextTick } from 'vue'
import ProviderIcon from '../ui/ProviderIcon.vue'
import BaseModal from '../ui/BaseModal.vue'
import { useConnections } from '../../composables/useConnections.js'
import { useToast } from '../../composables/useToast.js'
import { useConfirm } from '../../composables/useConfirm.js'

const props = defineProps({
  connections: { type: Array, default: () => [] },
})

const { listBuckets, createBucket, deleteBucket } = useConnections()
const toast = useToast()
const confirm = useConfirm()

const SUPPORTED_PROVIDERS = ['aws', 'alibaba', 'gcp', 'azure']

const supportedConnections = computed(() =>
  props.connections.filter(c => SUPPORTED_PROVIDERS.includes(c.provider))
)

const selected = ref(null)
const buckets = ref([])
const bucketsLoading = ref(false)
const bucketsError = ref('')

const showCreate = ref(false)
const newBucketName = ref('')
const newBucketRegion = ref('')
const creating = ref(false)
const createError = ref('')
const bucketNameInput = ref(null)

watch(showCreate, open => { if (open) nextTick(() => bucketNameInput.value?.focus()) })

function selectConn(c) {
  selected.value = c
  loadBuckets()
}

async function loadBuckets() {
  if (!selected.value) return
  bucketsLoading.value = true
  bucketsError.value = ''
  try {
    const data = await listBuckets(selected.value.provider, selected.value.id)
    buckets.value = Array.isArray(data) ? data : (data.containers ?? [])
  } catch (err) {
    bucketsError.value = err.message
  } finally {
    bucketsLoading.value = false
  }
}

async function doCreate() {
  const name = newBucketName.value.trim()
  if (!name || creating.value) return
  creating.value = true
  createError.value = ''
  try {
    const opts = {}
    if (newBucketRegion.value.trim()) opts.region = newBucketRegion.value.trim()
    await createBucket(selected.value.provider, selected.value.id, name, opts)
    toast.success(`Bucket "${name}" created.`)
    showCreate.value = false
    newBucketName.value = ''
    newBucketRegion.value = ''
    await loadBuckets()
  } catch (err) {
    createError.value = err.message
  } finally {
    creating.value = false
  }
}

async function confirmDelete(bucket) {
  const ok = await confirm.confirm(
    `Delete bucket "${bucket.name}"? The bucket must be empty. This cannot be undone.`,
    'Delete Bucket'
  )
  if (!ok) return
  try {
    await deleteBucket(selected.value.provider, selected.value.id, bucket.name)
    toast.success(`Bucket "${bucket.name}" deleted.`)
    await loadBuckets()
  } catch (err) {
    toast.error('Delete failed: ' + err.message)
  }
}

function formatDate(iso) {
  if (!iso) return '—'
  try { return new Date(iso).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' }) }
  catch { return '—' }
}
</script>
