<template>
  <div class="view-panel">
    <div class="view-panel__hd">
      <div>
        <h2 class="view-panel__title">Dashboard</h2>
        <p class="view-panel__sub">Platform overview and analytics</p>
      </div>
      <button class="icon-btn" @click="fetchAnalytics" title="Refresh" :disabled="loading">
        <svg :class="{ 'spin': loading }" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/>
          <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
        </svg>
      </button>
    </div>

    <div v-if="loading && !data" class="view-panel__loading">
      <div class="base-btn__spinner" style="width:16px;height:16px"></div>
      Loading analytics...
    </div>

    <div v-else-if="error" class="status-notice status-notice--error" style="margin:16px 24px">
      {{ error }}
    </div>

    <div v-else-if="data" class="view-panel__body dash-body">

      <!-- Hero stats row -->
      <div class="dash-hero">
        <div class="dash-hero-card">
          <div class="dash-hero-card__icon dash-hero-card__icon--conn">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M3 15a4 4 0 0 0 4 4h9a5 5 0 0 0 1.8-9.7 6 6 0 0 0-11.8-1A4 4 0 0 0 3 15z"/>
            </svg>
          </div>
          <div class="dash-hero-card__info">
            <div class="dash-hero-card__val">{{ data.connections?.total || 0 }}</div>
            <div class="dash-hero-card__label">Total Connections</div>
          </div>
        </div>

        <div class="dash-hero-card">
          <div class="dash-hero-card__icon dash-hero-card__icon--activity">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>
            </svg>
          </div>
          <div class="dash-hero-card__info">
            <div class="dash-hero-card__val">{{ totalActivity }}</div>
            <div class="dash-hero-card__label">Actions (24h)</div>
          </div>
        </div>

        <div class="dash-hero-card">
          <div class="dash-hero-card__icon dash-hero-card__icon--links">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/>
              <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>
            </svg>
          </div>
          <div class="dash-hero-card__info">
            <div class="dash-hero-card__val">{{ data.shared_links?.active || 0 }}</div>
            <div class="dash-hero-card__label">Active Links</div>
          </div>
        </div>

        <div class="dash-hero-card">
          <div class="dash-hero-card__icon dash-hero-card__icon--jobs">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <rect x="2" y="7" width="20" height="14" rx="2" ry="2"/><path d="M16 21V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v16"/>
            </svg>
          </div>
          <div class="dash-hero-card__info">
            <div class="dash-hero-card__val">{{ totalJobs }}</div>
            <div class="dash-hero-card__label">Total Jobs</div>
          </div>
        </div>
      </div>

      <!-- Two-column layout -->
      <div class="dash-grid">

        <!-- Left: Activity + Jobs -->
        <div class="dash-col">

          <!-- Activity 24h -->
          <div class="dash-panel">
            <div class="dash-panel__hd">
              <h3 class="dash-panel__title">Activity (24h)</h3>
            </div>
            <div class="dash-panel__body">
              <div class="dash-stat-list">
                <div class="dash-stat-row" v-for="(count, action) in data.activity_24h" :key="action">
                  <div class="dash-stat-row__left">
                    <span class="dash-stat-dot" :class="`dash-stat-dot--${action}`"></span>
                    <span class="dash-stat-row__label">{{ action }}</span>
                  </div>
                  <span class="dash-stat-row__val">{{ count }}</span>
                </div>
                <div v-if="!data.activity_24h || Object.keys(data.activity_24h).length === 0" class="dash-panel__empty">
                  No activity recorded
                </div>
              </div>
            </div>
          </div>

          <!-- Background Jobs -->
          <div class="dash-panel">
            <div class="dash-panel__hd">
              <h3 class="dash-panel__title">Background Jobs</h3>
            </div>
            <div class="dash-panel__body">
              <div class="dash-stat-list">
                <div class="dash-stat-row" v-for="(count, status) in data.jobs" :key="status">
                  <div class="dash-stat-row__left">
                    <span class="dash-stat-dot" :class="`dash-stat-dot--${status}`"></span>
                    <span class="dash-stat-row__label">{{ status }}</span>
                  </div>
                  <span class="dash-stat-row__val">{{ count }}</span>
                </div>
                <div v-if="!data.jobs || Object.keys(data.jobs).length === 0" class="dash-panel__empty">
                  No jobs found
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Right: Providers + Shared Links -->
        <div class="dash-col">

          <!-- Providers -->
          <div class="dash-panel">
            <div class="dash-panel__hd">
              <h3 class="dash-panel__title">Connections by Provider</h3>
            </div>
            <div class="dash-panel__body">
              <div class="dash-provider-list">
                <div class="dash-provider-row" v-for="(count, provider) in allProviders" :key="provider">
                  <div class="dash-provider-row__left">
                    <div class="dash-provider-badge" :class="`dash-provider-badge--${provider}`">
                      <ProviderIcon :provider="provider" :size="12" />
                    </div>
                    <span class="dash-provider-row__name">{{ PROV_NAMES[provider] || provider }}</span>
                  </div>
                  <div class="dash-provider-row__right">
                    <span class="dash-provider-row__count">{{ count }}</span>
                    <div class="dash-provider-bar">
                      <div
                        class="dash-provider-bar__fill"
                        :class="`dash-provider-bar__fill--${provider}`"
                        :style="{ width: barWidth(count) }"
                      ></div>
                    </div>
                  </div>
                </div>
                <div v-if="Object.keys(allProviders).length === 0" class="dash-panel__empty">
                  No connections yet
                </div>
              </div>
            </div>
          </div>

          <!-- Shared Links -->
          <div class="dash-panel">
            <div class="dash-panel__hd">
              <h3 class="dash-panel__title">Shared Links</h3>
            </div>
            <div class="dash-panel__body">
              <div class="dash-shared-grid">
                <div class="dash-shared-stat">
                  <div class="dash-shared-stat__val">{{ data.shared_links?.active || 0 }}</div>
                  <div class="dash-shared-stat__label">Active</div>
                </div>
                <div class="dash-shared-divider"></div>
                <div class="dash-shared-stat">
                  <div class="dash-shared-stat__val">{{ data.shared_links?.total_downloads || 0 }}</div>
                  <div class="dash-shared-stat__label">Downloads</div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Connection details table -->
      <div v-if="data.connection_details?.length" class="dash-panel dash-panel--full">
        <div class="dash-panel__hd">
          <h3 class="dash-panel__title">Connection Details</h3>
          <span class="dash-panel__badge">{{ data.connection_details.length }}</span>
        </div>
        <div class="dash-panel__body dash-panel__body--flush">
          <DragTable
            :columns="connColumns"
            :rows="data.connection_details"
            :row-key="c => c.provider + '-' + c.id"
            :resizable-columns="true"
            :reorderable-columns="true"
            striped
          >
            <template #cell-provider="{ row }">
              <div class="dash-table__provider">
                <div class="dash-provider-badge dash-provider-badge--sm" :class="`dash-provider-badge--${row.provider}`">
                  <ProviderIcon :provider="row.provider" :size="10" />
                </div>
                {{ PROV_NAMES[row.provider] || row.provider }}
              </div>
            </template>
            <template #cell-name="{ row }">
              <span class="dash-table__name">{{ row.name }}</span>
            </template>
            <template #cell-bucket="{ row }">
              <code class="dash-table__bucket">{{ row.bucket }}</code>
            </template>
          </DragTable>
        </div>
      </div>

    </div>
  </div>
