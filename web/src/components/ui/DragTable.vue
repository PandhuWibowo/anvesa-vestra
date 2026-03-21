<template>
  <div class="dtable-wrap" ref="wrapEl">
    <!-- Column visibility dropdown -->
    <div v-if="columnToggle" class="dtable-toolbar">
      <button class="dtable-col-toggle" @click="colMenuOpen = !colMenuOpen" title="Toggle columns">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 3h7a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-7m0-18H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h7m0-18v18"/></svg>
        Columns
      </button>
      <transition name="dtable-dropdown">
        <div v-if="colMenuOpen" class="dtable-col-menu" @click.stop>
          <label v-for="col in allColumns" :key="col.key" class="dtable-col-menu__item">
            <input type="checkbox" :checked="!hiddenCols.has(col.key)" @change="toggleColumn(col.key)" />
            {{ col.label }}
          </label>
        </div>
      </transition>
    </div>

    <div class="dtable-scroll">
      <table class="dtable" :class="{ 'dtable--striped': striped }">
        <thead>
          <tr>
            <th v-if="draggableRows" class="dtable__grip-col"></th>
            <th
              v-for="(col, ci) in visibleColumns"
              :key="col.key"
              class="dtable__th"
              :class="{
                'dtable__th--sortable': col.sortable,
                'dtable__th--sorted': sortKey === col.key,
                'dtable__th--dragging': dragColIdx === ci,
              }"
              :style="colStyle(col)"
              :draggable="reorderableColumns"
              @dragstart.stop="onColDragStart($event, ci)"
              @dragover.prevent="onColDragOver($event, ci)"
              @drop.prevent="onColDrop($event, ci)"
              @dragend="onColDragEnd"
              @click="col.sortable ? cycleSort(col.key) : null"
            >
              <span class="dtable__th-label">
                <slot :name="'header-' + col.key" :column="col">{{ col.label }}</slot>
              </span>
              <span v-if="col.sortable" class="dtable__sort-icon">
                <svg v-if="sortKey === col.key && sortDir === 'asc'" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><polyline points="18 15 12 9 6 15"/></svg>
                <svg v-else-if="sortKey === col.key && sortDir === 'desc'" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><polyline points="6 9 12 15 18 9"/></svg>
                <svg v-else width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="opacity:.3"><polyline points="18 15 12 9 6 15"/></svg>
              </span>
              <!-- Resize handle -->
              <span
                v-if="resizableColumns && ci < visibleColumns.length - 1"
                class="dtable__resize"
                @mousedown.stop.prevent="onResizeStart($event, ci)"
              ></span>
            </th>
          </tr>
        </thead>
        <tbody>
          <template v-if="sortedRows.length">
            <tr
              v-for="(row, ri) in sortedRows"
              :key="getRowKey(row, ri)"
              class="dtable__tr"
              :class="{
                'dtable__tr--dragging': dragRowIdx === ri,
                'dtable__tr--drop-above': dropRowIdx === ri && dropPos === 'above',
                'dtable__tr--drop-below': dropRowIdx === ri && dropPos === 'below',
              }"
              :draggable="draggableRows"
              @dragstart="draggableRows ? onRowDragStart($event, ri) : null"
              @dragover.prevent="draggableRows ? onRowDragOver($event, ri) : null"
              @drop.prevent="draggableRows ? onRowDrop($event, ri) : null"
              @dragend="draggableRows ? onRowDragEnd() : null"
              @click="$emit('row-click', row, ri)"
            >
              <td v-if="draggableRows" class="dtable__grip">
                <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="9" cy="5" r="1"/><circle cx="9" cy="12" r="1"/><circle cx="9" cy="19" r="1"/><circle cx="15" cy="5" r="1"/><circle cx="15" cy="12" r="1"/><circle cx="15" cy="19" r="1"/></svg>
              </td>
              <td
                v-for="col in visibleColumns"
                :key="col.key"
                class="dtable__td"
                :class="[col.cellClass, col.align ? `dtable__td--${col.align}` : '']"
                :style="colStyle(col)"
              >
                <slot :name="'cell-' + col.key" :row="row" :value="row[col.key]" :index="ri">
                  {{ row[col.key] ?? '—' }}
                </slot>
              </td>
            </tr>
          </template>
          <tr v-else>
            <td :colspan="visibleColumns.length + (draggableRows ? 1 : 0)" class="dtable__empty">
              <slot name="empty">No data</slot>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'

const props = defineProps({
  columns:            { type: Array, required: true },
  rows:               { type: Array, default: () => [] },
  rowKey:             { type: [String, Function], default: null },
  draggableRows:      { type: Boolean, default: false },
  resizableColumns:   { type: Boolean, default: true },
  reorderableColumns: { type: Boolean, default: true },
  columnToggle:       { type: Boolean, default: false },
  striped:            { type: Boolean, default: false },
  defaultSort:        { type: Object, default: null },
})

const emit = defineEmits(['sort', 'row-reorder', 'column-reorder', 'row-click'])

const wrapEl = ref(null)
const allColumns = computed(() => props.columns)
const hiddenCols = ref(new Set())
const colMenuOpen = ref(false)
const colWidths = ref({})

const columnOrder = ref(null)
const visibleColumns = computed(() => {
  let cols = columnOrder.value
    ? columnOrder.value.map(k => allColumns.value.find(c => c.key === k)).filter(Boolean)
    : [...allColumns.value]
  return cols.filter(c => !hiddenCols.value.has(c.key))
})

