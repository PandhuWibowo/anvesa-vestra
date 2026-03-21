<template>
  <div class="browser-view" ref="rootEl"
       @dragover.prevent="onDragOver"
       @dragleave.self="onDragLeave"
       @drop.prevent="onDrop">

    <!-- ── Header ──────────────────────────────────────────────── -->
    <div class="browser-hd">
      <div class="browser-hd__left">
        <div class="browser-prov-icon" :class="`browser-prov-icon--${conn.provider}`">
          <ProviderIcon :provider="conn.provider" :size="16" />
        </div>
        <div style="min-width:0">
          <div class="browser-conn-name">
            {{ conn.name }}
            <BaseBadge :provider="conn.provider" />
          </div>
          <div class="browser-conn-bucket">{{ conn.bucket }}</div>
          <div class="breadcrumbs" v-if="currentPrefix">
            <button class="bread-item" @click="navigateTo('')">root</button>
            <template v-for="(crumb, i) in breadcrumbs" :key="i">
              <span class="bread-sep">/</span>
              <button
                class="bread-item"
                :style="i === breadcrumbs.length - 1 ? 'color:var(--text-2);cursor:default' : ''"
                @click="i < breadcrumbs.length - 1 && navigateTo(crumb.prefix)"
              >{{ crumb.label }}</button>
            </template>
          </div>
        </div>
      </div>

      <div class="browser-hd__actions">
        <!-- Bookmark this location -->
        <button class="icon-btn" :class="{ 'is-bookmarked': isCurrentBookmarked }" @click="toggleCurrentBookmark" :title="isCurrentBookmarked ? 'Remove bookmark' : 'Bookmark this location'">
          <svg width="13" height="13" viewBox="0 0 24 24" :fill="isCurrentBookmarked ? 'currentColor' : 'none'" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M19 21l-7-5-7 5V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2z"/>
          </svg>
        </button>
        <button class="icon-btn" :style="showStats ? 'background:var(--accent-bg);color:var(--accent);border-color:var(--accent-ring)' : ''" @click="toggleStats" title="Bucket stats">
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="18" y1="20" x2="18" y2="10"/><line x1="12" y1="20" x2="12" y2="4"/><line x1="6" y1="20" x2="6" y2="14"/>
          </svg>
        </button>
        <button class="icon-btn" :disabled="loading" @click="refresh" title="Refresh (R)">
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" :style="loading ? 'animation:spin .6s linear infinite' : ''">
            <polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/>
            <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
          </svg>
        </button>
        <button class="icon-btn danger" @click="$emit('delete')" title="Delete connection">
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/>
            <path d="M10 11v6"/><path d="M14 11v6"/><path d="M9 6V4h6v2"/>
          </svg>
        </button>
      </div>
    </div>

    <!-- ── Stats bar ────────────────────────────────────────────── -->
    <transition name="slide-down">
      <div v-if="showStats" class="stats-bar">
        <template v-if="statsLoading">
          <div class="stat-item">
            <div class="skeleton-item" style="width:60px;height:22px;border-radius:4px"></div>
            <span class="stat-lbl">loading…</span>
          </div>
        </template>
        <template v-else-if="stats">
          <div class="stat-item">
            <span class="stat-val">{{ stats.object_count.toLocaleString() }}</span>
            <span class="stat-lbl">{{ stats.truncated ? 'objects (est.)' : 'objects' }}</span>
          </div>
          <div class="stat-item">
            <span class="stat-val">{{ formatSize(stats.total_size) }}</span>
            <span class="stat-lbl">total size</span>
          </div>
        </template>
        <template v-else-if="statsError">
          <span style="font-size:12px;color:var(--muted)">{{ statsError }}</span>
        </template>
      </div>
    </transition>

    <!-- ── Toolbar ──────────────────────────────────────────────── -->
    <div class="browser-toolbar">
      <div class="search-field">
        <span class="search-field__icon">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
          </svg>
        </span>
        <input ref="searchInput" class="search-field__input" v-model="searchQuery" placeholder="Search files… (/)" @keydown.escape.stop="searchQuery = ''" />
        <button v-if="searchQuery" class="search-field__clear" @click="searchQuery = ''">×</button>
      </div>

      <div class="toolbar-spacer"></div>

      <!-- View toggle -->
      <button class="icon-btn" @click="toggleView" :title="viewMode === 'table' ? 'Switch to grid view' : 'Switch to table view'">
        <svg v-if="viewMode === 'table'" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/>
        </svg>
        <svg v-else width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/>
          <line x1="3" y1="6" x2="3.01" y2="6"/><line x1="3" y1="12" x2="3.01" y2="12"/><line x1="3" y1="18" x2="3.01" y2="18"/>
        </svg>
      </button>

      <!-- New folder -->
      <button class="icon-btn" @click="showFolderModal = true" title="New folder">
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
          <line x1="12" y1="11" x2="12" y2="17"/><line x1="9" y1="14" x2="15" y2="14"/>
        </svg>
      </button>

      <!-- Upload files -->
      <label class="upload-label" title="Upload files">
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <polyline points="16 16 12 12 8 16"/><line x1="12" y1="12" x2="12" y2="21"/>
          <path d="M20.39 18.39A5 5 0 0 0 18 9h-1.26A8 8 0 1 0 3 16.3"/>
        </svg>
        Upload
        <input type="file" multiple style="display:none" @change="onFileInput" />
      </label>

      <!-- Upload folder -->
      <label class="upload-label" title="Upload folder (preserves subfolders)">
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
          <polyline points="16 16 12 12 8 16"/><line x1="12" y1="12" x2="12" y2="19"/>
        </svg>
        Folder
        <input ref="folderUploadInput" type="file" webkitdirectory style="display:none" @change="onFolderInput" />
      </label>
    </div>

    <!-- ── Upload queue ─────────────────────────────────────────── -->
    <transition name="slide-down">
      <div v-if="uploadQueue.length" class="upload-queue">
        <div v-for="(item, i) in uploadQueue" :key="i" class="upload-queue-item">
          <div class="upload-item-row">
            <span class="upload-item-name">{{ item.name }}</span>
            <span class="upload-item-status" :class="{ done: item.done, error: item.error }">
              {{ item.error ? '✗' : item.done ? '✓' : `${Math.round(item.progress * 100)}%` }}
            </span>
            <button v-if="item.error && item.file" class="upload-retry-btn" @click="retryUpload(i)" title="Retry">↻</button>
          </div>
          <div class="progress-bar">
            <div class="progress-fill"
                 :class="{ 'progress-fill--error': item.error }"
                 :style="`width:${item.progress * 100}%;transition:width .1s`" />
          </div>
        </div>
      </div>
    </transition>

    <!-- ── Selection action bar ─────────────────────────────────── -->
    <transition name="slide-down">
      <div v-if="selected.size > 0" class="selection-bar">
        <span class="selection-bar__count">{{ selected.size }} selected</span>
        <button class="base-btn base-btn--ghost" style="font-size:12px;padding:5px 10px" @click="bulkDownload" :disabled="bulkWorking">
          Download all
        </button>
        <button class="base-btn base-btn--ghost" style="font-size:12px;padding:5px 10px" @click="downloadZip('', [...selected])" :disabled="zipping || bulkWorking">
          {{ zipping ? 'Zipping…' : 'Download as Zip' }}
        </button>
        <button class="base-btn base-btn--danger" style="font-size:12px;padding:5px 10px;border:1px solid var(--danger)" @click="bulkDelete" :disabled="bulkWorking">
          Delete all
        </button>
        <button class="icon-btn" style="width:26px;height:26px;margin-left:auto" title="Deselect all" @click="selected.clear(); selected = new Set(selected)">
          <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
            <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
          </svg>
        </button>
      </div>
    </transition>

    <!-- ── Body ─────────────────────────────────────────────────── -->
    <div class="browser-body" ref="bodyEl">

      <!-- Drag overlay -->
      <div v-if="isDragging" class="drop-overlay">
        <div class="drop-overlay__inner">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="margin-bottom:8px">
            <polyline points="16 16 12 12 8 16"/><line x1="12" y1="12" x2="12" y2="21"/>
            <path d="M20.39 18.39A5 5 0 0 0 18 9h-1.26A8 8 0 1 0 3 16.3"/>
          </svg>
          {{ isCrossTarget ? 'Drop to copy here' : 'Drop to upload here' }}
        </div>
      </div>

      <!-- Loading skeleton -->
      <table v-if="loading && entries.length === 0" class="file-table">
        <thead><tr><th class="col-check"></th><th>Name</th><th>Size</th><th>Modified</th><th></th></tr></thead>
        <tbody>
          <tr v-for="i in 10" :key="i">
            <td class="col-check"></td>
            <td colspan="4"><div class="skeleton-item" :style="`height:15px;border-radius:3px;width:${30 + (i * 37 % 55)}%`"></div></td>
          </tr>
        </tbody>
      </table>

      <!-- Browse error -->
      <div v-else-if="browseError && entries.length === 0" class="empty-state">
        <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" style="opacity:.4;margin-bottom:6px">
          <circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/>
        </svg>
        <p style="font-size:13px;color:var(--text-2);max-width:320px">{{ browseError }}</p>
        <button class="base-btn base-btn--ghost" @click="refresh" style="font-size:12px;padding:6px 12px;margin-top:4px">Retry</button>
      </div>

      <!-- Empty -->
      <div v-else-if="!loading && filteredEntries.length === 0 && !nextPageToken" class="empty-state">
        <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" style="opacity:.4;margin-bottom:6px">
          <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
        </svg>
        <p>{{ searchQuery ? 'No files match your search.' : 'This folder is empty.' }}</p>
        <button v-if="searchQuery" class="base-btn base-btn--ghost" @click="searchQuery = ''" style="font-size:12px;padding:6px 12px;margin-top:4px">Clear search</button>
      </div>

      <!-- Grid view -->
      <div v-else-if="viewMode === 'grid'" class="file-grid">
        <div
          v-for="(entry, entryIdx) in filteredEntries"
          :key="entry.name"
          class="file-card"
          :class="{ 'is-dir': entry.type === 'dir', 'is-selected': selected.has(entry.name), 'is-focused': entryIdx === focusedIdx }"
          @mouseenter="focusedIdx = entryIdx"
          @click="entry.type === 'dir' ? navigateTo(entry.name) : openPreview(entry)"
        >
          <div class="file-card__thumb">
            <img v-if="isImage(entry) && entry.url" :src="entry.url" class="file-card__img" :alt="entry.display || 'thumbnail'" />
            <svg v-else-if="entry.type === 'dir'" width="28" height="28" viewBox="0 0 24 24" fill="currentColor" stroke="none" style="color:var(--aws);opacity:.8">
              <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
            </svg>
            <!-- Video -->
            <svg v-else-if="isVideo(entry)" width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" style="color:var(--accent)">
              <polygon points="23 7 16 12 23 17 23 7"/><rect x="1" y="5" width="15" height="14" rx="2" ry="2"/>
            </svg>
            <!-- Audio -->
            <svg v-else-if="isAudio(entry)" width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" style="color:#e879f9">
              <path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/>
            </svg>
            <!-- PDF -->
            <svg v-else-if="isPdf(entry)" width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" style="color:#ef4444">
              <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/>
            </svg>
            <!-- Code -->
            <svg v-else-if="isCode(entry) || isConfig(entry)" width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" style="color:#22d3ee">
              <polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/>
            </svg>
            <!-- Archive -->
            <svg v-else-if="isArchive(entry)" width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" style="color:#f59e0b">
              <path d="M21 8v13H3V8"/><path d="M23 3H1v5h22V3z"/><path d="M10 12h4"/>
            </svg>
            <!-- Default file -->
            <svg v-else width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" style="color:var(--muted)">
              <path d="M13 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z"/><polyline points="13 2 13 9 20 9"/>
            </svg>
            <!-- Checkbox for files -->
            <input v-if="entry.type === 'file'" type="checkbox"
              class="file-card__check"
              :checked="selected.has(entry.name)"
              @click.stop="toggleSelect(entry.name)" />
          </div>
          <div class="file-card__name" :title="entry.display">{{ entry.display }}</div>
          <div class="file-card__meta">{{ entry.type === 'dir' ? '—' : formatSize(entry.size) }}</div>
        </div>
      </div>

      <!-- File table -->
      <table v-else class="file-table">
        <thead>
          <tr>
            <th class="col-check">
              <input
                type="checkbox"
                :checked="allFilesSelected"
                :indeterminate="someFilesSelected && !allFilesSelected"
                @change="toggleSelectAll"
                title="Select all files"
              />
            </th>
            <th class="sortable" @click="cycleSort('name')">
              Name <span class="sort-indicator" :class="{active: sortKey==='name'}">{{ sortKey==='name' ? (sortDir==='asc'?'↑':'↓') : '↕' }}</span>
            </th>
            <th class="sortable" @click="cycleSort('size')">
              Size <span class="sort-indicator" :class="{active: sortKey==='size'}">{{ sortKey==='size' ? (sortDir==='asc'?'↑':'↓') : '↕' }}</span>
            </th>
            <th class="sortable" @click="cycleSort('date')">
              Modified <span class="sort-indicator" :class="{active: sortKey==='date'}">{{ sortKey==='date' ? (sortDir==='asc'?'↑':'↓') : '↕' }}</span>
            </th>
            <th style="width:198px"></th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="(entry, entryIdx) in filteredEntries"
            :key="entry.name"
            class="file-row"
            :class="{ 'is-dir': entry.type === 'dir', 'is-selected': selected.has(entry.name), 'is-focused': entryIdx === focusedIdx }"
            @mouseenter="focusedIdx = entryIdx"
            :draggable="entry.type === 'file'"
            @dragstart="onRowDragStart($event, entry)"
            @dragend="onRowDragEnd"
          >
            <td class="col-check">
              <input v-if="entry.type === 'file'" type="checkbox" :checked="selected.has(entry.name)" @change="toggleSelect(entry.name)" />
            </td>
            <td>
              <div class="file-name" :style="entry.type === 'dir' ? 'cursor:pointer' : ''" @click="entry.type === 'dir' && navigateTo(entry.name)">
                <svg v-if="entry.type === 'dir'" class="file-icon" width="13" height="13" viewBox="0 0 24 24" fill="currentColor" stroke="none" style="color:var(--aws)">
                  <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z" opacity=".8"/>
                </svg>
                <!-- Image -->
                <svg v-else-if="isImage(entry)" class="file-icon file-icon--image" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/><circle cx="8.5" cy="8.5" r="1.5"/><polyline points="21 15 16 10 5 21"/>
                </svg>
                <!-- Video -->
                <svg v-else-if="isVideo(entry)" class="file-icon file-icon--video" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <polygon points="23 7 16 12 23 17 23 7"/><rect x="1" y="5" width="15" height="14" rx="2" ry="2"/>
                </svg>
                <!-- Audio -->
                <svg v-else-if="isAudio(entry)" class="file-icon file-icon--audio" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/>
                </svg>
                <!-- PDF -->
                <svg v-else-if="isPdf(entry)" class="file-icon file-icon--pdf" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/>
                </svg>
                <!-- Code -->
                <svg v-else-if="isCode(entry) || isConfig(entry)" class="file-icon file-icon--code" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/>
                </svg>
                <!-- Archive -->
                <svg v-else-if="isArchive(entry)" class="file-icon file-icon--archive" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M21 8v13H3V8"/><path d="M23 3H1v5h22V3z"/><path d="M10 12h4"/>
                </svg>
                <!-- Default file -->
                <svg v-else class="file-icon" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M13 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z"/><polyline points="13 2 13 9 20 9"/>
                </svg>
                {{ entry.display }}
              </div>
            </td>
            <td class="file-size">{{ entry.type === 'dir' ? '—' : formatSize(entry.size) }}</td>
            <td class="file-date">{{ entry.type === 'dir' ? '—' : formatDate(entry.updated) }}</td>
            <td class="file-actions">
              <!-- Copy path -->
              <button class="row-btn" @click.stop="copyPath(entry)" title="Copy storage path (s3://…)">
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
                </svg>
              </button>
              <!-- Copy public URL (file only) -->
              <button v-if="entry.type === 'file'" class="row-btn" @click.stop="copyPublicUrl(entry)" title="Copy link (presigned URL)">
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/>
                  <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>
                </svg>
              </button>
              <!-- Folder actions -->
              <template v-if="entry.type === 'dir'">
                <!-- Download folder as zip -->
                <button class="row-btn" @click.stop="downloadZip(entry.name, [])" :disabled="zipping" title="Download folder as Zip">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/>
                  </svg>
                </button>
                <!-- Delete folder -->
                <button class="row-btn danger" @click.stop="confirmDeleteFolder(entry)" title="Delete folder">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/>
                    <path d="M10 11v6"/><path d="M14 11v6"/><path d="M9 6V4h6v2"/>
                  </svg>
                </button>
              </template>
              <template v-if="entry.type === 'file'">
                <!-- Rename -->
                <button class="row-btn" @click.stop="openRename(entry)" title="Rename / Move">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                    <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
                  </svg>
                </button>
                <!-- Transfer -->
                <button class="row-btn" @click.stop="openTransfer(entry)" title="Transfer to another connection">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/>
                  </svg>
                </button>
                <!-- Download -->
                <button class="row-btn" @click.stop="download(entry)" title="Download">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/>
                  </svg>
                </button>
                <!-- Preview / info -->
                <button class="row-btn" @click.stop="openPreview(entry)" title="Preview">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/>
                  </svg>
                </button>
                <!-- Metadata -->
                <button class="row-btn" @click.stop="openMeta(entry)" title="Metadata">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/>
                  </svg>
                </button>
                <!-- Generate shareable link -->
                <button class="row-btn" @click.stop="openPresign(entry)" title="Generate shareable link">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <circle cx="18" cy="5" r="3"/><circle cx="6" cy="12" r="3"/><circle cx="18" cy="19" r="3"/>
                    <line x1="8.59" y1="13.51" x2="15.42" y2="17.49"/><line x1="15.41" y1="6.51" x2="8.59" y2="10.49"/>
                  </svg>
                </button>
                <!-- CLI command -->
                <button class="row-btn" @click.stop="openCli(entry)" title="Copy CLI command">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/>
                  </svg>
                </button>
                <!-- Delete -->
                <button class="row-btn danger" @click.stop="confirmDelete(entry)" title="Delete">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/>
                    <path d="M10 11v6"/><path d="M14 11v6"/><path d="M9 6V4h6v2"/>
                  </svg>
                </button>
              </template>
            </td>
          </tr>

          <!-- Infinite scroll sentinel -->
          <tr v-if="nextPageToken" ref="sentinel">
            <td colspan="5" style="padding:12px 22px;text-align:center;font-size:12px;color:var(--muted)">
              <span v-if="loadingMore">Loading more…</span>
            </td>
          </tr>
        </tbody>
      </table>
      <!-- Keyboard hints -->
      <div class="kb-hints">
        <span><kbd>j</kbd><kbd>k</kbd> navigate</span>
        <span><kbd>↵</kbd> open</span>
        <span><kbd>Space</kbd> preview</span>
        <span><kbd>d</kbd> download</span>
        <span><kbd>Del</kbd> delete</span>
        <span><kbd>/</kbd> search</span>
        <span><kbd>r</kbd> refresh</span>
        <span><kbd>f</kbd> fullscreen</span>
        <span><kbd>⌫</kbd> up</span>
      </div>
    </div>

    <!-- ── Preview panel ────────────────────────────────────────── -->
    <transition name="slide-right">
      <div v-if="previewEntry && !metaEntry" class="preview-panel" :class="{ 'preview-panel--fullscreen': previewFullscreen }">
        <div class="preview-hd">
          <span class="preview-hd__name">{{ previewEntry.display }}</span>
          <div style="display:flex;align-items:center;gap:4px">
            <button class="preview-hd-btn" @click="togglePreviewFullscreen" :title="previewFullscreen ? 'Exit fullscreen (f)' : 'Fullscreen (f)'">
              <svg v-if="!previewFullscreen" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <polyline points="15 3 21 3 21 9"/><polyline points="9 21 3 21 3 15"/><line x1="21" y1="3" x2="14" y2="10"/><line x1="3" y1="21" x2="10" y2="14"/>
              </svg>
              <svg v-else width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <polyline points="4 14 10 14 10 20"/><polyline points="20 10 14 10 14 4"/><line x1="14" y1="10" x2="21" y2="3"/><line x1="3" y1="21" x2="10" y2="14"/>
              </svg>
            </button>
            <button class="preview-close" @click="closePreview">×</button>
          </div>
        </div>

        <!-- Preview toolbar -->
        <div class="preview-toolbar" v-if="!previewLoading">
          <!-- Image controls -->
          <template v-if="isImage(previewEntry) && previewUrl && !previewLoadError">
            <button class="preview-tb-btn" @click="previewDoZoomOut" title="Zoom out">
              <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="5" y1="12" x2="19" y2="12"/></svg>
            </button>
            <span class="preview-tb-zoom">{{ previewZoomIsFit ? 'Fit' : Math.round(previewZoom * 100) + '%' }}</span>
            <button class="preview-tb-btn" @click="previewDoZoomIn" title="Zoom in">
              <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
            </button>
            <div class="preview-tb-sep"></div>
            <button class="preview-tb-btn" :class="{ active: previewZoomIsFit }" @click="previewDoZoomFit" title="Fit to panel">Fit</button>
            <button class="preview-tb-btn" @click="previewDoZoomReset" title="Actual size (1:1)">1:1</button>
          </template>

          <!-- Text/Code controls -->
          <template v-if="(isCode(previewEntry) || isConfig(previewEntry) || isPlainText(previewEntry) || isJson(previewEntry)) && previewContent">
            <button class="preview-tb-btn" :class="{ active: previewShowLineNumbers }" @click="previewShowLineNumbers = !previewShowLineNumbers" title="Line numbers">
              <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <line x1="4" y1="6" x2="4.01" y2="6"/><line x1="4" y1="12" x2="4.01" y2="12"/><line x1="4" y1="18" x2="4.01" y2="18"/>
                <line x1="10" y1="6" x2="21" y2="6"/><line x1="10" y1="12" x2="21" y2="12"/><line x1="10" y1="18" x2="21" y2="18"/>
              </svg>
            </button>
            <button class="preview-tb-btn" :class="{ active: previewWordWrap }" @click="previewWordWrap = !previewWordWrap" title="Word wrap">
              <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M3 6h18"/><path d="M3 12h15a3 3 0 1 1 0 6h-4"/><polyline points="16 16 14 18 16 20"/><path d="M3 18h7"/>
              </svg>
            </button>
            <span class="preview-tb-info" v-if="previewTextAllLines.length">{{ previewLineProgress }}</span>
            <div class="preview-tb-spacer"></div>
            <button v-if="previewHasMoreLines" class="preview-tb-btn preview-tb-btn--accent" @click="loadAllPreviewLines" title="Load all remaining lines">Load all</button>
          </template>
        </div>

        <div class="preview-body" ref="previewBodyRef">
          <div v-if="previewLoading" class="preview-unsupported">
            <div class="base-btn__spinner" style="width:20px;height:20px;border-width:2px"></div>
          </div>

          <!-- Image -->
          <div v-else-if="isImage(previewEntry) && previewUrl" class="preview-img-wrap" @wheel="onPreviewImageWheel">
            <img :src="previewUrl"
                 class="preview-img"
                 :class="{ 'preview-img--fit': previewZoomIsFit }"
                 :style="!previewZoomIsFit ? `transform:scale(${previewZoom});transform-origin:top left` : ''"
                 alt="File preview"
                 @error="previewLoadError=true" />
          </div>

          <!-- Video -->
          <video v-else-if="isVideo(previewEntry) && previewUrl"
                 :src="previewUrl" class="preview-video" controls controlslist="nodownload" />

          <!-- Audio -->
          <div v-else-if="isAudio(previewEntry) && previewUrl" class="preview-audio-wrap">
            <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" style="color:var(--muted);opacity:.5">
              <path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/>
            </svg>
            <audio :src="previewUrl" class="preview-audio" controls />
          </div>

          <!-- PDF -->
          <iframe v-else-if="isPdf(previewEntry) && previewUrl"
                  :src="previewUrl" class="preview-pdf" title="PDF preview" />

          <!-- Markdown (rendered) -->
          <div v-else-if="isMarkdown(previewEntry) && previewHtml"
               class="preview-markdown" v-html="previewHtml" />

          <!-- JSON (formatted) -->
          <div v-else-if="isJson(previewEntry) && previewJsonFormatted" class="preview-code-wrap">
            <div class="preview-code-lang">JSON</div>
            <div class="preview-code-body" :class="{ 'preview-code-body--nowrap': !previewWordWrap }" @scroll="onPreviewCodeScroll">
              <pre v-if="previewShowLineNumbers && previewTextAllLines.length" class="preview-line-gutter" aria-hidden="true">{{ previewLineNumbersText }}</pre>
              <pre class="preview-text preview-text--code">{{ previewVisibleText || previewJsonFormatted }}</pre>
            </div>
            <div v-if="previewHasMoreLines" class="preview-load-more-bar">
              <button class="preview-load-more-btn" @click="loadMorePreviewLines">
                Load {{ Math.min(PREVIEW_LINE_CHUNK, previewTextAllLines.length - previewVisibleLineCount).toLocaleString() }} more lines
              </button>
              <span class="preview-load-more-info">{{ previewLineProgress }}</span>
            </div>
          </div>

          <!-- CSV / TSV (table) -->
          <div v-else-if="isCsv(previewEntry) && previewCsvRows.length" class="preview-csv-wrap">
            <div class="preview-code-lang">{{ previewLang }}</div>
            <div class="preview-csv-scroll">
              <table class="preview-csv-table">
                <thead v-if="previewCsvRows.length > 1">
                  <tr>
                    <th v-for="(cell, ci) in previewCsvRows[0]" :key="ci">{{ cell }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="(row, ri) in previewCsvRows.slice(1)" :key="ri">
                    <td v-for="(cell, ci) in row" :key="ci">{{ cell }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <div v-if="previewCsvRows.length >= 2000" class="preview-truncated">Showing first 2,000 rows</div>
          </div>

          <!-- Code / Config / Plain text (with language label) -->
          <div v-else-if="isTextPreviewable(previewEntry) && previewContent" class="preview-code-wrap">
            <div class="preview-code-lang">{{ previewLang }}</div>
            <div class="preview-code-body" :class="{ 'preview-code-body--nowrap': !previewWordWrap }" @scroll="onPreviewCodeScroll">
              <pre v-if="previewShowLineNumbers && previewTextAllLines.length" class="preview-line-gutter" aria-hidden="true">{{ previewLineNumbersText }}</pre>
              <pre class="preview-text preview-text--code">{{ previewTextAllLines.length ? previewVisibleText : previewContent }}</pre>
            </div>
            <div v-if="previewHasMoreLines" class="preview-load-more-bar">
              <button class="preview-load-more-btn" @click="loadMorePreviewLines">
                Load {{ Math.min(PREVIEW_LINE_CHUNK, previewTextAllLines.length - previewVisibleLineCount).toLocaleString() }} more lines
              </button>
              <span class="preview-load-more-info">{{ previewLineProgress }}</span>
            </div>
          </div>

          <!-- Excel / Spreadsheet -->
          <div v-else-if="isExcel(previewEntry) && previewExcelSheets.length" class="preview-excel-wrap">
            <div class="preview-excel-tabs" v-if="previewExcelSheets.length > 1">
              <button
                v-for="(sheet, si) in previewExcelSheets" :key="si"
                class="preview-excel-tab"
                :class="{ active: previewExcelActive === si }"
                @click="previewExcelActive = si"
              >{{ sheet.name }}</button>
            </div>
            <div class="preview-excel-tab-label" v-else>
              {{ previewExcelSheets[0]?.name }}
            </div>
            <div class="preview-csv-scroll">
              <table class="preview-csv-table">
                <thead>
                  <tr>
                    <th class="preview-csv-rownum">#</th>
                    <th v-for="(h, hi) in previewExcelSheets[previewExcelActive]?.headers" :key="hi">{{ h }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="(row, ri) in previewExcelSheets[previewExcelActive]?.rows" :key="ri">
                    <td class="preview-csv-rownum">{{ ri + 1 }}</td>
                    <td v-for="(cell, ci) in row" :key="ci">{{ cell }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <div v-if="(previewExcelSheets[previewExcelActive]?.totalRows ?? 0) > 2000" class="preview-truncated">
              Showing first 2,000 of {{ previewExcelSheets[previewExcelActive].totalRows.toLocaleString() }} rows
            </div>
          </div>

          <!-- Word / docx -->
          <div v-else-if="isWord(previewEntry) && previewWordHtml" class="preview-word-wrap">
            <div class="preview-code-lang">{{ previewLang }}</div>
            <div class="preview-word-body" v-html="previewWordHtml" />
          </div>

          <!-- PowerPoint (download prompt) -->
          <div v-else-if="isPowerPoint(previewEntry)" class="preview-unsupported">
            <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" style="opacity:.4">
              <rect x="2" y="3" width="20" height="14" rx="2" ry="2"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/>
            </svg>
            <p>Presentation — {{ languageLabel(previewEntry) }}</p>
            <p style="font-size:11px;color:var(--muted)">Download to open in your presentation app.</p>
          </div>

          <!-- Font files -->
          <div v-else-if="isFont(previewEntry)" class="preview-unsupported">
            <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" style="opacity:.4">
              <polyline points="4 7 4 4 20 4 20 7"/><line x1="9" y1="20" x2="15" y2="20"/><line x1="12" y1="4" x2="12" y2="20"/>
            </svg>
            <p>Font file — {{ previewLang || languageLabel(previewEntry) }}</p>
            <p style="font-size:11px;color:var(--muted)">Download to install or preview in a font viewer.</p>
          </div>

          <!-- Archive -->
          <div v-else-if="isArchive(previewEntry)" class="preview-unsupported">
            <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" style="opacity:.4">
              <path d="M21 8v13H3V8"/><path d="M23 3H1v5h22V3z"/><path d="M10 12h4"/>
            </svg>
            <p>Archive — {{ languageLabel(previewEntry) }}</p>
            <p style="font-size:11px;color:var(--muted)">Download to extract contents.</p>
          </div>

          <!-- Other Office documents (epub, etc.) -->
          <div v-else-if="isOffice(previewEntry)" class="preview-unsupported">
            <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" style="opacity:.4">
              <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
              <polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/><polyline points="10 9 9 9 8 9"/>
            </svg>
            <p>Document — {{ languageLabel(previewEntry) }}</p>
            <p style="font-size:11px;color:var(--muted)">Download to open in your office application.</p>
          </div>

          <!-- Unsupported / fallback -->
          <div v-else class="preview-unsupported">
            <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" style="opacity:.3">
              <path d="M13 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z"/><polyline points="13 2 13 9 20 9"/>
            </svg>
            <p>{{ previewLoadError ? 'Failed to load preview.' : `No preview available for .${getExt(previewEntry) || 'this'} files.` }}</p>
            <p v-if="!previewLoadError" style="font-size:11px;color:var(--muted)">Download to view this file.</p>
          </div>
        </div>
        <div class="preview-ft">
          <span class="preview-meta">
            {{ formatSize(previewEntry.size) }}
            <template v-if="previewLang"> · {{ previewLang }}</template>
            <template v-else-if="previewEntry.content_type"> · {{ previewEntry.content_type }}</template>
          </span>
          <button class="base-btn base-btn--ghost" @click="download(previewEntry)" style="font-size:12px;padding:5px 10px">Download</button>
        </div>
      </div>
    </transition>

    <!-- ── Metadata panel ───────────────────────────────────────── -->
    <transition name="slide-right">
      <div v-if="metaEntry" class="preview-panel">
        <div class="preview-hd">
          <span class="preview-hd__name">{{ metaEntry.display }} — Metadata</span>
          <button class="preview-close" @click="metaEntry = null">×</button>
        </div>
        <div class="preview-body" style="padding:0">
          <div v-if="metaLoading" class="preview-unsupported">
            <div class="base-btn__spinner" style="width:20px;height:20px;border-width:2px"></div>
          </div>
          <div v-else-if="metaData" style="padding:16px;display:flex;flex-direction:column;gap:14px">
            <!-- Content-Type -->
            <div class="meta-field">
              <label class="meta-label">Content-Type</label>
              <input class="base-input" v-model="metaEdit.content_type" style="font-size:12px;padding:6px 10px" />
            </div>
            <!-- Cache-Control -->
            <div class="meta-field">
              <label class="meta-label">Cache-Control</label>
              <input class="base-input" v-model="metaEdit.cache_control" style="font-size:12px;padding:6px 10px" />
            </div>
            <!-- Custom metadata -->
            <div class="meta-field">
              <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:6px">
                <label class="meta-label" style="margin-bottom:0">Custom Metadata</label>
                <button class="base-btn base-btn--ghost" @click="addMetaRow" style="font-size:11px;padding:3px 8px">+ Add</button>
              </div>
              <div v-for="(pair, i) in metaRows" :key="i" class="meta-row">
                <input class="base-input" v-model="pair.key" placeholder="key" style="font-size:11px;padding:5px 8px;flex:1" />
                <input class="base-input" v-model="pair.val" placeholder="value" style="font-size:11px;padding:5px 8px;flex:2" />
                <button class="row-btn danger" @click="metaRows.splice(i,1)" style="opacity:1;flex-shrink:0">
                  <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                    <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
                  </svg>
                </button>
              </div>
              <p v-if="metaRows.length === 0" style="font-size:11px;color:var(--muted)">No custom metadata.</p>
            </div>
            <!-- Read-only info -->
            <div style="padding:10px;background:var(--surface-2);border-radius:var(--r-sm);font-size:11px;color:var(--muted);line-height:1.8">
              <div>Size: <strong style="color:var(--text-2)">{{ formatSize(metaData.size) }}</strong></div>
              <div v-if="metaData.etag">ETag: <strong style="color:var(--text-2);font-family:var(--mono)">{{ metaData.etag }}</strong></div>
              <div v-if="metaData.md5">MD5: <strong style="color:var(--text-2);font-family:var(--mono)">{{ metaData.md5 }}</strong></div>
              <div v-if="metaData.updated">Modified: <strong style="color:var(--text-2)">{{ formatDate(metaData.updated) }}</strong></div>
            </div>
          </div>
          <div v-else-if="metaError" class="preview-unsupported" style="font-size:12px">{{ metaError }}</div>
        </div>
        <div class="preview-ft">
          <button class="base-btn base-btn--ghost" @click="metaEntry = null" style="font-size:12px;padding:5px 10px">Cancel</button>
          <button class="base-btn base-btn--primary" @click="saveMeta" :disabled="metaSaving" style="font-size:12px;padding:5px 12px">
            {{ metaSaving ? 'Saving…' : 'Save' }}
          </button>
        </div>
      </div>
    </transition>

    <!-- ── Modals ────────────────────────────────────────────────── -->

    <!-- New folder -->
    <BaseModal :open="showFolderModal" title="New Folder" @update:open="showFolderModal = false">
      <div style="display:flex;flex-direction:column;gap:10px">
        <label class="form-label">Folder name</label>
        <input
          ref="folderInput"
          class="base-input"
          v-model="newFolderName"
          placeholder="e.g. images"
          @keydown.enter="createFolder"
          @keydown.escape.stop="showFolderModal = false"
        />
        <p class="form-hint">A placeholder file (<code style="font-family:var(--mono)">.keep</code>) will be uploaded inside the new folder.</p>
      </div>
      <template #footer>
        <button class="base-btn base-btn--ghost" @click="showFolderModal = false">Cancel</button>
        <button class="base-btn base-btn--primary" @click="createFolder" :disabled="!newFolderName.trim()">Create</button>
      </template>
    </BaseModal>

    <!-- Rename / Move -->
    <BaseModal :open="showRenameModal" :title="`Rename: ${renameEntry?.display ?? ''}`" @update:open="showRenameModal = false">
      <div style="display:flex;flex-direction:column;gap:10px">
        <label class="form-label">New name (within current folder)</label>
        <input
          ref="renameInput"
          class="base-input"
          v-model="renameTarget"
          @keydown.enter="doRename"
          @keydown.escape.stop="showRenameModal = false"
        />
        <p class="form-hint">Current: <code style="font-family:var(--mono)">{{ renameEntry?.name }}</code></p>
      </div>
      <template #footer>
        <button class="base-btn base-btn--ghost" @click="showRenameModal = false">Cancel</button>
        <button class="base-btn base-btn--primary" @click="doRename" :disabled="!renameTarget.trim() || renaming">
          {{ renaming ? 'Moving…' : 'Move' }}
        </button>
      </template>
    </BaseModal>

    <!-- Transfer -->
    <BaseModal :open="showTransferModal" title="Transfer File" @update:open="showTransferModal = false">
      <div style="display:flex;flex-direction:column;gap:14px">
        <p style="font-size:13px;color:var(--text-2)">Transfer <strong style="color:var(--text)">{{ transferEntry?.display }}</strong> to another connection.</p>
        <div>
          <label class="form-label">Destination Connection</label>
          <div class="transfer-conn-list">
            <div
              v-for="c in otherConnections" :key="`${c.provider}-${c.id}`"
              class="transfer-conn-item"
              :class="{ active: transferDstConn?.id === c.id && transferDstConn?.provider === c.provider }"
              @click="transferDstConn = c"
            >
              <ProviderIcon :provider="c.provider" :size="12" />
              <span>{{ c.name }}</span>
              <span style="color:var(--muted);font-size:11px">{{ c.bucket }}</span>
            </div>
            <p v-if="!otherConnections.length" style="font-size:12px;color:var(--muted);padding:8px 4px">No other connections available.</p>
          </div>
        </div>
        <div>
          <label class="form-label">Destination Prefix <span style="font-weight:400;color:var(--muted)">(optional)</span></label>
          <input class="base-input" v-model="transferDstPrefix" placeholder="e.g. backups/" style="font-size:12px" />
        </div>
      </div>
      <template #footer>
        <button class="base-btn base-btn--ghost" @click="showTransferModal = false">Cancel</button>
        <button class="base-btn base-btn--primary" @click="doTransfer" :disabled="!transferDstConn || transferring">
          {{ transferring ? 'Transferring…' : 'Transfer' }}
        </button>
      </template>
    </BaseModal>

    <!-- Shareable link modal -->
    <BaseModal :open="showPresignModal" title="Generate Shareable Link" @update:open="showPresignModal = false">
      <div style="display:flex;flex-direction:column;gap:14px">
        <p style="font-size:13px;color:var(--text-2)">Create a time-limited link for <strong style="color:var(--text)">{{ presignEntry?.display }}</strong>.</p>
        <div>
          <label class="form-label">Link expires in</label>
          <div class="presign-presets">
            <button
              v-for="p in PRESIGN_PRESETS" :key="p.value"
              class="presign-chip"
              :class="{ active: presignExpiresIn === p.value }"
              @click="presignExpiresIn = p.value; presignUrl_ = ''"
            >{{ p.label }}</button>
          </div>
        </div>
        <div v-if="presignUrl_" class="presign-result">
          <input class="base-input" :value="presignUrl_" readonly style="font-size:11px;font-family:var(--mono)" />
          <button class="base-btn base-btn--primary" @click="copyPresignUrl" style="font-size:12px;padding:6px 12px;flex-shrink:0">Copy</button>
        </div>
      </div>
      <template #footer>
        <button class="base-btn base-btn--ghost" @click="showPresignModal = false">Close</button>
        <button class="base-btn base-btn--primary" @click="generatePresignUrl" :disabled="presignLoading">
          {{ presignLoading ? 'Generating…' : presignUrl_ ? 'Regenerate' : 'Generate Link' }}
        </button>
      </template>
    </BaseModal>

    <!-- CLI command modal -->
    <BaseModal :open="showCliModal" :title="`CLI — ${cliEntry?.display ?? ''}`" @update:open="showCliModal = false">
      <div v-if="cliEntry" style="display:flex;flex-direction:column;gap:12px">
        <div v-for="(cmd, label) in buildCliCommands(cliEntry)" :key="label" class="cli-row">
          <span class="cli-label">{{ label }}</span>
          <code class="cli-code">{{ cmd }}</code>
          <button class="base-btn base-btn--ghost" @click="copyCli(cmd)" style="font-size:11px;padding:4px 10px;flex-shrink:0">Copy</button>
        </div>
      </div>
      <template #footer>
        <button class="base-btn base-btn--ghost" @click="showCliModal = false">Close</button>
      </template>
    </BaseModal>

  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import BaseBadge    from '../ui/BaseBadge.vue'
import BaseModal    from '../ui/BaseModal.vue'
import ProviderIcon from '../ui/ProviderIcon.vue'
import { useConnections }   from '../../composables/useConnections.js'
import { useToast }         from '../../composables/useToast.js'
import { useConfirm }       from '../../composables/useConfirm.js'
import { useBookmarks }     from '../../composables/useBookmarks.js'
import { useActivity }      from '../../composables/useActivity.js'
import { useDragState }     from '../../composables/useDragState.js'
import {
  isImage, isVideo, isAudio, isPdf, isMarkdown,
  isJson, isCsv, isSvg, isCode, isConfig, isPlainText,
  isTextPreviewable, isArchive, isOffice, isFont,
  isExcel, isWord, isPowerPoint,
  fileCategory, languageLabel,
} from '../../utils/fileTypes.js'
import * as XLSX from 'xlsx'
import mammoth from 'mammoth'

const props = defineProps({
  conn:        { type: Object, required: true },
  connections: { type: Array,  default: () => [] },
  startPrefix: { type: String, default: '' },
  paneId:      { type: String, default: 'solo' }, // 'left' | 'right' | 'solo'
})
defineEmits(['delete'])

const { browseObjects, getDownloadURL, proxyDownload, presignUrl, zipDownload, deleteObject, copyObject, uploadObjects, uploadObjectWithProgress, deletePrefix, transferObject, getBucketStats, getObjectMetadata, updateObjectMetadata } = useConnections()
const toast   = useToast()
const confirm = useConfirm()
const { isBookmarked, toggleBookmark }            = useBookmarks()
const { log: activityLog }                        = useActivity()
const { dragState, startPaneDrag, clearPaneDrag } = useDragState()

// ── Core state ──────────────────────────────────────────────────
const currentPrefix  = ref('')
const entries        = ref([])
const nextPageToken  = ref('')
const loading        = ref(false)
const loadingMore    = ref(false)
const browseError    = ref('')

const searchQuery    = ref('')
const sortKey        = ref(localStorage.getItem('anveesa-sort-key') ?? 'name')
const sortDir        = ref(localStorage.getItem('anveesa-sort-dir') ?? 'asc')

// ── Stats ───────────────────────────────────────────────────────
const stats        = ref(null)
const statsLoading = ref(false)
const statsError   = ref('')
const statsLoaded  = ref(false)
const showStats    = ref(false)

// ── Upload ──────────────────────────────────────────────────────
const uploadQueue    = ref([]) // [{ name, progress, done, error }]
const isDragging     = ref(false)

// ── Bulk select ─────────────────────────────────────────────────
let selected = ref(new Set())
const bulkWorking = ref(false)

const fileEntries = computed(() => entries.value.filter(e => e.type === 'file'))
const allFilesSelected  = computed(() => fileEntries.value.length > 0 && fileEntries.value.every(e => selected.value.has(e.name)))
const someFilesSelected = computed(() => fileEntries.value.some(e => selected.value.has(e.name)))

function toggleSelect(name) {
  const s = new Set(selected.value)
  s.has(name) ? s.delete(name) : s.add(name)
  selected.value = s
}

function toggleSelectAll() {
  if (allFilesSelected.value) {
    selected.value = new Set()
  } else {
    selected.value = new Set(fileEntries.value.map(e => e.name))
  }
}

// ── Preview ─────────────────────────────────────────────────────
const previewEntry    = ref(null)
const previewUrl      = ref('')
const previewContent  = ref('')
const previewHtml     = ref('')
const previewLoading  = ref(false)
const previewLoadError = ref(false)

const previewFullscreen = ref(false)
const previewZoom = ref(1)
const previewZoomIsFit = ref(true)
const previewWordWrap = ref(true)
const previewShowLineNumbers = ref(true)
const previewVisibleLineCount = ref(500)
const PREVIEW_LINE_CHUNK = 500
const PREVIEW_MAX_BYTES = 5_000_000
const previewTextAllLines = ref([])
const previewBodyRef = ref(null)

// ── Transfer modal ───────────────────────────────────────────────
const showTransferModal = ref(false)
const transferEntry     = ref(null)
const transferDstConn   = ref(null)
const transferDstPrefix = ref('')
const transferring      = ref(false)
const otherConnections  = computed(() => props.connections.filter(
  c => !(c.provider === props.conn.provider && c.id === props.conn.id)
))

// ── Metadata editor ─────────────────────────────────────────────
const metaEntry   = ref(null)
const metaData    = ref(null)
const metaEdit    = ref({ content_type: '', cache_control: '' })
const metaRows    = ref([]) // [{ key, val }]
const metaLoading = ref(false)
const metaError   = ref('')
const metaSaving  = ref(false)

// ── Modals ──────────────────────────────────────────────────────
const showFolderModal = ref(false)
const newFolderName   = ref('')
const folderInput     = ref(null)

const showRenameModal = ref(false)
const renameEntry     = ref(null)
const renameTarget    = ref('')
const renameInput     = ref(null)
const renaming        = ref(false)

// ── View mode ────────────────────────────────────────────────────
const viewMode = ref(localStorage.getItem('anveesa-view') ?? 'table') // 'table' | 'grid'
function toggleView() {
  viewMode.value = viewMode.value === 'table' ? 'grid' : 'table'
  localStorage.setItem('anveesa-view', viewMode.value)
}

// ── Presign modal ────────────────────────────────────────────────
const showPresignModal  = ref(false)
const presignEntry      = ref(null)
const presignExpiresIn  = ref(3600) // seconds
const presignUrl_       = ref('')   // generated URL
const presignLoading    = ref(false)
const PRESIGN_PRESETS   = [
  { label: '15 min',  value: 900   },
  { label: '1 hour',  value: 3600  },
  { label: '24 hours',value: 86400 },
  { label: '7 days',  value: 604800},
]

// ── CLI copy modal ───────────────────────────────────────────────
const showCliModal = ref(false)
const cliEntry     = ref(null)

// ── Zip ─────────────────────────────────────────────────────────
const zipping = ref(false)

// ── Bookmarks ───────────────────────────────────────────────────
const isCurrentBookmarked = computed(() =>
  isBookmarked(props.conn.provider, props.conn.id, currentPrefix.value)
)
function toggleCurrentBookmark() {
  toggleBookmark(props.conn.provider, props.conn.id, props.conn.bucket, currentPrefix.value, props.conn.name)
}

// ── Cross-pane drag ─────────────────────────────────────────────
const isCrossTarget = computed(() =>
  dragState.value !== null && dragState.value.paneId !== props.paneId
)

// ── Keyboard navigation ─────────────────────────────────────────
const focusedIdx = ref(-1)

// ── DOM refs ────────────────────────────────────────────────────
const searchInput      = ref(null)
const bodyEl           = ref(null)
const sentinel         = ref(null)
const folderUploadInput = ref(null)
let   observer         = null

// ── Breadcrumbs ─────────────────────────────────────────────────
const breadcrumbs = computed(() => {
  if (!currentPrefix.value) return []
  const parts = currentPrefix.value.replace(/\/$/, '').split('/')
  return parts.map((label, i) => ({
    label,
    prefix: parts.slice(0, i + 1).join('/') + '/',
  }))
})

// ── Filtered + sorted entries ───────────────────────────────────
const filteredEntries = computed(() => {
  let list = entries.value
  if (searchQuery.value.trim()) {
    const q = searchQuery.value.trim().toLowerCase()
    list = list.filter(e => e.display.toLowerCase().includes(q))
  }
  const dirs  = list.filter(e => e.type === 'dir')
  const files = list.filter(e => e.type === 'file')
  const sortFn = (a, b) => {
    let va, vb
    if (sortKey.value === 'size')      { va = a.size ?? 0; vb = b.size ?? 0 }
    else if (sortKey.value === 'date') { va = a.updated ? new Date(a.updated).getTime() : 0; vb = b.updated ? new Date(b.updated).getTime() : 0 }
    else                               { va = a.display.toLowerCase(); vb = b.display.toLowerCase() }
    if (va < vb) return sortDir.value === 'asc' ? -1 : 1
    if (va > vb) return sortDir.value === 'asc' ?  1 : -1
    return 0
  }
  return [...dirs.sort(sortFn), ...files.sort(sortFn)]
})

// ── Navigation ──────────────────────────────────────────────────
function navigateTo(prefix) {
  currentPrefix.value = prefix
  searchQuery.value   = ''
  selected.value      = new Set()
  focusedIdx.value    = -1
  load()
}

function navigateUp() {
  if (!currentPrefix.value) return
  const t = currentPrefix.value.replace(/\/$/, '')
  navigateTo(t.includes('/') ? t.slice(0, t.lastIndexOf('/') + 1) : '')
}

// ── Data loading ────────────────────────────────────────────────
async function load() {
  loading.value     = true
  browseError.value = ''
  entries.value     = []
  nextPageToken.value = ''
  focusedIdx.value    = -1
  try {
    const result = await browseObjects(props.conn.provider, props.conn.bucket, props.conn.credentials, currentPrefix.value)
    entries.value       = result.entries ?? []
    nextPageToken.value = result.next_page_token ?? ''
  } catch (err) {
    browseError.value = err.message
  } finally {
    loading.value = false
    setupObserver()
  }
}

async function loadMore() {
  if (!nextPageToken.value || loadingMore.value) return
  loadingMore.value = true
  try {
    const result = await browseObjects(props.conn.provider, props.conn.bucket, props.conn.credentials, currentPrefix.value, nextPageToken.value)
    entries.value.push(...(result.entries ?? []))
    nextPageToken.value = result.next_page_token ?? ''
  } catch (err) {
    toast.error('Failed to load more: ' + err.message)
  } finally {
    loadingMore.value = false
  }
}

function refresh() { load() }

// ── Infinite scroll ─────────────────────────────────────────────
function setupObserver() {
  if (observer) { observer.disconnect(); observer = null }
  nextTick(() => {
    if (!sentinel.value) return
    observer = new IntersectionObserver(entries => {
      if (entries[0].isIntersecting) loadMore()
    }, { root: bodyEl.value, threshold: 0.1 })
    observer.observe(sentinel.value)
  })
}

// ── Stats ───────────────────────────────────────────────────────
async function loadStats() {
  if (statsLoaded.value) return
  statsLoading.value = true
  statsError.value   = ''
  try {
    stats.value      = await getBucketStats(props.conn.provider, props.conn.bucket, props.conn.credentials)
    statsLoaded.value = true
  } catch (err) {
    statsError.value = 'Stats unavailable'
  } finally {
    statsLoading.value = false
  }
}

function toggleStats() {
  showStats.value = !showStats.value
  if (showStats.value && !statsLoaded.value) loadStats()
}

// ── Sort ────────────────────────────────────────────────────────
function cycleSort(key) {
  if (sortKey.value === key) sortDir.value = sortDir.value === 'asc' ? 'desc' : 'asc'
  else { sortKey.value = key; sortDir.value = 'asc' }
  localStorage.setItem('anveesa-sort-key', sortKey.value)
  localStorage.setItem('anveesa-sort-dir', sortDir.value)
}

// ── Bulk operations ─────────────────────────────────────────────
async function bulkDelete() {
  const names = [...selected.value]
  const ok = await confirm.confirm(`Delete ${names.length} file${names.length > 1 ? 's' : ''}? This cannot be undone.`, 'Bulk Delete')
  if (!ok) return
  bulkWorking.value = true
  let failed = 0
  for (const name of names) {
    try {
      await deleteObject(props.conn.provider, props.conn.bucket, props.conn.credentials, name)
    } catch { failed++ }
  }
  selected.value = new Set()
  bulkWorking.value = false
  if (failed) toast.error(`${failed} file(s) could not be deleted.`)
  else {
    toast.success(`${names.length} file${names.length > 1 ? 's' : ''} deleted.`)
    activityLog('delete', `Deleted ${names.length} file${names.length > 1 ? 's' : ''}`, props.conn.provider)
  }
  await load()
  if (statsLoaded.value) { statsLoaded.value = false; loadStats() }
}

async function bulkDownload() {
  const files = filteredEntries.value.filter(e => e.type === 'file' && selected.value.has(e.name))
  bulkWorking.value = true
  for (const entry of files) {
    try {
      const url = await getDownloadURL(props.conn.provider, props.conn.bucket, props.conn.credentials, entry.name)
      const a = document.createElement('a')
      a.href = url; a.download = entry.display; a.target = '_blank'; a.rel = 'noopener'
      document.body.appendChild(a); a.click(); document.body.removeChild(a)
      await new Promise(r => setTimeout(r, 300)) // small delay to avoid popup block
    } catch (err) { toast.error('Download failed: ' + entry.display) }
  }
  bulkWorking.value = false
}

// ── Presign URL modal ────────────────────────────────────────────
function openPresign(entry) {
  presignEntry.value     = entry
  presignExpiresIn.value = 3600
  presignUrl_.value      = ''
  showPresignModal.value = true
}

async function generatePresignUrl() {
  if (!presignEntry.value || presignLoading.value) return
  presignLoading.value = true
  try {
    const url = await presignUrl(
      props.conn.provider, props.conn.bucket, props.conn.credentials,
      presignEntry.value.name, presignExpiresIn.value
    )
    presignUrl_.value = url
  } catch (err) {
    toast.error('Failed to generate link: ' + err.message)
  } finally {
    presignLoading.value = false
  }
}

function copyPresignUrl() {
  navigator.clipboard?.writeText(presignUrl_.value).then(
    () => toast.success('Link copied!'),
    () => toast.error('Clipboard not available'),
  )
}

// ── CLI copy ──────────────────────────────────────────────────────
function buildCliCommands(entry) {
  const p = props.conn.provider
  const b = props.conn.bucket
  const k = entry.name
  const display = entry.display

  if (p === 'gcp') return {
    download: `gsutil cp gs://${b}/${k} ./${display}`,
    delete:   `gsutil rm gs://${b}/${k}`,
    ls:       `gsutil ls gs://${b}/${currentPrefix.value}`,
  }
  if (p === 'azure') return {
    download: `azcopy copy "https://<account>.blob.core.windows.net/${b}/${k}" "./${display}"`,
    delete:   `azcopy remove "https://<account>.blob.core.windows.net/${b}/${k}"`,
    ls:       `azcopy list "https://<account>.blob.core.windows.net/${b}/${currentPrefix.value}"`,
  }
  if (p === 'alibaba') return {
    download: `ossutil cp oss://${b}/${k} ./${display}`,
    delete:   `ossutil rm oss://${b}/${k}`,
    ls:       `ossutil ls oss://${b}/${currentPrefix.value}`,
  }
  if (p === 'huawei') return {
    download: `obsutil cp obs://${b}/${k} ./${display}`,
    delete:   `obsutil rm obs://${b}/${k}`,
    ls:       `obsutil ls obs://${b}/${currentPrefix.value}`,
  }
  // aws / default S3-compatible
  return {
    download: `aws s3 cp s3://${b}/${k} ./${display}`,
    delete:   `aws s3 rm s3://${b}/${k}`,
    ls:       `aws s3 ls s3://${b}/${currentPrefix.value}`,
  }
}

function openCli(entry) {
  cliEntry.value  = entry
  showCliModal.value = true
}

function copyCli(cmd) {
  navigator.clipboard?.writeText(cmd).then(
    () => toast.success('Command copied!'),
    () => toast.error('Clipboard not available'),
  )
}

// ── Zip download ──────────────────────────────────────────────────
async function downloadZip(prefix, objects) {
  if (zipping.value) return
  zipping.value = true
  try {
    const blob = await zipDownload(props.conn.provider, props.conn.bucket, props.conn.credentials, prefix, objects)
    const archiveName = objects?.length
      ? 'selection'
      : (prefix.replace(/\/$/, '').split('/').pop() || props.conn.bucket)
    const a = document.createElement('a')
    a.href = URL.createObjectURL(blob)
    a.download = archiveName + '.zip'
    document.body.appendChild(a); a.click(); document.body.removeChild(a)
    URL.revokeObjectURL(a.href)
    toast.success('Zip downloaded.')
  } catch (err) {
    toast.error('Zip failed: ' + err.message)
  } finally {
    zipping.value = false
  }
}

// ── Copy public URL ──────────────────────────────────────────────
async function copyPublicUrl(entry) {
  try {
    const url = await getDownloadURL(props.conn.provider, props.conn.bucket, props.conn.credentials, entry.name)
    await navigator.clipboard.writeText(url)
    toast.success('Link copied to clipboard')
  } catch (err) {
    toast.error('Failed to copy link: ' + err.message)
  }
}

// ── Copy path ───────────────────────────────────────────────────
function copyPath(entry) {
  const path = props.conn.provider === 'gcp'
    ? `gs://${props.conn.bucket}/${entry.name}`
    : props.conn.provider === 'azure'
    ? `az://${props.conn.bucket}/${entry.name}`
    : props.conn.provider === 'gdrive'
    ? `gdrive://${props.conn.bucket}/${entry.name}`
    : `s3://${props.conn.bucket}/${entry.name}`
  navigator.clipboard?.writeText(path).then(
    () => toast.success('Path copied to clipboard'),
    () => toast.error('Clipboard not available'),
  )
}

// ── Download ────────────────────────────────────────────────────
async function download(entry) {
  try {
    const url = await getDownloadURL(props.conn.provider, props.conn.bucket, props.conn.credentials, entry.name)
    const a = document.createElement('a')
    a.href = url; a.download = entry.display; a.target = '_blank'; a.rel = 'noopener'
    document.body.appendChild(a); a.click(); document.body.removeChild(a)
    activityLog('download', `Downloaded "${entry.display}"`, props.conn.provider)
  } catch (err) {
    toast.error('Download failed: ' + err.message)
  }
}

// ── Delete single ───────────────────────────────────────────────
async function confirmDelete(entry) {
  const ok = await confirm.confirm(`Delete "${entry.display}"? This cannot be undone.`)
  if (!ok) return
  try {
    await deleteObject(props.conn.provider, props.conn.bucket, props.conn.credentials, entry.name)
    if (previewEntry.value?.name === entry.name) closePreview()
    toast.success(`"${entry.display}" deleted.`)
    activityLog('delete', `Deleted "${entry.display}"`, props.conn.provider)
    await load()
    if (statsLoaded.value) { statsLoaded.value = false; loadStats() }
  } catch (err) {
    toast.error('Delete failed: ' + err.message)
  }
}

// ── Folder delete (recursive) ────────────────────────────────────
async function confirmDeleteFolder(entry) {
  const ok = await confirm.confirm(
    `Delete folder "${entry.display}" and all its contents? This cannot be undone.`, 'Delete Folder'
  )
  if (!ok) return
  try {
    const r = await deletePrefix(props.conn.provider, props.conn.bucket, props.conn.credentials, entry.name)
    toast.success(`Folder deleted (${r.deleted} file${r.deleted !== 1 ? 's' : ''} removed).`)
    activityLog('delete', `Deleted folder "${entry.display}" (${r.deleted} files)`, props.conn.provider)
    await load()
    if (statsLoaded.value) { statsLoaded.value = false; loadStats() }
  } catch (err) {
    toast.error('Delete folder failed: ' + err.message)
  }
}

// ── Cross-connection transfer ────────────────────────────────────
function openTransfer(entry) {
  transferEntry.value     = entry
  transferDstConn.value   = null
  transferDstPrefix.value = ''
  showTransferModal.value = true
}

async function doTransfer() {
  if (!transferDstConn.value || transferring.value) return
  transferring.value = true
  try {
    const r = await transferObject(
      { provider: props.conn.provider, bucket: props.conn.bucket, credentials: props.conn.credentials, object: transferEntry.value.name },
      { provider: transferDstConn.value.provider, bucket: transferDstConn.value.bucket, credentials: transferDstConn.value.credentials, prefix: transferDstPrefix.value }
    )
    toast.success(`Transferred to ${transferDstConn.value.name}: ${r.destination}`)
    activityLog('transfer', `Transferred "${transferEntry.value.display}" → ${transferDstConn.value.name}`, props.conn.provider)
    showTransferModal.value = false
  } catch (err) {
    toast.error('Transfer failed: ' + err.message)
  } finally {
    transferring.value = false
  }
}

// ── Upload ──────────────────────────────────────────────────────
async function handleUpload(files) {
  if (!files?.length) return
  const fileArr = Array.from(files)
  uploadQueue.value = fileArr.map(f => ({ name: f.name, file: f, progress: 0, done: false, error: false }))
  let anyError = false
  for (let i = 0; i < fileArr.length; i++) {
    try {
      await uploadObjectWithProgress(
        props.conn.provider, props.conn.bucket, props.conn.credentials,
        currentPrefix.value, fileArr[i],
        p => { uploadQueue.value[i].progress = p }
      )
      uploadQueue.value[i].done     = true
      uploadQueue.value[i].progress = 1
    } catch {
      uploadQueue.value[i].error = true
      anyError = true
    }
  }
  const done = uploadQueue.value.filter(u => u.done).length
  const hasErrors = uploadQueue.value.some(u => u.error)
  if (!hasErrors) setTimeout(() => { uploadQueue.value = [] }, 1500)
  if (done > 0) {
    toast.success(`${done} file${done !== 1 ? 's' : ''} uploaded.`)
    activityLog('upload', `Uploaded ${done} file${done !== 1 ? 's' : ''}`, props.conn.provider)
    await load()
    if (statsLoaded.value) { statsLoaded.value = false; loadStats() }
  }
  if (anyError) toast.error('Some files failed to upload.')
}

function onFileInput(e) { handleUpload(e.target.files); e.target.value = '' }

async function retryUpload(index) {
  const item = uploadQueue.value[index]
  if (!item?.file) return
  item.error = false
  item.progress = 0
  item.done = false
  try {
    await uploadObjectWithProgress(
      props.conn.provider, props.conn.bucket, props.conn.credentials,
      currentPrefix.value, item.file,
      p => { uploadQueue.value[index].progress = p }
    )
    item.done = true
    item.progress = 1
    toast.success(`${item.name} uploaded.`)
    await load()
  } catch {
    item.error = true
    toast.error(`${item.name} failed again.`)
  }
}

// ── Folder upload ────────────────────────────────────────────────
async function handleFolderUpload(files) {
  if (!files?.length) return
  const fileArr = Array.from(files)

  // Build queue items showing the relative path for context
  uploadQueue.value = fileArr.map(f => ({
    name:     f.webkitRelativePath || f.name,
    progress: 0,
    done:     false,
    error:    false,
  }))

  let anyError = false
  for (let i = 0; i < fileArr.length; i++) {
    const file    = fileArr[i]
    // Derive the per-file prefix: currentPrefix + everything except the base filename
    const relPath = file.webkitRelativePath || file.name
    const relDir  = relPath.includes('/')
      ? relPath.slice(0, relPath.lastIndexOf('/') + 1)
      : ''
    const filePrefix = currentPrefix.value + relDir

    try {
      await uploadObjectWithProgress(
        props.conn.provider, props.conn.bucket, props.conn.credentials,
        filePrefix, file,
        p => { uploadQueue.value[i].progress = p }
      )
      uploadQueue.value[i].done     = true
      uploadQueue.value[i].progress = 1
    } catch {
      uploadQueue.value[i].error = true
      anyError = true
    }
  }

  const done = uploadQueue.value.filter(u => u.done).length
  setTimeout(() => { uploadQueue.value = [] }, 1500)
  if (done > 0) {
    toast.success(`${done} file${done !== 1 ? 's' : ''} uploaded.`)
    activityLog('upload', `Uploaded ${done} file${done !== 1 ? 's' : ''} (folder)`, props.conn.provider)
    await load()
    if (statsLoaded.value) { statsLoaded.value = false; loadStats() }
  }
  if (anyError) toast.error('Some files failed to upload.')
}

function onFolderInput(e) {
  handleFolderUpload(e.target.files)
  if (folderUploadInput.value) folderUploadInput.value.value = ''
}

let dragCounter = 0
function onDragOver(e) {
  if (isCrossTarget.value) { isDragging.value = true; return }
  if (!e.dataTransfer?.types.includes('Files')) return
  dragCounter++; isDragging.value = true
}
function onDragLeave() {
  if (isCrossTarget.value) { isDragging.value = false; return }
  if (--dragCounter <= 0) { dragCounter = 0; isDragging.value = false }
}
function onDrop(e) {
  dragCounter = 0; isDragging.value = false
  if (dragState.value && dragState.value.paneId !== props.paneId) {
    performCrossPaneCopy(dragState.value)
    clearPaneDrag()
    return
  }
  handleUpload(e.dataTransfer?.files)
}

function onRowDragStart(e, entry) {
  e.stopPropagation()
  e.dataTransfer.effectAllowed = 'copy'
  startPaneDrag(entry, props.conn, currentPrefix.value, props.paneId)
}
function onRowDragEnd() { clearPaneDrag(); isDragging.value = false }

async function performCrossPaneCopy(state) {
  try {
    const r = await transferObject(
      { provider: state.conn.provider, bucket: state.conn.bucket, credentials: state.conn.credentials, object: state.entry.name },
      { provider: props.conn.provider,  bucket: props.conn.bucket,  credentials: props.conn.credentials,  prefix: currentPrefix.value }
    )
    activityLog('transfer', `Copied "${state.entry.display}" to ${props.conn.name}`, props.conn.provider)
    toast.success(`Copied to ${props.conn.name}: ${r.destination}`)
    await load()
    if (statsLoaded.value) { statsLoaded.value = false; loadStats() }
  } catch (err) {
    toast.error('Copy failed: ' + err.message)
  }
}

// ── Create folder ────────────────────────────────────────────────
async function createFolder() {
  const name = newFolderName.value.trim()
  if (!name) return
  const prefix  = currentPrefix.value + name.replace(/\/+$/, '') + '/'
  const keepFile = new File([''], '.keep', { type: 'application/octet-stream' })
  showFolderModal.value = false
  newFolderName.value   = ''
  try {
    await uploadObjects(props.conn.provider, props.conn.bucket, props.conn.credentials, prefix, [keepFile])
    toast.success(`Folder "${name}" created.`)
    await load()
  } catch (err) {
    toast.error('Create folder failed: ' + err.message)
  }
}

watch(showFolderModal, open => { if (open) nextTick(() => folderInput.value?.focus()) })

// ── Rename / Move ────────────────────────────────────────────────
function openRename(entry) {
  renameEntry.value  = entry
  renameTarget.value = entry.display
  showRenameModal.value = true
  nextTick(() => renameInput.value?.focus())
}

async function doRename() {
  const target = renameTarget.value.trim()
  if (!target || renaming.value) return
  const destination = currentPrefix.value + target
  if (destination === renameEntry.value.name) { showRenameModal.value = false; return }
  renaming.value = true
  try {
    await copyObject(props.conn.provider, props.conn.bucket, props.conn.credentials, renameEntry.value.name, destination, true)
    if (previewEntry.value?.name === renameEntry.value.name) closePreview()
    toast.success(`Renamed to "${target}".`)
    activityLog('rename', `Renamed "${renameEntry.value.display}" → "${target}"`, props.conn.provider)
    showRenameModal.value = false
    await load()
  } catch (err) {
    toast.error('Rename failed: ' + err.message)
  } finally {
    renaming.value = false
  }
}

// ── Preview ─────────────────────────────────────────────────────
const previewJsonFormatted = ref('')
const previewCsvRows       = ref([])
const previewLang          = ref('')
const previewExcelSheets   = ref([])       // [{ name, headers, rows }]
const previewExcelActive   = ref(0)        // active sheet tab index
const previewWordHtml      = ref('')       // rendered Word docx HTML

const previewVisibleText = computed(() => {
  if (!previewTextAllLines.value.length) return previewContent.value
  return previewTextAllLines.value.slice(0, previewVisibleLineCount.value).join('\n')
})

const previewHasMoreLines = computed(() =>
  previewTextAllLines.value.length > previewVisibleLineCount.value
)

const previewLineProgress = computed(() => {
  if (!previewTextAllLines.value.length) return ''
  const shown = Math.min(previewVisibleLineCount.value, previewTextAllLines.value.length)
  return `${shown.toLocaleString()} / ${previewTextAllLines.value.length.toLocaleString()} lines`
})

const previewLineNumbersText = computed(() => {
  const count = Math.min(previewVisibleLineCount.value, previewTextAllLines.value.length)
  if (count === 0) return ''
  const arr = []
  for (let i = 1; i <= count; i++) arr.push(i)
  return arr.join('\n')
})

async function openPreview(entry) {
  metaEntry.value            = null
  previewEntry.value         = entry
  previewUrl.value           = ''
  previewContent.value       = ''
  previewHtml.value          = ''
  previewJsonFormatted.value = ''
  previewCsvRows.value       = []
  previewLang.value          = ''
  previewExcelSheets.value   = []
  previewExcelActive.value   = 0
  previewWordHtml.value           = ''
  previewLoadError.value          = false
  previewTextAllLines.value       = []
  previewVisibleLineCount.value   = PREVIEW_LINE_CHUNK
  previewZoom.value               = 1
  previewZoomIsFit.value          = true
  previewLoading.value            = true
  try {
    // For media types (img/video/audio/pdf), we use presigned URLs since
    // HTML media elements aren't subject to CORS.
    // For everything else, we proxy through the backend to avoid CORS.
    const needsProxy = isExcel(entry) || isWord(entry) || isTextPreviewable(entry)

    if (!needsProxy) {
      const url = await getDownloadURL(props.conn.provider, props.conn.bucket, props.conn.credentials, entry.name)
      previewUrl.value = url
    }

    if (isExcel(entry)) {
      await loadExcelPreview(entry)
    } else if (isWord(entry)) {
      await loadWordPreview(entry)
    } else if (isTextPreviewable(entry)) {
      const res = await proxyDownload(props.conn.provider, props.conn.bucket, props.conn.credentials, entry.name)
      const text = (await res.text()).slice(0, PREVIEW_MAX_BYTES)
      previewContent.value = text
      previewLang.value = languageLabel(entry)

      if (isMarkdown(entry)) {
        previewHtml.value = DOMPurify.sanitize(marked.parse(text))
      } else if (isJson(entry)) {
        try {
          const formatted = JSON.stringify(JSON.parse(text), null, 2)
          previewJsonFormatted.value = formatted
          previewTextAllLines.value = formatted.split('\n')
        } catch {
          previewJsonFormatted.value = text
          previewTextAllLines.value = text.split('\n')
        }
        previewVisibleLineCount.value = PREVIEW_LINE_CHUNK
      } else if (isCsv(entry)) {
        parseCsvPreview(text, entry)
      } else {
        previewTextAllLines.value = text.split('\n')
        previewVisibleLineCount.value = PREVIEW_LINE_CHUNK
      }
    }
  } catch { previewLoadError.value = true }
  finally { previewLoading.value = false }
}

async function loadExcelPreview(entry) {
  const res = await proxyDownload(props.conn.provider, props.conn.bucket, props.conn.credentials, entry.name)
  const buf = await res.arrayBuffer()
  const wb = XLSX.read(new Uint8Array(buf), { type: 'array' })
  previewExcelSheets.value = wb.SheetNames.map(name => {
    const sheet = wb.Sheets[name]
    const json = XLSX.utils.sheet_to_json(sheet, { header: 1, defval: '' })
    const headers = json[0] || []
    const rows = json.slice(1, 2001)
    return { name, headers, rows, totalRows: json.length - 1 }
  })
  previewLang.value = languageLabel(previewEntry.value)
}

async function loadWordPreview(entry) {
  const res = await proxyDownload(props.conn.provider, props.conn.bucket, props.conn.credentials, entry.name)
  const buf = await res.arrayBuffer()
  const result = await mammoth.convertToHtml({ arrayBuffer: buf })
  previewWordHtml.value = DOMPurify.sanitize(result.value)
  previewLang.value = languageLabel(previewEntry.value)
}

function parseCsvPreview(text, entry) {
  const sep = getExt(entry) === 'tsv' ? '\t' : ','
  const lines = text.split('\n').filter(l => l.trim()).slice(0, 2000)
  previewCsvRows.value = lines.map(line => {
    const cells = []
    let current = '', inQuotes = false
    for (let i = 0; i < line.length; i++) {
      const ch = line[i]
      if (ch === '"') { inQuotes = !inQuotes; continue }
      if (ch === sep && !inQuotes) { cells.push(current.trim()); current = ''; continue }
      current += ch
    }
    cells.push(current.trim())
    return cells
  })
}

function getExt(entry) {
  const name = entry?.display || entry?.name || ''
  const dot = name.lastIndexOf('.')
  return dot >= 0 ? name.slice(dot + 1).toLowerCase() : ''
}

function closePreview() {
  previewEntry.value              = null
  previewUrl.value                = ''
  previewContent.value            = ''
  previewHtml.value               = ''
  previewJsonFormatted.value      = ''
  previewCsvRows.value            = []
  previewLang.value               = ''
  previewExcelSheets.value        = []
  previewExcelActive.value        = 0
  previewWordHtml.value           = ''
  previewFullscreen.value         = false
  previewZoom.value               = 1
  previewZoomIsFit.value          = true
  previewTextAllLines.value       = []
  previewVisibleLineCount.value   = PREVIEW_LINE_CHUNK
}

function togglePreviewFullscreen() {
  previewFullscreen.value = !previewFullscreen.value
}

function loadMorePreviewLines() {
  if (previewVisibleLineCount.value < previewTextAllLines.value.length) {
    previewVisibleLineCount.value = Math.min(
      previewVisibleLineCount.value + PREVIEW_LINE_CHUNK,
      previewTextAllLines.value.length
    )
  }
}

function loadAllPreviewLines() {
  previewVisibleLineCount.value = previewTextAllLines.value.length
}

function onPreviewCodeScroll(e) {
  const el = e.target
  if (previewTextAllLines.value.length && el.scrollHeight - el.scrollTop - el.clientHeight < 300) {
    loadMorePreviewLines()
  }
}

function previewDoZoomIn() {
  previewZoomIsFit.value = false
  previewZoom.value = Math.min(+(previewZoom.value + 0.25).toFixed(2), 5)
}

function previewDoZoomOut() {
  previewZoomIsFit.value = false
  previewZoom.value = Math.max(+(previewZoom.value - 0.25).toFixed(2), 0.1)
}

function previewDoZoomReset() {
  previewZoomIsFit.value = false
  previewZoom.value = 1
}

function previewDoZoomFit() {
  previewZoomIsFit.value = true
  previewZoom.value = 1
}

function onPreviewImageWheel(e) {
  if (!e.ctrlKey && !e.metaKey) return
  e.preventDefault()
  previewZoomIsFit.value = false
  const delta = e.deltaY > 0 ? -0.1 : 0.1
  previewZoom.value = Math.max(0.1, Math.min(5, +(previewZoom.value + delta).toFixed(2)))
}

// ── Metadata editor ─────────────────────────────────────────────
async function openMeta(entry) {
  previewEntry.value = null
  metaEntry.value    = entry
  metaData.value     = null
  metaError.value    = ''
  metaLoading.value  = true
  try {
    const data = await getObjectMetadata(props.conn.provider, props.conn.bucket, props.conn.credentials, entry.name)
    metaData.value = data
    metaEdit.value = { content_type: data.content_type || '', cache_control: data.cache_control || '' }
    metaRows.value = Object.entries(data.metadata || {}).map(([key, val]) => ({ key, val }))
  } catch (err) {
    metaError.value = 'Failed to load metadata: ' + err.message
  } finally {
    metaLoading.value = false
  }
}

function addMetaRow() { metaRows.value.push({ key: '', val: '' }) }

async function saveMeta() {
  metaSaving.value = true
  try {
    const metadata = {}
    for (const { key, val } of metaRows.value) {
      if (key.trim()) metadata[key.trim()] = val
    }
    await updateObjectMetadata(props.conn.provider, props.conn.bucket, props.conn.credentials, metaEntry.value.name, {
      content_type:  metaEdit.value.content_type,
      cache_control: metaEdit.value.cache_control,
      metadata,
    })
    toast.success('Metadata saved.')
    metaEntry.value = null
  } catch (err) {
    toast.error('Save failed: ' + err.message)
  } finally {
    metaSaving.value = false
  }
}

// ── Keyboard shortcuts ──────────────────────────────────────────
function scrollFocused() {
  nextTick(() => {
    const rows = bodyEl.value?.querySelectorAll('.file-table tbody tr.file-row')
    rows?.[focusedIdx.value]?.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
  })
}

function onKeyDown(e) {
  const inInput = ['INPUT', 'TEXTAREA'].includes(document.activeElement?.tagName)

  // Focus search
  if (e.key === '/' && !inInput) { e.preventDefault(); searchInput.value?.focus(); return }

  // Refresh
  if ((e.key === 'r' || e.key === 'R') && !inInput && !e.metaKey && !e.ctrlKey) { refresh(); return }

  // f — toggle preview fullscreen
  if ((e.key === 'f' || e.key === 'F') && !inInput && !e.metaKey && !e.ctrlKey && previewEntry.value) {
    togglePreviewFullscreen()
    return
  }

  // Close / collapse
  if (e.key === 'Escape') {
    if (previewFullscreen.value) { previewFullscreen.value = false; return }
    if (metaEntry.value)    { metaEntry.value = null; return }
    if (previewEntry.value) { closePreview(); return }
    if (searchQuery.value)  { searchQuery.value = ''; return }
    if (focusedIdx.value >= 0) { focusedIdx.value = -1; return }
  }

  // Navigate up (Backspace)
  if (e.key === 'Backspace' && !inInput && currentPrefix.value) { e.preventDefault(); navigateUp(); return }

  // Row navigation — j / ArrowDown
  if ((e.key === 'j' || e.key === 'ArrowDown') && !inInput) {
    e.preventDefault()
    focusedIdx.value = Math.min(focusedIdx.value + 1, filteredEntries.value.length - 1)
    scrollFocused()
    return
  }

  // Row navigation — k / ArrowUp
  if ((e.key === 'k' || e.key === 'ArrowUp') && !inInput) {
    e.preventDefault()
    focusedIdx.value = Math.max(focusedIdx.value - 1, 0)
    scrollFocused()
    return
  }

  // Enter — open dir or preview file
  if (e.key === 'Enter' && !inInput && focusedIdx.value >= 0) {
    const entry = filteredEntries.value[focusedIdx.value]
    if (!entry) return
    if (entry.type === 'dir') navigateTo(entry.name)
    else openPreview(entry)
    return
  }

  // d — download focused file
  if ((e.key === 'd' || e.key === 'D') && !inInput && !e.metaKey && !e.ctrlKey && focusedIdx.value >= 0) {
    const entry = filteredEntries.value[focusedIdx.value]
    if (entry?.type === 'file') download(entry)
    return
  }

  // Delete — delete focused entry
  if (e.key === 'Delete' && !inInput && focusedIdx.value >= 0) {
    const entry = filteredEntries.value[focusedIdx.value]
    if (entry?.type === 'file') confirmDelete(entry)
    else if (entry?.type === 'dir') confirmDeleteFolder(entry)
    return
  }

  // Space — quick-look toggle
  if (e.key === ' ' && !inInput && focusedIdx.value >= 0) {
    e.preventDefault()
    const entry = filteredEntries.value[focusedIdx.value]
    if (!entry || entry.type === 'dir') return
    if (previewEntry.value?.name === entry.name) closePreview()
    else openPreview(entry)
    return
  }
}

// ── Watchers / lifecycle ─────────────────────────────────────────
watch([() => props.conn, () => props.startPrefix], ([newConn, newPrefix], [oldConn]) => {
  if (newConn !== oldConn) {
    currentPrefix.value     = newPrefix || ''
    searchQuery.value       = ''
    selected.value          = new Set()
    focusedIdx.value        = -1
    stats.value             = null
    statsLoaded.value       = false
    previewEntry.value      = null
    metaEntry.value         = null
    showTransferModal.value = false
    showPresignModal.value  = false
    showCliModal.value      = false
    uploadQueue.value       = []
    load()
  } else if (newPrefix !== undefined && newPrefix !== currentPrefix.value) {
    navigateTo(newPrefix || '')
  }
})

watch(sentinel, val => {
  if (observer) observer.disconnect()
  if (val && nextPageToken.value) {
    observer = new IntersectionObserver(es => { if (es[0].isIntersecting) loadMore() }, { root: bodyEl.value, threshold: 0.1 })
    observer.observe(val)
  }
})

onMounted(() => {
  currentPrefix.value = props.startPrefix || ''
  load()
  window.addEventListener('keydown', onKeyDown)
})
onUnmounted(() => { window.removeEventListener('keydown', onKeyDown); observer?.disconnect() })

// ── Formatters ──────────────────────────────────────────────────
function formatSize(bytes) {
  if (!bytes && bytes !== 0) return '—'
  if (bytes === 0) return '0 B'
  const u = ['B','KB','MB','GB','TB']
  const i = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), u.length - 1)
  return (bytes / Math.pow(1024, i)).toFixed(i === 0 ? 0 : 1) + ' ' + u[i]
}
function formatDate(iso) {
  if (!iso) return '—'
  try { return new Date(iso).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' }) }
  catch { return '—' }
}
</script>
