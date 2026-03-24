<template>
  <div class="view-panel">
    <div class="view-panel__hd">
      <div>
        <h2 class="view-panel__title">Two-Factor Authentication</h2>
        <p class="view-panel__sub">Protect your account with TOTP-based 2FA</p>
      </div>
    </div>

    <div class="view-panel__body">

      <!-- Status card -->
      <div class="dash-section">
        <div class="tfa-status-row">
          <div class="tfa-status-info">
            <div class="tfa-status-badge" :class="totpEnabled ? 'tfa-status-badge--on' : 'tfa-status-badge--off'">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                <path v-if="totpEnabled" d="M20 6L9 17l-5-5"/>
                <path v-else d="M18 6L6 18M6 6l12 12"/>
              </svg>
              {{ totpEnabled ? 'Enabled' : 'Disabled' }}
            </div>
            <p style="font-size:13px;color:var(--text-2);margin-top:8px">
              {{ totpEnabled
                ? 'Your account is protected with two-factor authentication.'
                : 'Add an extra layer of security to your account.' }}
            </p>
          </div>
          <button
            v-if="!totpEnabled && !setupMode"
            class="base-btn base-btn--primary"
            @click="startSetup"
            :disabled="loading"
            style="font-size:13px;padding:8px 16px;flex-shrink:0"
          >
            Enable 2FA
          </button>
          <button
            v-if="totpEnabled && !disableMode"
            class="base-btn base-btn--danger"
            @click="disableMode = true"
            style="font-size:13px;padding:8px 16px;flex-shrink:0;border:1px solid var(--danger)"
          >
            Disable 2FA
          </button>
        </div>
      </div>

      <!-- Setup flow -->
      <div v-if="setupMode && !totpEnabled" class="dash-section">
        <h3 class="dash-section__title">Setup Authenticator</h3>

        <div v-if="setupStep === 'scan'" class="tfa-setup">
          <p style="font-size:13px;color:var(--text-2);margin-bottom:16px">
            Scan the QR code below with your authenticator app (e.g. Google Authenticator, Authy), then enter the 6-digit code to confirm.
          </p>

          <!-- QR code placeholder using otpauth URL -->
          <div class="tfa-qr-area">
            <img v-if="qrDataUrl" :src="qrDataUrl" alt="QR Code" class="tfa-qr" />
            <div v-else class="tfa-qr-placeholder">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" style="opacity:.3">
                <rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/>
                <rect x="3" y="14" width="7" height="7"/><rect x="14" y="14" width="3" height="3"/>
              </svg>
              Loading…
            </div>
          </div>

          <div class="tfa-manual-key">
            <span class="tfa-manual-label">Manual entry key:</span>
            <code class="tfa-key-code">{{ secret }}</code>
            <button class="row-btn" @click="copySecret" title="Copy">
              <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
              </svg>
            </button>
          </div>

          <div style="margin-top:20px">
            <label style="font-size:12px;font-weight:600;color:var(--text-2);display:block;margin-bottom:6px">Verification Code</label>
            <div style="display:flex;gap:8px;align-items:center">
              <input
                ref="verifyInput"
                v-model="verifyCode"
                class="base-input"
                type="text"
                inputmode="numeric"
                maxlength="6"
                placeholder="000000"
                style="width:140px;font-size:18px;letter-spacing:0.2em;text-align:center"
                @keydown.enter="verifySetup"
              />
              <button
                class="base-btn base-btn--primary"
                @click="verifySetup"
                :disabled="verifyCode.length !== 6 || verifying"
                style="font-size:13px;padding:8px 16px"
              >
                {{ verifying ? 'Verifying…' : 'Confirm' }}
              </button>
            </div>
            <p v-if="verifyError" class="tfa-error">{{ verifyError }}</p>
          </div>

          <button class="base-btn base-btn--ghost" @click="cancelSetup" style="font-size:12px;margin-top:12px">
            Cancel
          </button>
        </div>
      </div>

      <!-- Disable flow -->
      <div v-if="disableMode && totpEnabled" class="dash-section">
        <h3 class="dash-section__title">Disable Two-Factor Authentication</h3>
        <p style="font-size:13px;color:var(--text-2);margin-bottom:16px">
          Enter your current authenticator code to disable 2FA.
        </p>
        <div style="display:flex;gap:8px;align-items:center">
          <input
            v-model="disableCode"
            class="base-input"
            type="text"
            inputmode="numeric"
            maxlength="6"
            placeholder="000000"
            style="width:140px;font-size:18px;letter-spacing:0.2em;text-align:center"
            @keydown.enter="confirmDisable"
          />
          <button
            class="base-btn base-btn--danger"
            @click="confirmDisable"
            :disabled="disableCode.length !== 6 || disabling"
            style="font-size:13px;padding:8px 16px;border:1px solid var(--danger)"
          >
            {{ disabling ? 'Disabling…' : 'Disable 2FA' }}
          </button>
          <button class="base-btn base-btn--ghost" @click="disableMode = false; disableCode = ''" style="font-size:13px">
            Cancel
          </button>
        </div>
        <p v-if="disableError" class="tfa-error">{{ disableError }}</p>
      </div>

    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, nextTick } from 'vue'
