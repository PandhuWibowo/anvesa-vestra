<template>
  <aside class="sidebar">
    <!-- Brand -->
    <div class="sidebar__brand">
      <div class="brand-icon">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M3 15a4 4 0 0 0 4 4h9a5 5 0 0 0 1.8-9.7 6 6 0 0 0-11.8-1A4 4 0 0 0 3 15z"/>
        </svg>
      </div>
      <div>
        <div class="brand-name"><span class="brand-anvesa">Anveesa</span> Vestra</div>
        <div class="brand-sub">Cloud storage manager</div>
      </div>
    </div>

    <!-- Scrollable body: connections + bookmarks + management nav -->
    <div class="sidebar__body">
      <!-- New connection -->
      <button class="btn-new-conn" @click="$emit('new-connection')">
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
          <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
        New Connection
      </button>

      <!-- Search -->
      <div v-if="connections.length > 0" class="sidebar-search">
        <svg class="sidebar-search__icon" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
        </svg>
        <input
          class="sidebar-search__input"
          v-model="query"
          placeholder="Filter connections…"
          aria-label="Filter connections"
        />
      </div>

      <!-- Provider filter chips -->
      <div class="prov-filter" v-if="availableProviders.length > 1">
        <button
          v-for="prov in availableProviders"
          :key="prov"
          class="prov-chip"
          :class="[`prov-chip--${prov}`, { 'prov-chip--active': filterProviders.has(prov) }]"
          @click="toggleFilter(prov)"
        >
          <ProviderIcon :provider="prov" :size="10" />
          {{ PROV_SHORT[prov] ?? prov }}
        </button>
      </div>

      <!-- Skeleton while loading -->
      <SkeletonLoader v-if="loading" :count="3" height="42px" />

      <template v-else>
        <!-- Section label -->
        <div v-if="filtered.length" class="section-label">
          {{ filtered.some(c => isPinned(c.provider, c.id)) ? 'Pinned · All' : 'Connections' }}
        </div>

        <!-- Empty -->
        <div v-if="!filtered.length && !connections.length" class="sidebar-empty">
          No connections yet.<br>Add your first bucket above.
        </div>

        <!-- Items -->
        <div
          v-for="c in filtered"
          :key="c.provider + '-' + c.id"
          class="conn-item"
          :class="{ 'is-active': activeConn?.id === c.id && activeConn?.provider === c.provider }"
          role="button"
          tabindex="0"
          @click="$emit('select', c)"
          @keydown.enter="$emit('select', c)"
          @keydown.space.prevent="$emit('select', c)"
        >
          <div class="conn-badge" :class="`conn-badge--${c.provider}`">
            <ProviderIcon :provider="c.provider" :size="11" />
          </div>
          <div class="conn-item__body">
            <div class="conn-item__name">{{ c.name }}</div>
            <div class="conn-item__bucket">{{ c.bucket }}</div>
          </div>
          <button class="conn-item__del" :class="{ 'is-pinned': isPinned(c.provider, c.id) }" @click.stop="togglePin(c.provider, c.id)" :title="isPinned(c.provider, c.id) ? 'Unpin' : 'Pin to top'">
            <svg width="11" height="11" viewBox="0 0 24 24" :fill="isPinned(c.provider, c.id) ? 'currentColor' : 'none'" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/></svg>
          </button>
          <button class="conn-item__del" @click.stop="$emit('edit', c)" title="Edit connection">
            <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
          </button>
          <button class="conn-item__del" @click.stop="$emit('delete', c.provider, c.id)" title="Delete connection">
            <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
          </button>
        </div>
      </template>

      <!-- Bookmarks section -->
      <template v-if="bookmarks.length">
        <div class="section-label section-label--bookmarks">
          <svg width="10" height="10" viewBox="0 0 24 24" fill="currentColor" stroke="none" style="opacity:.7"><path d="M19 21l-7-5-7 5V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2z"/></svg>
          Bookmarks
        </div>
        <div
          v-for="bm in bookmarks"
          :key="bm.provider + bm.id + bm.prefix"
          class="bookmark-item"
          :class="{ 'is-active': activeConn?.id === bm.id && activeConn?.provider === bm.provider && activePrefix === bm.prefix }"
          role="button" tabindex="0"
          @click="$emit('bookmark-navigate', bm)"
          @keydown.enter="$emit('bookmark-navigate', bm)"
          @keydown.space.prevent="$emit('bookmark-navigate', bm)"
          :title="`${bm.connName} / ${bm.prefix || bm.bucket}`"
        >
          <div class="conn-badge conn-badge--xs" :class="`conn-badge--${bm.provider}`"><ProviderIcon :provider="bm.provider" :size="9" /></div>
          <div class="bookmark-item__body">
            <div class="bookmark-item__label">{{ bm.label }}</div>
            <div class="bookmark-item__conn">{{ bm.connName }}</div>
          </div>
          <button class="conn-item__del" @click.stop="removeBookmark(bm.provider, bm.id, bm.prefix)" title="Remove bookmark">
            <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
          </button>
        </div>
      </template>

      <!-- ── Management nav (inside scroll) ── -->
      <div class="sidebar-nav-group">
        <button class="sidebar-nav-group__toggle" @click="navOpen = !navOpen">
          <span class="section-label" style="margin:0;cursor:pointer">Management</span>
          <svg :class="{ 'is-open': navOpen }" class="sidebar-nav-group__chevron" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="6 9 12 15 18 9"/></svg>
        </button>

        <div v-show="navOpen" class="sidebar-nav-group__items">
          <button class="nav-item" :class="{ 'is-active': activeView === 'dashboard' }" @click="$emit('navigate', 'dashboard')">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/></svg>
            Dashboard
          </button>
          <button class="nav-item" :class="{ 'is-active': activeView === 'search' }" @click="$emit('navigate', 'search')">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
            Search
          </button>
          <button class="nav-item" :class="{ 'is-active': activeView === 'shared' }" @click="$emit('navigate', 'shared')">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>
            Shared Links
          </button>
          <button class="nav-item" :class="{ 'is-active': activeView === 'jobs' }" @click="$emit('navigate', 'jobs')">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="7" width="20" height="14" rx="2" ry="2"/><path d="M16 21V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v16"/></svg>
            Jobs
          </button>
          <button class="nav-item" :class="{ 'is-active': activeView === 'sync' }" @click="$emit('navigate', 'sync')">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/></svg>
            Sync
          </button>
          <button class="nav-item" :class="{ 'is-active': activeView === 'audit' }" @click="$emit('navigate', 'audit')">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/></svg>
            Audit
          </button>

          <!-- Settings sub-group -->
          <div class="section-label" style="margin-top:8px">Settings</div>
          <button class="nav-item" :class="{ 'is-active': activeView === 'webhooks' }" @click="$emit('navigate', 'webhooks')">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"/><path d="M13.73 21a2 2 0 0 1-3.46 0"/></svg>
            Webhooks
          </button>
          <button class="nav-item" :class="{ 'is-active': activeView === 'notifications' }" @click="$emit('navigate', 'notifications')">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 17H2a3 3 0 0 0 3-3V9a7 7 0 0 1 14 0v5a3 3 0 0 0 3 3zm-8.27 4a2 2 0 0 1-3.46 0"/></svg>
            Notifications
          </button>
          <button class="nav-item" :class="{ 'is-active': activeView === 'users' }" @click="$emit('navigate', 'users')">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
            Users
          </button>

          <!-- Export / Import inline -->
          <div class="nav-item-pair">
            <button class="nav-item nav-item--half" @click="$emit('export-connections')" title="Export all connections">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
              Export
            </button>
            <button class="nav-item nav-item--half" @click="$emit('import-connections')" title="Import connections from file">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>
              Import
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Bottom toolbar: compact icon row -->
    <div class="sidebar__toolbar">
      <button class="tb-icon" :class="{ 'is-active': splitActive }" @click="$emit('split')" title="Split view">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2"/><line x1="12" y1="3" x2="12" y2="21"/></svg>
      </button>
      <button class="tb-icon" :class="{ 'is-active': activityActive }" @click="$emit('activity')" title="Activity log">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>
      </button>
      <button class="tb-icon" :class="{ 'is-active': docsActive }" @click="$emit('docs')" title="Documentation">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/><path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/></svg>
      </button>
      <button class="tb-icon" @click="toggleTheme" :title="isLight ? 'Dark mode' : 'Light mode'">
        <svg v-if="isLight" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>
        <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/></svg>
      </button>
      <div class="tb-sep"></div>
      <a class="tb-icon" href="https://github.com/PandhuWibowo/anveesa-vestra" target="_blank" rel="noopener" title="GitHub">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor"><path d="M12 .3a12 12 0 0 0-3.8 23.38c.6.12.83-.26.83-.57L9 21.07c-3.34.72-4.04-1.61-4.04-1.61-.55-1.39-1.34-1.76-1.34-1.76-1.08-.74.08-.73.08-.73 1.2.09 1.84 1.24 1.84 1.24 1.07 1.83 2.8 1.3 3.49 1 .1-.78.42-1.3.76-1.6-2.67-.31-5.47-1.34-5.47-5.93 0-1.31.47-2.38 1.24-3.22-.14-.3-.54-1.52.1-3.18 0 0 1-.32 3.3 1.23a11.5 11.5 0 0 1 6.02 0c2.28-1.55 3.29-1.23 3.29-1.23.64 1.66.24 2.88.12 3.18a4.65 4.65 0 0 1 1.23 3.22c0 4.61-2.8 5.62-5.48 5.92.42.36.81 1.1.81 2.22l-.01 3.29c0 .31.2.69.82.57A12 12 0 0 0 12 .3"/></svg>
      </a>
      <a class="tb-icon" href="https://anveesa.com" target="_blank" rel="noopener" title="anveesa.com">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>
      </a>
    </div>

    <!-- User bar (only when auth is enabled) -->
    <div v-if="authEnabled" class="sidebar__userbar">
      <div class="sidebar__userbar-info" v-if="username">
        <div class="sidebar__avatar">{{ username.charAt(0).toUpperCase() }}</div>
        <span class="sidebar__userbar-name">{{ username }}</span>
      </div>
      <button class="tb-icon" @click="$emit('logout')" title="Sign out">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><polyline points="16 17 21 12 16 7"/><line x1="21" y1="12" x2="9" y2="12"/></svg>
      </button>
    </div>
  </aside>
