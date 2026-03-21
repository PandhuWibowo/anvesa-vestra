<template>
  <div class="auth-screen">
    <div class="auth-card">
      <div class="auth-brand">
        <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <path d="M3 15a4 4 0 0 0 4 4h9a5 5 0 0 0 1.8-9.7 6 6 0 0 0-11.8-1A4 4 0 0 0 3 15z"/>
        </svg>
        <div>
          <div class="auth-brand__name"><span class="auth-brand__anvesa">Anveesa</span> Vestra</div>
          <div class="auth-brand__sub">Cloud storage manager</div>
        </div>
      </div>

      <h2 class="auth-title">{{ setupRequired ? 'Create Admin Account' : 'Sign In' }}</h2>
      <p class="auth-sub">
        {{ setupRequired
          ? 'Set up your admin account to get started.'
          : 'Enter your credentials to continue.' }}
      </p>

      <form @submit.prevent="handleSubmit" class="auth-form">
        <div class="auth-field">
          <label class="auth-label" for="auth-user">Username</label>
          <input
            id="auth-user"
            class="auth-input"
            v-model="username"
            type="text"
            autocomplete="username"
            placeholder="admin"
            required
            :minlength="setupRequired ? 3 : 1"
          />
        </div>
        <div class="auth-field">
          <label class="auth-label" for="auth-pass">Password</label>
          <input
            id="auth-pass"
            class="auth-input"
            v-model="password"
            type="password"
            autocomplete="current-password"
            placeholder="••••••••"
            required
            :minlength="setupRequired ? 8 : 1"
          />
        </div>
        <div v-if="setupRequired" class="auth-field">
          <label class="auth-label" for="auth-pass2">Confirm Password</label>
          <input
            id="auth-pass2"
            class="auth-input"
            v-model="confirmPassword"
            type="password"
            autocomplete="new-password"
            placeholder="••••••••"
            required
            minlength="8"
          />
        </div>

        <p v-if="authError" class="auth-error">{{ authError }}</p>
        <p v-if="localError" class="auth-error">{{ localError }}</p>

        <button class="auth-submit" type="submit" :disabled="authLoading">
          {{ authLoading ? 'Please wait…' : (setupRequired ? 'Create Account' : 'Sign In') }}
        </button>
      </form>

      <!-- OAuth buttons (only on login, not setup) -->
      <template v-if="!setupRequired && oauthProviders">
        <div v-if="oauthProviders.google?.enabled || oauthProviders.github?.enabled" class="auth-divider">
          <span>or continue with</span>
        </div>
        <div class="auth-oauth">
          <button v-if="oauthProviders.google?.enabled" class="auth-oauth-btn auth-oauth-btn--google" @click="loginWithGoogle">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 01-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z" fill="#4285F4"/><path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853"/><path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05"/><path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335"/></svg>
            Google
          </button>
          <button v-if="oauthProviders.github?.enabled" class="auth-oauth-btn auth-oauth-btn--github" @click="loginWithGitHub">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M12 .3a12 12 0 00-3.8 23.38c.6.12.83-.26.83-.57L9 21.07c-3.34.72-4.04-1.61-4.04-1.61-.55-1.39-1.34-1.76-1.34-1.76-1.08-.74.08-.73.08-.73 1.2.09 1.84 1.24 1.84 1.24 1.07 1.83 2.8 1.3 3.49 1 .1-.78.42-1.3.76-1.6-2.67-.31-5.47-1.34-5.47-5.93 0-1.31.47-2.38 1.24-3.22-.14-.3-.54-1.52.1-3.18 0 0 1-.32 3.3 1.23a11.5 11.5 0 016.02 0c2.28-1.55 3.29-1.23 3.29-1.23.64 1.66.24 2.88.12 3.18a4.65 4.65 0 011.23 3.22c0 4.61-2.8 5.62-5.48 5.92.42.36.81 1.1.81 2.22l-.01 3.29c0 .31.2.69.82.57A12 12 0 0012 .3"/></svg>
            GitHub
          </button>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useAuth } from '../../composables/useAuth.js'