import { useAuth } from '../../composables/useAuth.js'
import { useToast } from '../../composables/useToast.js'

const { authHeaders } = useAuth()
const toast = useToast()

const totpEnabled = ref(false)
const loading = ref(false)
const setupMode = ref(false)
const setupStep = ref('scan')
const secret = ref('')
const otpauthUrl = ref('')
const qrDataUrl = ref('')
const verifyCode = ref('')
const verifyError = ref('')
const verifying = ref(false)
const verifyInput = ref(null)
const disableMode = ref(false)
const disableCode = ref('')
const disableError = ref('')
const disabling = ref(false)

onMounted(fetchStatus)

async function fetchStatus() {
  loading.value = true
  try {
    const res = await fetch('/api/auth/2fa/status', { headers: authHeaders() })
    if (res.ok) {
      const data = await res.json()
      totpEnabled.value = data.enabled
    }
  } finally {
    loading.value = false
  }
}

async function startSetup() {
  loading.value = true
  verifyError.value = ''
  try {
    const res = await fetch('/api/auth/2fa/setup', {
      method: 'POST',
      headers: authHeaders(),
    })
    if (!res.ok) throw new Error(await res.text())
    const data = await res.json()
    secret.value = data.secret
    otpauthUrl.value = data.otpauth
    setupMode.value = true
    setupStep.value = 'scan'
    // Generate QR code using a simple canvas-based approach
    await generateQR(data.otpauth)
    await nextTick()
    verifyInput.value?.focus()
  } catch (err) {
    toast.error('Setup failed: ' + err.message)
  } finally {
    loading.value = false
  }
}

async function generateQR(url) {
  // Use qrcode-generator or a simple SVG approach via a public API
  // For simplicity, we'll use a data URI approach
  try {
    const response = await fetch(
      `https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(url)}`
    )
    if (response.ok) {
      const blob = await response.blob()
      qrDataUrl.value = URL.createObjectURL(blob)
    }
  } catch {
    // Fallback: show manual key only
    qrDataUrl.value = ''
  }
}

function copySecret() {
  navigator.clipboard?.writeText(secret.value).then(
    () => toast.success('Key copied!'),
    () => toast.error('Clipboard unavailable'),
  )
}

async function verifySetup() {
  if (verifyCode.value.length !== 6 || verifying.value) return
  verifyError.value = ''
  verifying.value = true
  try {
    const res = await fetch('/api/auth/2fa/verify', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body: JSON.stringify({ code: verifyCode.value }),
    })
    if (!res.ok) {
      const err = await res.json().catch(() => ({}))
      verifyError.value = err.error || 'Invalid code. Please try again.'
      verifyCode.value = ''
      verifyInput.value?.focus()
      return
    }
    totpEnabled.value = true
    setupMode.value = false
    toast.success('Two-factor authentication enabled!')
  } catch (err) {
    verifyError.value = 'Verification failed: ' + err.message
  } finally {
    verifying.value = false
  }
}

function cancelSetup() {
  setupMode.value = false
  verifyCode.value = ''
  verifyError.value = ''
  secret.value = ''
  qrDataUrl.value = ''
}

async function confirmDisable() {
  if (disableCode.value.length !== 6 || disabling.value) return
  disableError.value = ''
  disabling.value = true
  try {
    const res = await fetch('/api/auth/2fa/disable', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body: JSON.stringify({ code: disableCode.value }),
    })
    if (!res.ok) {
      const err = await res.json().catch(() => ({}))
      disableError.value = err.error || 'Invalid code.'
      disableCode.value = ''
      return
    }
    totpEnabled.value = false
    disableMode.value = false
    toast.success('Two-factor authentication disabled.')
  } catch (err) {
    disableError.value = 'Failed: ' + err.message
  } finally {
    disabling.value = false
  }
}
</script>

<style scoped>
.tfa-status-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.tfa-status-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 600;
}

.tfa-status-badge--on {
  background: rgba(34, 197, 94, 0.12);
  color: #22c55e;
  border: 1px solid rgba(34, 197, 94, 0.25);
}

.tfa-status-badge--off {
  background: var(--surface-2);
  color: var(--muted);
  border: 1px solid var(--border);
}

.tfa-setup {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  max-width: 440px;
}

.tfa-qr-area {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 200px;
  height: 200px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: #fff;
  margin-bottom: 16px;
}

.tfa-qr {
  width: 200px;
  height: 200px;
  border-radius: 8px;
}

.tfa-qr-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  color: var(--muted);
  font-size: 12px;
}

.tfa-manual-key {
  display: flex;
  align-items: center;
  gap: 8px;
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 8px 12px;
  font-size: 12px;
}

.tfa-manual-label {
  color: var(--muted);
  white-space: nowrap;
}

.tfa-key-code {
  font-family: var(--mono);
  font-size: 12px;
  color: var(--text);
  letter-spacing: 0.05em;
  word-break: break-all;
}

.tfa-error {
  margin: 8px 0 0;
  font-size: 12px;
  color: var(--danger, #e55);
  background: rgba(220, 50, 50, 0.08);
  border-radius: 6px;
  padding: 6px 10px;
}
</style>
