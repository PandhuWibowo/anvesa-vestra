<template>
  <div class="view-panel">
    <div class="view-panel__hd">
      <div>
        <h2 class="view-panel__title">Webhooks</h2>
        <p class="view-panel__sub">Manage webhook endpoints for event notifications</p>
      </div>
      <button class="icon-btn" @click="fetchWebhooks" title="Refresh">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/>
          <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
        </svg>
      </button>
    </div>

    <!-- Create form -->
    <div class="view-panel__body">
      <div class="dash-section">
        <h3 class="dash-section__title">Add Webhook</h3>
        <form @submit.prevent="handleCreate" class="webhook-form">
          <div class="webhook-form__row">
            <label>URL</label>
            <input v-model="newUrl" type="url" placeholder="https://example.com/webhook" required class="webhook-input" />
          </div>
          <div class="webhook-form__row">
            <label>Events</label>
            <div class="webhook-events">
              <label v-for="evt in allEvents" :key="evt" class="webhook-event-check">
                <input type="checkbox" :value="evt" v-model="newEvents" />
                {{ evt }}
              </label>
            </div>
          </div>
          <div class="webhook-form__row">
            <label>Secret (optional)</label>
            <input v-model="newSecret" type="text" placeholder="HMAC signing secret" class="webhook-input" />
          </div>
          <button type="submit" class="base-btn base-btn--primary" :disabled="!newUrl || newEvents.length === 0">
            Add Webhook
          </button>
        </form>
      </div>

      <!-- List -->
      <div class="dash-section">
        <h3 class="dash-section__title">Active Webhooks</h3>
        <div v-if="loading" class="view-panel__loading">
          <div class="base-btn__spinner" style="width:16px;height:16px"></div>
          Loading...
        </div>
        <div v-else-if="!webhooks.length" class="sidebar-empty" style="padding:16px 0">
          No webhooks configured yet.
        </div>
        <div v-else class="webhook-list">
          <div v-for="wh in webhooks" :key="wh.id" class="webhook-item">
            <div class="webhook-item__info">
              <div class="webhook-item__url">{{ wh.url }}</div>
              <div class="webhook-item__events">
                <span v-for="e in wh.events" :key="e" class="audit-action">{{ e }}</span>
              </div>
            </div>
            <button class="icon-btn icon-btn--danger" @click="handleDelete(wh.id)" title="Delete webhook">
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
import { ref, onMounted } from 'vue'
import { useWebhooks } from '../../composables/useWebhooks.js'
import { useToast } from '../../composables/useToast.js'

const { webhooks, loading, fetchWebhooks, createWebhook, deleteWebhook } = useWebhooks()
const toast = useToast()

const allEvents = ['upload', 'download', 'delete', 'transfer', 'share']
const newUrl = ref('')
const newEvents = ref([])
const newSecret = ref('')

async function handleCreate() {
  try {
    await createWebhook(newUrl.value, newEvents.value, newSecret.value)
    toast.success('Webhook created')
    newUrl.value = ''
    newEvents.value = []
    newSecret.value = ''
  } catch (err) {
    toast.error(err.message)
  }
}

async function handleDelete(id) {
  try {
    await deleteWebhook(id)
    toast.success('Webhook deleted')
  } catch (err) {
    toast.error(err.message)
  }
}

onMounted(fetchWebhooks)
</script>
