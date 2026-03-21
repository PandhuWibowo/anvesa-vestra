<template>
  <div class="docs-shell">

    <!-- Left nav -->
    <nav class="docs-nav">
      <div class="docs-nav__title">Documentation</div>
      <button
        v-for="p in pages"
        :key="p.id"
        class="docs-nav__item"
        :class="{ 'is-active': current === p.id }"
        @click="navigate(p.id)"
      >
        <span class="docs-nav__icon" v-html="p.icon" />
        {{ p.label }}
      </button>
    </nav>

    <!-- Content area — intercept internal .md link clicks -->
    <div class="docs-content" @click.capture="handleLinkClick">
      <div v-if="fetchError" class="docs-state docs-state--error">{{ fetchError }}</div>
      <article v-else class="docs-prose" v-html="rendered" />
    </div>

  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { marked } from 'marked'
import DOMPurify from 'dompurify'

// Static imports — bundled at build time, no fetch needed
import indexMd         from '../../../../docs/index.md?raw'
import gettingStarted  from '../../../../docs/getting-started.md?raw'
import authentication  from '../../../../docs/authentication.md?raw'
import connections     from '../../../../docs/connections.md?raw'
import browser         from '../../../../docs/browser.md?raw'
import filePreview     from '../../../../docs/file-preview.md?raw'
import management      from '../../../../docs/management.md?raw'
import apiReference    from '../../../../docs/api-reference.md?raw'
import deployment      from '../../../../docs/deployment.md?raw'
import contributing    from '../../../../docs/contributing.md?raw'

const docMap = {
  'index':           indexMd,
  'getting-started': gettingStarted,
  'authentication':  authentication,
  'connections':     connections,
  'browser':         browser,
  'file-preview':    filePreview,
  'management':      management,
  'api-reference':   apiReference,
  'deployment':      deployment,
  'contributing':    contributing,
}

const pages = [
  {
    id: 'index',
    label: 'Overview',
    icon: '<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><polyline points="9 22 9 12 15 12 15 22"/></svg>',
  },
  {
    id: 'getting-started',
    label: 'Getting Started',
    icon: '<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="5 3 19 12 5 21 5 3"/></svg>',
  },
  {
    id: 'authentication',
    label: 'Authentication',
    icon: '<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>',
  },
  {
    id: 'connections',
    label: 'Connections',
    icon: '<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 15a4 4 0 0 0 4 4h9a5 5 0 0 0 1.8-9.7 6 6 0 0 0-11.8-1A4 4 0 0 0 3 15z"/></svg>',
  },
  {
    id: 'browser',
    label: 'File Browser',
    icon: '<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>',
  },
  {
    id: 'file-preview',
    label: 'File Preview',
    icon: '<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>',
  },
  {
    id: 'management',
    label: 'Management',
    icon: '<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 20V10"/><path d="M18 20V4"/><path d="M6 20v-4"/></svg>',
  },
  {
    id: 'api-reference',
    label: 'API Reference',
    icon: '<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg>',
  },
  {
    id: 'deployment',
    label: 'Deployment',
    icon: '<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="3" width="20" height="14" rx="2"/><path d="M8 21h8M12 17v4"/></svg>',
  },
  {
    id: 'contributing',
    label: 'Contributing',
    icon: '<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="16"/><line x1="8" y1="12" x2="16" y2="12"/></svg>',
  },
]

const current    = ref('index')
const markdown   = ref(docMap['index'])
const fetchError = ref('')

const rendered = computed(() =>
  markdown.value ? DOMPurify.sanitize(marked.parse(markdown.value)) : ''
)

function navigate(page) {
  const content = docMap[page]
  if (!content) {
    fetchError.value = `Unknown page: ${page}`
    return
  }
  current.value    = page
  markdown.value   = content
  fetchError.value = ''
}

// Intercept clicks on rendered anchor tags.
// If the href points to a relative .md doc, navigate in-app instead.
function handleLinkClick(e) {
  const a = e.target.closest('a')
  if (!a) return
  const href = a.getAttribute('href')
  if (!href) return
  const match = href.match(/(?:\.\/)?([a-z-]+)\.md$/)
  if (match) {
    e.preventDefault()
    navigate(match[1])
  } else if (href.startsWith('http')) {
    e.preventDefault()
    window.open(href, '_blank', 'noopener')
  }
}
</script>