function toggleColumn(key) {
  const next = new Set(hiddenCols.value)
  next.has(key) ? next.delete(key) : next.add(key)
  if (next.size >= allColumns.value.length) return
  hiddenCols.value = next
}

function getRowKey(row, idx) {
  if (!props.rowKey) return idx
  if (typeof props.rowKey === 'function') return props.rowKey(row)
  return row[props.rowKey] ?? idx
}

// ── Sorting ──
const sortKey = ref(props.defaultSort?.key || '')
const sortDir = ref(props.defaultSort?.dir || '')

function cycleSort(key) {
  if (sortKey.value !== key) {
    sortKey.value = key
    sortDir.value = 'asc'
  } else if (sortDir.value === 'asc') {
    sortDir.value = 'desc'
  } else {
    sortKey.value = ''
    sortDir.value = ''
  }
  emit('sort', { key: sortKey.value, dir: sortDir.value })
}

const sortedRows = computed(() => {
  if (!sortKey.value || !sortDir.value) return props.rows
  const k = sortKey.value
  const dir = sortDir.value === 'asc' ? 1 : -1
  return [...props.rows].sort((a, b) => {
    const va = a[k], vb = b[k]
    if (va == null && vb == null) return 0
    if (va == null) return 1
    if (vb == null) return -1
    if (typeof va === 'number' && typeof vb === 'number') return (va - vb) * dir
    return String(va).localeCompare(String(vb)) * dir
  })
})

// ── Row drag-and-drop ──
const dragRowIdx = ref(null)
const dropRowIdx = ref(null)
const dropPos = ref(null)

function onRowDragStart(e, idx) {
  dragRowIdx.value = idx
  e.dataTransfer.effectAllowed = 'move'
  e.dataTransfer.setData('text/plain', String(idx))
}
function onRowDragOver(e, idx) {
  if (dragRowIdx.value === null) return
  dropRowIdx.value = idx
  const rect = e.currentTarget.getBoundingClientRect()
  dropPos.value = e.clientY < rect.top + rect.height / 2 ? 'above' : 'below'
}
function onRowDrop(e, idx) {
  if (dragRowIdx.value === null) return
  const from = dragRowIdx.value
  let to = idx
  if (dropPos.value === 'below') to += 1
  if (from !== to && from !== to - 1) {
    const newRows = [...props.rows]
    const [item] = newRows.splice(from, 1)
    const insertAt = from < to ? to - 1 : to
    newRows.splice(insertAt, 0, item)
    emit('row-reorder', { from, to: insertAt, rows: newRows })
  }
  onRowDragEnd()
}
function onRowDragEnd() {
  dragRowIdx.value = null
  dropRowIdx.value = null
  dropPos.value = null
}

// ── Column drag-and-drop ──
const dragColIdx = ref(null)

function onColDragStart(e, idx) {
  if (!props.reorderableColumns) return
  dragColIdx.value = idx
  e.dataTransfer.effectAllowed = 'move'
  e.dataTransfer.setData('text/plain', String(idx))
}
function onColDragOver(e, idx) {
  if (dragColIdx.value === null || !props.reorderableColumns) return
}
function onColDrop(e, idx) {
  if (dragColIdx.value === null || !props.reorderableColumns) return
  const from = dragColIdx.value
  if (from !== idx) {
    const order = visibleColumns.value.map(c => c.key)
    const [key] = order.splice(from, 1)
    order.splice(idx, 0, key)
    const hidden = allColumns.value.filter(c => hiddenCols.value.has(c.key)).map(c => c.key)
    columnOrder.value = [...order, ...hidden]
    emit('column-reorder', { from, to: idx, columns: order })
  }
  onColDragEnd()
}
function onColDragEnd() {
  dragColIdx.value = null
}

// ── Column resize ──
let resizeColIdx = null
let resizeStartX = 0
let resizeStartW = 0

function onResizeStart(e, idx) {
  resizeColIdx = idx
  resizeStartX = e.clientX
  const col = visibleColumns.value[idx]
  const th = wrapEl.value?.querySelectorAll('.dtable__th')[props.draggableRows ? idx + 1 : idx]
  resizeStartW = th?.offsetWidth || col.width || 120
  document.addEventListener('mousemove', onResizeMove)
  document.addEventListener('mouseup', onResizeEnd)
  document.body.style.cursor = 'col-resize'
  document.body.style.userSelect = 'none'
}
function onResizeMove(e) {
  if (resizeColIdx === null) return
  const diff = e.clientX - resizeStartX
  const col = visibleColumns.value[resizeColIdx]
  const minW = col.minWidth || 60
  colWidths.value = { ...colWidths.value, [col.key]: Math.max(minW, resizeStartW + diff) }
}
function onResizeEnd() {
  resizeColIdx = null
  document.removeEventListener('mousemove', onResizeMove)
  document.removeEventListener('mouseup', onResizeEnd)
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
}

function colStyle(col) {
  const w = colWidths.value[col.key] || col.width
  if (!w) return {}
  const px = typeof w === 'number' ? w + 'px' : w
  return { width: px, minWidth: px, maxWidth: px }
}

onUnmounted(() => {
  document.removeEventListener('mousemove', onResizeMove)
  document.removeEventListener('mouseup', onResizeEnd)
})
</script>