</template>

<script setup>
import { computed, onMounted } from 'vue'
import DragTable from '../ui/DragTable.vue'
import { useAnalytics } from '../../composables/useAnalytics.js'
import ProviderIcon from '../ui/ProviderIcon.vue'

const PROV_NAMES = {
  gcp: 'Google Cloud', aws: 'Amazon S3', huawei: 'Huawei OBS',
  alibaba: 'Alibaba OSS', azure: 'Azure Blob', gdrive: 'Google Drive',
}

const ALL_PROVIDERS = ['gcp', 'aws', 'azure', 'alibaba', 'huawei', 'gdrive']

const connColumns = [
  { key: 'provider', label: 'Provider', sortable: true, width: 160 },
  { key: 'name', label: 'Name', sortable: true },
  { key: 'bucket', label: 'Bucket', sortable: true },
]

const { data, loading, error, fetchAnalytics } = useAnalytics()

const allProviders = computed(() => {
  if (!data.value?.connections) return {}
  const { total, ...providers } = data.value.connections
  const result = {}
  for (const p of ALL_PROVIDERS) {
    const v = providers[p]
    if (typeof v === 'number' && v > 0) result[p] = v
  }
  return result
})

const totalActivity = computed(() => {
  if (!data.value?.activity_24h) return 0
  return Object.values(data.value.activity_24h).reduce((a, b) => a + b, 0)
})

const totalJobs = computed(() => {
  if (!data.value?.jobs) return 0
  return Object.values(data.value.jobs).reduce((a, b) => a + b, 0)
})

function barWidth(count) {
  const total = data.value?.connections?.total || 1
  return Math.max(4, (count / total) * 100) + '%'
}

onMounted(fetchAnalytics)
</script>
