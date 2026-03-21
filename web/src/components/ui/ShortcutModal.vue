<script setup>
import { onMounted, onUnmounted } from 'vue'

defineProps({
  open: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(['close'])

function handleEscape(e) {
  if (e.key === 'Escape') {
    emit('close')
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleEscape)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleEscape)
})
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="shortcut-overlay" @click.self="emit('close')">
      <div class="shortcut-modal">
        <div class="shortcut-modal__hd">
          <h3>Keyboard Shortcuts</h3>
          <button class="shortcut-modal__close" @click="emit('close')" aria-label="Close">&times;</button>
        </div>
        <div class="shortcut-modal__body">
          <div class="shortcut-group">
            <h4>Navigation</h4>
            <div class="shortcut-row"><kbd>N</kbd><span>New connection</span></div>
            <div class="shortcut-row"><kbd>/</kbd><span>Focus search</span></div>
            <div class="shortcut-row"><kbd>J</kbd><span>Move down</span></div>
            <div class="shortcut-row"><kbd>K</kbd><span>Move up</span></div>
            <div class="shortcut-row"><kbd>Enter</kbd><span>Open directory or preview</span></div>
            <div class="shortcut-row"><kbd>Backspace</kbd><span>Go up one level</span></div>
            <div class="shortcut-row"><kbd>Escape</kbd><span>Close preview / modal</span></div>
          </div>
          <div class="shortcut-group">
            <h4>File Actions</h4>
            <div class="shortcut-row"><kbd>Space</kbd><span>Toggle file preview</span></div>
            <div class="shortcut-row"><kbd>D</kbd><span>Download selected file</span></div>
            <div class="shortcut-row"><kbd>Delete</kbd><span>Delete selected file</span></div>
            <div class="shortcut-row"><kbd>R</kbd><span>Refresh file list</span></div>
          </div>
          <div class="shortcut-group">
            <h4>Global</h4>
            <div class="shortcut-row"><kbd>?</kbd><span>Show this help</span></div>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.shortcut-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
}

.shortcut-modal {
  background: var(--n-color-modal, #fff);
  border-radius: 8px;
  box-shadow: 0 4px 24px rgba(0, 0, 0, 0.15);
  max-width: 420px;
  width: 90%;
  max-height: 85vh;
  overflow: hidden;
}

.shortcut-modal__hd {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--n-border-color, #e0e0e0);
}

.shortcut-modal__hd h3 {
  margin: 0;
  font-size: 1.125rem;
  font-weight: 600;
}

.shortcut-modal__close {
  background: none;
  border: none;
  font-size: 1.5rem;
  line-height: 1;
  cursor: pointer;
  padding: 0 4px;
  opacity: 0.7;
}

.shortcut-modal__close:hover {
  opacity: 1;
}

.shortcut-modal__body {
  padding: 16px 20px;
  overflow-y: auto;
  max-height: calc(85vh - 60px);
}

.shortcut-group {
  margin-bottom: 16px;
}

.shortcut-group:last-child {
  margin-bottom: 0;
}

.shortcut-group h4 {
  margin: 0 0 8px 0;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--n-text-color-3, #999);
}

.shortcut-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 6px 0;
}

.shortcut-row kbd {
  display: inline-block;
  min-width: 28px;
  padding: 4px 8px;
  font-family: inherit;
  font-size: 0.8125rem;
  text-align: center;
  background: var(--n-color, #f0f0f0);
  border: 1px solid var(--n-border-color, #ddd);
  border-radius: 4px;
  box-shadow: 0 1px 0 rgba(0, 0, 0, 0.05);
}

.shortcut-row span {
  font-size: 0.875rem;
  color: var(--n-text-color-2, #333);
}
</style>