</template>

<script setup>
import { ref, computed } from 'vue'
import SkeletonLoader from '../ui/SkeletonLoader.vue'
import ProviderIcon   from '../ui/ProviderIcon.vue'
import { useTheme }     from '../../composables/useTheme.js'
import { usePins }      from '../../composables/usePins.js'
import { useBookmarks } from '../../composables/useBookmarks.js'

const PROV_SHORT = { gcp: 'GCS', aws: 'S3', huawei: 'OBS', alibaba: 'OSS', azure: 'Azure' }

const props = defineProps({
  connections:    { type: Array,   default: () => [] },
  loading:        { type: Boolean, default: false },
  activeConn:     { type: Object,  default: null },
  activePrefix:   { type: String,  default: '' },
  docsActive:     { type: Boolean, default: false },
  activityActive: { type: Boolean, default: false },
  splitActive:    { type: Boolean, default: false },
  activeView:     { type: String,  default: '' },
  username:       { type: String,  default: '' },
  authEnabled:    { type: Boolean, default: true },
})

defineEmits(['new-connection', 'select', 'edit', 'delete', 'docs', 'activity', 'split', 'bookmark-navigate', 'logout', 'navigate', 'export-connections', 'import-connections'])

const { isLight, toggleTheme }          = useTheme()
const { isPinned, togglePin }           = usePins()
const { bookmarks, removeBookmark }     = useBookmarks()

const query           = ref('')
const filterProviders = ref(new Set())
const navOpen         = ref(true)

const availableProviders = computed(() => {
  const seen = new Set()
  for (const c of props.connections) seen.add(c.provider)
  return [...seen]
})

function toggleFilter(prov) {
  const next = new Set(filterProviders.value)
  next.has(prov) ? next.delete(prov) : next.add(prov)
  filterProviders.value = next
}

const filtered = computed(() => {
  let list = props.connections
  if (filterProviders.value.size > 0)
    list = list.filter(c => filterProviders.value.has(c.provider))
  const q = query.value.toLowerCase().trim()
  if (q) list = list.filter(c => c.name.toLowerCase().includes(q) || c.bucket.toLowerCase().includes(q))
  return [...list].sort((a, b) => {
    const pa = isPinned(a.provider, a.id) ? 0 : 1
    const pb = isPinned(b.provider, b.id) ? 0 : 1
    return pa - pb
  })
})
</script>