const { setupRequired, authLoading, authError, register, login } = useAuth()

const oauthProviders = ref(null)

onMounted(async () => {
  try {
    const res = await fetch('/api/auth/oauth/config')
    if (res.ok) oauthProviders.value = await res.json()
  } catch { /* OAuth not available */ }
})

function loginWithGoogle() {
  const cfg = oauthProviders.value?.google
  if (!cfg?.client_id) return
  const params = new URLSearchParams({
    client_id: cfg.client_id,
    redirect_uri: window.location.origin + '/api/auth/oauth/google/callback',
    response_type: 'code',
    scope: 'openid email profile',
    access_type: 'offline',
  })
  window.location.href = 'https://accounts.google.com/o/oauth2/v2/auth?' + params
}

function loginWithGitHub() {
  const cfg = oauthProviders.value?.github
  if (!cfg?.client_id) return
  const params = new URLSearchParams({
    client_id: cfg.client_id,
    redirect_uri: window.location.origin + '/api/auth/oauth/github/callback',
    scope: 'read:user user:email',
  })
  window.location.href = 'https://github.com/login/oauth/authorize?' + params
}

const username = ref('')
const password = ref('')
const confirmPassword = ref('')
const localError = ref('')

async function handleSubmit() {
  localError.value = ''
  if (setupRequired.value) {
    if (password.value !== confirmPassword.value) {
      localError.value = 'Passwords do not match'
      return
    }
    if (password.value.length < 8) {
      localError.value = 'Password must be at least 8 characters'
      return
    }
    await register(username.value, password.value)
  } else {
    await login(username.value, password.value)
  }
}
</script>

<style scoped>
.auth-screen {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background: var(--bg);
  padding: 24px;
}

.auth-card {
  width: 100%;
  max-width: 380px;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 36px 32px 32px;
}

.auth-brand {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 28px;
  color: var(--text);
}

.auth-brand__name {
  font-size: 15px;
  font-weight: 600;
  letter-spacing: -0.02em;
}

.auth-brand__anvesa {
  color: var(--accent);
}

.auth-brand__sub {
  font-size: 11px;
  color: var(--muted);
  margin-top: 1px;
}

.auth-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--text);
  margin: 0 0 6px;
}

.auth-sub {
  font-size: 13px;
  color: var(--muted);
  margin: 0 0 24px;
}

.auth-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.auth-field {
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.auth-label {
  font-size: 12px;
  font-weight: 500;
  color: var(--text);
}

.auth-input {
  padding: 9px 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--bg);
  color: var(--text);
  font-size: 13px;
  outline: none;
  transition: border-color 0.15s;
}

.auth-input:focus {
  border-color: var(--accent);
}

.auth-input::placeholder {
  color: var(--muted);
  opacity: 0.6;
}

.auth-error {
  margin: 0;
  font-size: 12px;
  color: var(--danger, #e55);
  background: rgba(220, 50, 50, 0.08);
  border-radius: 6px;
  padding: 8px 10px;
}

.auth-submit {
  padding: 10px 0;
  border: none;
  border-radius: 8px;
  background: var(--accent);
  color: #fff;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: opacity 0.15s;
}

.auth-submit:hover:not(:disabled) {
  opacity: 0.9;
}

.auth-submit:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.auth-divider {
  display: flex;
  align-items: center;
  gap: 12px;
  margin: 20px 0 16px;
  color: var(--muted);
  font-size: 11px;
}
.auth-divider::before,
.auth-divider::after {
  content: '';
  flex: 1;
  height: 1px;
  background: var(--border);
}

.auth-oauth {
  display: flex;
  gap: 8px;
}
.auth-oauth-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 9px 0;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface);
  color: var(--text);
  font-size: 12.5px;
  font-weight: 500;
  cursor: pointer;
  transition: border-color .15s, background .15s;
}
.auth-oauth-btn:hover {
  border-color: var(--border-2);
  background: var(--surface-2);
}
.auth-oauth-btn--github svg { color: var(--text); }
</style>
